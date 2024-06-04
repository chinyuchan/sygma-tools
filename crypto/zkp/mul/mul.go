// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpmul

import (
	"crypto/elliptic"
	"errors"
	"fmt"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/common/hash"
	big "github.com/binance-chain/tss-lib/common/int"
	int2 "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto/paillier"
)

const (
	ProofMulBytesParts = 5
)

type (
	ProofMul struct {
		A, B, Z, U, V *big.Int
	}
)

// NewProof implements proofenc
func NewProof(ec elliptic.Curve, pk *paillier.PublicKey, X, Y, C, x, rhox *big.Int) (*ProofMul, error) {
	if pk == nil || X == nil || Y == nil || C == nil || rhox == nil {
		return nil, errors.New("ProveMul constructor received nil value(s)")
	}
	q := big.Wrap(ec.Params().N)

	alpha, r, s, A, B := initProof(pk, Y)

	// Fig 28.2 e
	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), X, Y, C, A, B)...)
		e = hash.RejectionSample(q, eHash)
	}

	z, v := coda(e, x, alpha, pk, rhox, s)

	return &ProofMul{A: A, B: B, Z: z, U: r, V: v}, nil
}

func NewProofGivenNonce(ec elliptic.Curve, pk *paillier.PublicKey, X, Y, C, x, rhox, nonce *big.Int) (*ProofMul, error) {
	if pk == nil || X == nil || Y == nil || C == nil || rhox == nil || nonce == nil ||
		big.NewInt(0).Cmp(nonce) == 0 {
		return nil, errors.New("ProveMul constructor received nil value(s)")
	}
	if nonce.BitLen() < ec.Params().N.BitLen()-1 {
		return nil, errors.New("invalid nonce")
	}
	q := big.Wrap(ec.Params().N)

	alpha, r, s, A, B := initProof(pk, Y)

	// Fig 28.2 e
	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), X, Y, C, A, B, nonce)...)
		e = hash.RejectionSample(q, eHash)
	}

	z, v := coda(e, x, alpha, pk, rhox, s)

	return &ProofMul{A: A, B: B, Z: z, U: r, V: v}, nil
}

func coda(e *int2.Int, x *int2.Int, alpha *int2.Int, pk *paillier.PublicKey, rhox *int2.Int, s *int2.Int) (*int2.Int, *int2.Int) {
	// Fig 14.3
	z := new(big.Int).Mul(e, x)
	z = new(big.Int).Add(z, alpha)

	modN := int2.ModInt(pk.N)
	// u := modN.Exp(rho, e)
	// u = modN.Mul(u, r)

	v := modN.Exp(rhox, e)
	v = modN.Mul(v, s)
	return z, v
}

func initProof(pk *paillier.PublicKey, Y *int2.Int) (*int2.Int, *int2.Int, *int2.Int, *int2.Int, *int2.Int) {
	// Fig 28.1 sample
	alpha := common.GetRandomPositiveRelativelyPrimeInt(pk.N)
	r := common.GetRandomPositiveRelativelyPrimeInt(pk.N)
	s := common.GetRandomPositiveRelativelyPrimeInt(pk.N)

	modNSquared := int2.ModInt(pk.NSquare())
	A := modNSquared.Exp(Y, alpha)
	A = modNSquared.Mul(A, modNSquared.Exp(r, pk.N))

	B := modNSquared.Exp(pk.Gamma(), alpha)
	B = modNSquared.Mul(B, modNSquared.Exp(s, pk.N))
	return alpha, r, s, A, B
}

func NewProofFromBytes(bzs [][]byte) (*ProofMul, error) {
	if !common.NonEmptyMultiBytes(bzs, ProofMulBytesParts) {
		return nil, fmt.Errorf("expected %d byte parts to construct ProofMul", ProofMulBytesParts)
	}
	return &ProofMul{
		A: new(big.Int).SetBytes(bzs[0]),
		B: new(big.Int).SetBytes(bzs[1]),
		Z: new(big.Int).SetBytes(bzs[2]),
		U: new(big.Int).SetBytes(bzs[3]),
		V: new(big.Int).SetBytes(bzs[4]),
	}, nil
}

func (pf *ProofMul) Verify(ec elliptic.Curve, pk *paillier.PublicKey, X, Y, C *big.Int) bool {
	if pf == nil || !pf.ValidateBasic() || ec == nil || pk == nil || X == nil || Y == nil || C == nil {
		return false
	}

	q := big.Wrap(ec.Params().N)

	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), X, Y, C, pf.A, pf.B)...)
		e = hash.RejectionSample(q, eHash)
	}

	return doVerify(pk, Y, pf, C, e, X)
}

func (pf *ProofMul) VerifyWithNonce(ec elliptic.Curve, pk *paillier.PublicKey, X, Y, C, nonce *big.Int) bool {
	if pf == nil || !pf.ValidateBasic() || ec == nil || pk == nil || X == nil || Y == nil || C == nil {
		return false
	}

	q := big.Wrap(ec.Params().N)

	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), X, Y, C, pf.A, pf.B, nonce)...)
		e = hash.RejectionSample(q, eHash)
	}

	return doVerify(pk, Y, pf, C, e, X)
}

func doVerify(pk *paillier.PublicKey, Y *int2.Int, pf *ProofMul, C *int2.Int, e *int2.Int, X *int2.Int) bool {
	// Fig 14. Equality Check
	modNSquare := int2.ModInt(pk.NSquare())
	{
		YEXPz := modNSquare.Exp(Y, pf.Z)
		uEXPN := modNSquare.Exp(pf.U, pk.N)
		left := modNSquare.Mul(YEXPz, uEXPN)

		CEXPe := modNSquare.Exp(C, e)
		right := modNSquare.Mul(pf.A, CEXPe)

		if left.Cmp(right) != 0 {
			return false
		}
	}

	{
		Np1EXPz := modNSquare.Exp(pk.Gamma(), pf.Z)
		CEXPN := modNSquare.Exp(pf.V, pk.N)
		left := modNSquare.Mul(Np1EXPz, CEXPN)

		XEXPe := modNSquare.Exp(X, e)
		right := modNSquare.Mul(pf.B, XEXPe)
		if left.Cmp(right) != 0 {
			return false
		}
	}
	return true
}

func (pf *ProofMul) ValidateBasic() bool {
	return pf.A != nil &&
		pf.B != nil &&
		pf.Z != nil &&
		pf.U != nil &&
		pf.V != nil
}

func (pf *ProofMul) Bytes() [ProofMulBytesParts][]byte {
	return [...][]byte{
		pf.A.Bytes(),
		pf.B.Bytes(),
		pf.Z.Bytes(),
		pf.U.Bytes(),
		pf.V.Bytes(),
	}
}
