// Copyright © 2019-2020 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpmod

import (
	"crypto"
	"errors"
	"fmt"
	mathbig "math/big"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/common/hash"
	big "github.com/binance-chain/tss-lib/common/int"
	int2 "github.com/binance-chain/tss-lib/common/int"
)

const (
	Iterations         = 13
	ProofModBytesParts = Iterations*4 + 1
	MinBitLen          = 254
	DST                = "TSS-LIB-ZKP-MOD-DST"
)

var (
	one = big.NewInt(1)
)

type (
	ProofMod struct {
		W *big.Int
		X [Iterations]*big.Int
		A [Iterations]*big.Int
		B [Iterations]*big.Int
		Z [Iterations]*big.Int
	}
)

// isQuadraticResidue checks Euler criterion
func isQuadraticResidue(X, N *big.Int) bool {
	modN := int2.ModInt(N)
	XEXP := modN.Exp(X, new(big.Int).Rsh(N, 1))
	ok := XEXP.Cmp(big.NewInt(1)) == 0
	return ok
}

func NewProof(q, N, P, Q, nonce *big.Int) (*ProofMod, error) {
	if N == nil || P == nil || Q == nil || nonce == nil || big.NewInt(0).Cmp(nonce) == 0 {
		return nil, errors.New("nil value(s)")
	}
	if nonce.BitLen() < MinBitLen {
		return nil, errors.New("invalid nonce")
	}
	Phi := new(big.Int).Mul(new(big.Int).Sub(P, one), new(big.Int).Sub(Q, one))
	// Fig 16.1
	W := common.GetRandomQuadraticNonResidue(N)

	// Fig 16.2
	Y := [Iterations]*big.Int{}
	for i := range Y {
		ei := hash.SHA256i(append([]*big.Int{nonce, W, N}, Y[:i]...)...)
		expanded := hash.ExpandXMD(crypto.SHA256, N.Bytes(), ei.Bytes(), len(q.Bytes())+16)
		expandedI := new(big.Int).SetBytes(expanded)
		expandedI = expandedI.Mod(expandedI, q)
		Y[i] = expandedI
	}
	X, A, B, Z := proof(N, P, Q, Phi, Y, W)
	return &ProofMod{W: W, X: X, A: A, B: B, Z: Z}, nil
}

func proof(N *big.Int, P *big.Int, Q *big.Int, Phi *big.Int, Y [13]*big.Int, W *big.Int) ([13]*big.Int, [13]*big.Int, [13]*big.Int, [13]*big.Int) {
	// Fig 16.3
	modN, modPhi := int2.ModInt(N), int2.ModInt(Phi)
	NINV := new(big.Int).ModInverse(N, Phi)
	X := [Iterations]*big.Int{}
	A := [Iterations]*big.Int{}
	B := [Iterations]*big.Int{}
	Z := [Iterations]*big.Int{}

	for i := range Y {
		for j := 0; j < 4; j++ {
			a, b := j&1, j&2>>1
			Yi := new(big.Int).SetBytes(Y[i].Bytes()) // TODO use bool instead
			if a > 0 {
				Yi = modN.Mul(new(big.Int).SetInt64(-1), Yi)
			}
			if b > 0 {
				Yi = modN.Mul(W, Yi)
			}
			if isQuadraticResidue(Yi, P) && isQuadraticResidue(Yi, Q) {
				e := new(big.Int).Add(Phi, big.NewInt(4))
				e = new(big.Int).Rsh(e, 3)
				e = modPhi.Mul(e, e)
				Xi := modN.Exp(Yi, e)
				Zi := modN.Exp(Y[i], NINV)
				X[i], A[i], B[i], Z[i] = Xi, big.NewInt(uint64(a)), big.NewInt(uint64(b)), Zi
			}
		}
	}
	return X, A, B, Z
}

func NewProofFromBytes(bzs [][]byte) (*ProofMod, error) {
	if !common.AnyNonEmptyMultiByte(bzs, ProofModBytesParts) {
		return nil, fmt.Errorf("expected %d byte parts to construct ProofMod", ProofModBytesParts)
	}
	bis := make([]*big.Int, len(bzs))
	for i := range bis {
		bis[i] = new(big.Int).SetBytes(bzs[i])
	}

	X := [Iterations]*big.Int{}
	copy(X[:], bis[1:(Iterations+1)])

	A := [Iterations]*big.Int{}
	copy(A[:], bis[(Iterations+1):(Iterations*2+1)])

	B := [Iterations]*big.Int{}
	copy(B[:], bis[(Iterations*2+1):(Iterations*3+1)])

	Z := [Iterations]*big.Int{}
	copy(Z[:], bis[(Iterations*3+1):])

	return &ProofMod{
		W: bis[0],
		X: X,
		A: A,
		B: B,
		Z: Z,
	}, nil
}

func (pf *ProofMod) Verify(q, N, nonce *big.Int) bool {
	if pf == nil || !pf.ValidateBasic() {
		return false
	}
	Y := [Iterations]*big.Int{}
	for i := range Y {
		ei := hash.SHA256i(append([]*big.Int{nonce, pf.W, N}, Y[:i]...)...)
		expanded := hash.ExpandXMD(crypto.SHA256, N.Bytes(), ei.Bytes(), len(q.Bytes())+16)
		expandedI := new(big.Int).SetBytes(expanded)
		expandedI = expandedI.Mod(expandedI, q)
		Y[i] = expandedI
	}

	b, done := verification(N, pf, Y)
	if done {
		return b
	}
	return true
}

func verification(N *big.Int, pf *ProofMod, Y [13]*big.Int) (bool, bool) {
	// Fig 16. Verification
	if N.Bit(0) == 0 || N.ProbablyPrime(16) {
		return false, true
	}

	ch := make(chan bool)
	for i := 0; i < Iterations; i++ {
		go func(i int) {
			modN := int2.ModInt(N)
			left := new(big.Int).Set(modN.Exp(pf.Z[i], N))
			if left.Cmp(Y[i]) != 0 {
				ch <- false
				return
			}
			ch <- true
		}(i)

		go func(i int) {
			modN := int2.ModInt(N)
			a, b := pf.A[i].Int64(), pf.B[i].Int64()
			left := modN.Exp(pf.X[i], big.NewInt(4))
			right := Y[i]
			if a > 0 {
				right = modN.Mul(new(big.Int).SetInt64(-1), right)
			}
			if b > 0 {
				right = modN.Mul(pf.W, right)
			}
			if left.Cmp(right) != 0 {
				ch <- false
				return
			}
			ch <- true
		}(i)
	}

	// drain the channel (goroutines) before returning
	fail := false
	for i := 0; i < Iterations*2; i++ {
		if !<-ch {
			fail = true
		}
	}
	if fail {
		return false, true
	}
	return true, true
}

func (pf *ProofMod) ValidateBasic() bool {
	if pf.W == nil {
		return false
	}
	for i := range pf.X {
		if pf.X[i] == nil {
			return false
		}
	}
	for i := range pf.A {
		if pf.A[i] == nil {
			return false
		}
	}
	for i := range pf.B {
		if pf.B[i] == nil {
			return false
		}
	}
	for i := range pf.Z {
		if pf.Z[i] == nil {
			return false
		}
	}
	return true
}

func (pf *ProofMod) Bytes() [ProofModBytesParts][]byte {
	bzs := [ProofModBytesParts][]byte{}
	bzs[0] = pf.W.Bytes()
	for i := range pf.X {
		bzs[1+i] = pf.X[i].Bytes()
	}
	for i := range pf.A {
		bzs[Iterations+1+i] = pf.A[i].Bytes()
	}
	for i := range pf.B {
		bzs[Iterations*2+1+i] = pf.B[i].Bytes()
	}
	for i := range pf.Z {
		bzs[Iterations*3+1+i] = pf.Z[i].Bytes()
	}
	return bzs
}

// GetRandomNonQuadraticNonResidue Not quadratic non residue.
func GetRandomNonQuadraticNonResidue(n *big.Int) *big.Int {
	for {
		w := common.GetRandomPositiveInt(n)
		if mathbig.Jacobi(w, n) != -1 {
			return w
		}
	}
}

func FormatProofMod(pf *ProofMod) string {
	return "(W:" + common.FormatBigInt(pf.W) + ", X:" + common.FormatBigInt(pf.X[0]) + "..." + common.FormatBigInt(pf.X[len(pf.X)-1]) +
		", A:" + common.FormatBigInt(pf.A[0]) + "..." + common.FormatBigInt(pf.A[len(pf.A)-1]) +
		", B:" + common.FormatBigInt(pf.B[0]) + "..." + common.FormatBigInt(pf.B[len(pf.B)-1]) +
		", Z:" + common.FormatBigInt(pf.Z[0]) + "..." + common.FormatBigInt(pf.Z[len(pf.Z)-1]) +
		")"
}
