// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpdec

import (
	"crypto/elliptic"
	"errors"
	"fmt"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/common/hash"
	big "github.com/binance-chain/tss-lib/common/int"
	int2 "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/crypto/zkp"
)

const (
	ProofDecBytesParts = 7
)

type (
	ProofDec struct {
		S, T, A, Gamma, Z1, Z2, W *big.Int
	}
)

// NewProof implements proofenc
func NewProof(ec elliptic.Curve, pk *paillier.PublicKey, C, x, NCap, s, t, y, rho *big.Int) (*ProofDec, error) {
	if ec == nil || pk == nil || C == nil || x == nil || NCap == nil || s == nil || t == nil || y == nil || rho == nil {
		return nil, errors.New("ProveDec constructor received nil value(s)")
	}
	q := big.Wrap(ec.Params().N)
	alpha, mu, v, r, S, T, A, gamma := initProof(ec, pk, NCap, s, y, t)

	// Fig 29.2 e
	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), C, x, NCap, s, t, A, gamma)...)
		e = hash.RejectionSample(q, eHash)
	}

	z1, z2, w := coda(e, y, alpha, mu, v, pk, rho, r)

	return &ProofDec{S: S, T: T, A: A, Gamma: gamma, Z1: z1, Z2: z2, W: w}, nil
}

// NewProof implements proofenc
func NewProofGivenNonce(ec elliptic.Curve, pk *paillier.PublicKey, C, x, NCap, s, t, y, rho, nonce *big.Int) (*ProofDec, error) {
	if ec == nil || pk == nil || C == nil || x == nil || NCap == nil || s == nil || t == nil || y == nil || rho == nil ||
		nonce == nil || big.NewInt(0).Cmp(nonce) == 0 {
		return nil, errors.New("ProveDec constructor received nil value(s)")
	}
	if nonce.BitLen() < ec.Params().N.BitLen()-1 {
		return nil, errors.New("invalid nonce")
	}
	q := big.Wrap(ec.Params().N)
	alpha, mu, v, r, S, T, A, gamma := initProof(ec, pk, NCap, s, y, t)

	// Fig 29.2 e
	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), C, x, NCap, s, t, A, gamma, nonce)...)
		e = hash.RejectionSample(q, eHash)
	}

	z1, z2, w := coda(e, y, alpha, mu, v, pk, rho, r)

	return &ProofDec{S: S, T: T, A: A, Gamma: gamma, Z1: z1, Z2: z2, W: w}, nil
}

func coda(e *int2.Int, y *int2.Int, alpha *int2.Int, mu *int2.Int, v *int2.Int, pk *paillier.PublicKey, rho *int2.Int, r *int2.Int) (*int2.Int, *int2.Int, *int2.Int) {
	// Fig 14.3
	z1 := new(big.Int).Mul(e, y)
	z1 = new(big.Int).Add(alpha, z1)

	z2 := new(big.Int).Mul(e, mu)
	z2 = new(big.Int).Add(v, z2)

	modN := int2.ModInt(pk.N)
	w := modN.Exp(rho, e)
	w = modN.Mul(r, w)
	return z1, z2, w
}

func initProof(ec elliptic.Curve, pk *paillier.PublicKey, NCap *int2.Int, s *int2.Int, y *int2.Int, t *int2.Int) (*int2.Int,
	*int2.Int, *int2.Int, *int2.Int, *int2.Int, *int2.Int, *int2.Int, *int2.Int) {
	q := big.Wrap(ec.Params().N)
	q3 := new(big.Int).Mul(q, q)
	q3 = new(big.Int).Mul(q, q3)
	qNCap := new(big.Int).Mul(q, NCap)
	TwolPlus𝜀 := zkp.TwoTo768
	TwolPlus𝜀NCap := new(big.Int).Mul(TwolPlus𝜀, NCap)

	// Fig 30.1 sample
	alpha := common.GetRandomPositiveInt(q3)
	mu := common.GetRandomPositiveInt(qNCap)
	v := common.GetRandomPositiveInt(TwolPlus𝜀NCap)
	r := common.GetRandomPositiveRelativelyPrimeInt(pk.N)
	/* common.Logger.Infof("dec step 4 - C: %v, x: %v, alpha:%v, mu: %v, v:%v, r:%v",
	common.FormatBigInt(C), common.FormatBigInt(x),
	common.FormatBigInt(alpha), common.FormatBigInt(mu), common.FormatBigInt(v), common.FormatBigInt(r))
	*/

	// Fig 29.1 compute
	modNCap := int2.ModInt(NCap)
	S := modNCap.Exp(s, y)
	S = modNCap.Mul(S, modNCap.Exp(t, mu))

	T := modNCap.Exp(s, alpha)
	T = modNCap.Mul(T, modNCap.Exp(t, v))

	modNSquared := int2.ModInt(pk.NSquare())
	A := modNSquared.Exp(pk.Gamma(), alpha)
	A = modNSquared.Mul(A, modNSquared.Exp(r, pk.N))

	gamma := new(big.Int).Mod(alpha, q)
	return alpha, mu, v, r, S, T, A, gamma
}

func NewProofGivenAux(ec elliptic.Curve, pk *paillier.PublicKey, C, x, NCap, s, t, y, rho, alpha, mu, v, r *big.Int) (*ProofDec, error) {
	if ec == nil || pk == nil || C == nil || x == nil || NCap == nil || s == nil || t == nil || y == nil || rho == nil {
		return nil, errors.New("ProveDec constructor received nil value(s)")
	}

	q := big.Wrap(ec.Params().N)
	q3 := new(big.Int).Mul(q, q)
	q3 = new(big.Int).Mul(q, q3)

	// Fig 29.1 sample

	// Fig 30.1 compute
	modNCap := int2.ModInt(NCap)
	S := modNCap.Exp(s, y)
	S = modNCap.Mul(S, modNCap.Exp(t, mu))

	T := modNCap.Exp(s, alpha)
	T = modNCap.Mul(T, modNCap.Exp(t, v))

	modNSquared := int2.ModInt(pk.NSquare())
	A := modNSquared.Exp(pk.Gamma(), alpha)
	A = modNSquared.Mul(A, modNSquared.Exp(r, pk.N))

	gamma := new(big.Int).Mod(alpha, q)

	// Fig 30.2 e
	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), C, x, NCap, s, t, A, gamma)...)
		e = hash.RejectionSample(q, eHash)
	}

	// Fig 14.3
	z1 := new(big.Int).Mul(e, y)
	z1 = new(big.Int).Add(alpha, z1)

	z2 := new(big.Int).Mul(e, mu)
	z2 = new(big.Int).Add(v, z2)

	modN := int2.ModInt(pk.N)
	w := modN.Exp(rho, e)
	w = modN.Mul(r, w)

	return &ProofDec{S: S, T: T, A: A, Gamma: gamma, Z1: z1, Z2: z2, W: w}, nil
}

func NewProofFromBytes(bzs [][]byte) (*ProofDec, error) {
	if !common.NonEmptyMultiBytes(bzs, ProofDecBytesParts) {
		return nil, fmt.Errorf("expected %d byte parts to construct ProofDec", ProofDecBytesParts)
	}
	return &ProofDec{
		S:     new(big.Int).SetBytes(bzs[0]),
		T:     new(big.Int).SetBytes(bzs[1]),
		A:     new(big.Int).SetBytes(bzs[2]),
		Gamma: new(big.Int).SetBytes(bzs[3]),
		Z1:    new(big.Int).SetBytes(bzs[4]),
		Z2:    new(big.Int).SetBytes(bzs[5]),
		W:     new(big.Int).SetBytes(bzs[6]),
	}, nil
}

func (pf *ProofDec) Verify(ec elliptic.Curve, pk *paillier.PublicKey, C, x, NCap, s, t *big.Int) bool {
	if pf == nil || !pf.ValidateBasic() || ec == nil || pk == nil || C == nil || x == nil || NCap == nil || s == nil || t == nil {
		return false
	}

	q := big.Wrap(ec.Params().N)
	// q3 := new(big.Int).Mul(q, q)
	// q3 = new(big.Int).Mul(q, q3)

	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), C, x, NCap, s, t, pf.A, pf.Gamma)...)
		e = hash.RejectionSample(q, eHash)
	}

	return doVerify(pk, pf, C, e, q, x, NCap, s, t)
}

func (pf *ProofDec) VerifyWithNonce(ec elliptic.Curve, pk *paillier.PublicKey, C, x, NCap, s, t, nonce *big.Int) bool {
	if pf == nil || !pf.ValidateBasic() || ec == nil || pk == nil || C == nil || x == nil || NCap == nil || s == nil ||
		t == nil || nonce == nil || big.NewInt(0).Cmp(nonce) == 0 {
		return false
	}

	q := big.Wrap(ec.Params().N)
	// q3 := new(big.Int).Mul(q, q)
	// q3 = new(big.Int).Mul(q, q3)

	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), C, x, NCap, s, t, pf.A, pf.Gamma, nonce)...)
		e = hash.RejectionSample(q, eHash)
	}

	return doVerify(pk, pf, C, e, q, x, NCap, s, t)
}

func doVerify(pk *paillier.PublicKey, pf *ProofDec, C *int2.Int, e *int2.Int, q *int2.Int, x *int2.Int, NCap *int2.Int,
	s *int2.Int, t *int2.Int) bool {
	// Fig 30. Equality Check
	{
		modNSquare := int2.ModInt(pk.NSquare())
		Np1EXPz1 := modNSquare.Exp(pk.Gamma(), pf.Z1)
		wEXPN := modNSquare.Exp(pf.W, pk.N)
		left := modNSquare.Mul(Np1EXPz1, wEXPN)

		CEXPe := modNSquare.Exp(C, e)
		right := modNSquare.Mul(pf.A, CEXPe)

		if left.Cmp(right) != 0 {
			return false
		}
	}

	{
		modQ := int2.ModInt(q)
		left := new(big.Int).Mod(pf.Z1, q)
		right := modQ.Add(modQ.Mul(e, x), pf.Gamma)

		if left.Cmp(right) != 0 {
			return false
		}
	}

	{
		modNCap := int2.ModInt(NCap)
		sEXPz1 := modNCap.Exp(s, pf.Z1)
		tEXPz2 := modNCap.Exp(t, pf.Z2)
		left := modNCap.Mul(sEXPz1, tEXPz2)

		SEXPe := modNCap.Exp(pf.S, e)
		right := modNCap.Mul(pf.T, SEXPe)

		if left.Cmp(right) != 0 {
			return false
		}
	}
	return true
}

func (pf *ProofDec) ValidateBasic() bool {
	return pf.S != nil &&
		pf.T != nil &&
		pf.A != nil &&
		pf.Gamma != nil &&
		pf.Z1 != nil &&
		pf.Z2 != nil &&
		pf.W != nil
}

func (pf *ProofDec) Bytes() [ProofDecBytesParts][]byte {
	return [...][]byte{
		pf.S.Bytes(),
		pf.T.Bytes(),
		pf.A.Bytes(),
		pf.Gamma.Bytes(),
		pf.Z1.Bytes(),
		pf.Z2.Bytes(),
		pf.W.Bytes(),
	}
}

func FormatProofDec(proof *ProofDec) string {
	return "(S:" + common.FormatBigInt(proof.S) + ", T:" + common.FormatBigInt(proof.T) +
		", A:" + common.FormatBigInt(proof.A) + ", Gamma:" + common.FormatBigInt(proof.Gamma) +
		", Z1:" + common.FormatBigInt(proof.Z1) + ", Z2:" + common.FormatBigInt(proof.Z2) + ")"
}
