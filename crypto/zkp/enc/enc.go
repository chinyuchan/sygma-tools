// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpenc

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
	ProofEncBytesParts = 6
)

type (
	ProofEnc struct {
		S, A, C, Z1, Z2, Z3 *big.Int
	}
)

// NewProof implements proofenc
func NewProof(ec elliptic.Curve, pk *paillier.PublicKey, K, NCap, s, t, k, rho *big.Int) (*ProofEnc, error) {
	if ec == nil || pk == nil || K == nil || NCap == nil || s == nil || t == nil || k == nil || rho == nil {
		return nil, errors.New("ProveEnc constructor received nil value(s)")
	}

	q := big.Wrap(ec.Params().N)
	alpha, mu, r, gamma, S, A, C := initProof(q, NCap, pk, s, k, t)

	// Fig 14.2 e
	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), K, S, A, C)...)
		e = hash.RejectionSample(q, eHash)
	}

	z1, z2, z3 := coda(e, k, alpha, pk, rho, r, mu, gamma)

	return &ProofEnc{S: S, A: A, C: C, Z1: z1, Z2: z2, Z3: z3}, nil
}

func NewProofGivenNonce(ec elliptic.Curve, pk *paillier.PublicKey, K, NCap, s, t, k, rho, nonce *big.Int) (*ProofEnc, error) {
	if ec == nil || pk == nil || K == nil || NCap == nil || s == nil || t == nil || k == nil || rho == nil || nonce == nil ||
		big.NewInt(0).Cmp(nonce) == 0 {
		return nil, errors.New("ProveEnc constructor received nil value(s)")
	}
	if nonce.BitLen() < ec.Params().N.BitLen()-1 {
		return nil, errors.New("invalid nonce")
	}
	q := big.Wrap(ec.Params().N)
	alpha, mu, r, gamma, S, A, C := initProof(q, NCap, pk, s, k, t)

	// Fig 14.2 e
	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), K, S, A, C, nonce)...)
		e = hash.RejectionSample(q, eHash)
	}

	z1, z2, z3 := coda(e, k, alpha, pk, rho, r, mu, gamma)

	return &ProofEnc{S: S, A: A, C: C, Z1: z1, Z2: z2, Z3: z3}, nil
}

func coda(e *int2.Int, k *int2.Int, alpha *int2.Int, pk *paillier.PublicKey, rho *int2.Int, r *int2.Int, mu *int2.Int, gamma *int2.Int) (*int2.Int, *int2.Int, *int2.Int) {
	// Fig 14.3
	z1 := new(big.Int).Mul(e, k)
	z1 = new(big.Int).Add(z1, alpha)

	modN := int2.ModInt(pk.N)
	z2 := modN.Exp(rho, e)
	z2 = modN.Mul(z2, r)

	z3 := new(big.Int).Mul(e, mu)
	z3 = new(big.Int).Add(z3, gamma)
	return z1, z2, z3
}

func initProof(q *int2.Int, NCap *int2.Int, pk *paillier.PublicKey, s *int2.Int, k *int2.Int, t *int2.Int) (*int2.Int, *int2.Int, *int2.Int, *int2.Int, *int2.Int, *int2.Int, *int2.Int) {
	q3 := new(big.Int).Mul(q, q)
	q3 = new(big.Int).Mul(q, q3)
	qNCap := new(big.Int).Mul(q, NCap)
	q3NCap := new(big.Int).Mul(q3, NCap)

	// Fig 14.1 sample
	alpha := common.GetRandomPositiveInt(q3)
	mu := common.GetRandomPositiveInt(qNCap)
	r := common.GetRandomPositiveRelativelyPrimeInt(pk.N)
	gamma := common.GetRandomPositiveInt(q3NCap)

	// Fig 14.1 compute
	modNCap := int2.ModInt(NCap)
	S := modNCap.Exp(s, k)
	S = modNCap.Mul(S, modNCap.Exp(t, mu))

	modNSquared := int2.ModInt(pk.NSquare())
	A := modNSquared.Exp(pk.Gamma(), alpha)
	A = modNSquared.Mul(A, modNSquared.Exp(r, pk.N))

	C := modNCap.Exp(s, alpha)
	C = modNCap.Mul(C, modNCap.Exp(t, gamma))
	return alpha, mu, r, gamma, S, A, C
}

func NewProofFromBytes(bzs [][]byte) (*ProofEnc, error) {
	if !common.NonEmptyMultiBytes(bzs, ProofEncBytesParts) {
		return nil, fmt.Errorf("expected %d byte parts to construct ProofEnc", ProofEncBytesParts)
	}
	return &ProofEnc{
		S:  new(big.Int).SetBytes(bzs[0]),
		A:  new(big.Int).SetBytes(bzs[1]),
		C:  new(big.Int).SetBytes(bzs[2]),
		Z1: new(big.Int).SetBytes(bzs[3]),
		Z2: new(big.Int).SetBytes(bzs[4]),
		Z3: new(big.Int).SetBytes(bzs[5]),
	}, nil
}

func (pf *ProofEnc) Verify(ec elliptic.Curve, pk *paillier.PublicKey, NCap, s, t, K *big.Int) bool {
	if pf == nil || !pf.ValidateBasic() || ec == nil || pk == nil || NCap == nil || s == nil || t == nil || K == nil {
		return false
	}

	q := big.Wrap(ec.Params().N)
	q3 := new(big.Int).Mul(q, q)
	q3 = new(big.Int).Mul(q, q3)

	// Fig 14. Range Check
	if pf.Z1.Cmp(q3) == 1 {
		return false
	}

	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), K, pf.S, pf.A, pf.C)...)
		e = hash.RejectionSample(q, eHash)
	}

	return doVerify(pk, pf, K, e, NCap, s, t)
}

func (pf *ProofEnc) VerifyWithNonce(ec elliptic.Curve, pk *paillier.PublicKey, NCap, s, t, K, nonce *big.Int) bool {
	if pf == nil || !pf.ValidateBasic() || ec == nil || pk == nil || NCap == nil || s == nil || t == nil || K == nil {
		return false
	}

	q := big.Wrap(ec.Params().N)
	q3 := new(big.Int).Mul(q, q)
	q3 = new(big.Int).Mul(q, q3)

	// Fig 14. Range Check
	if pf.Z1.Cmp(q3) == 1 {
		return false
	}

	var e *big.Int
	{
		eHash := hash.SHA256i(append(pk.AsInts(), K, pf.S, pf.A, pf.C, nonce)...)
		e = hash.RejectionSample(q, eHash)
	}

	return doVerify(pk, pf, K, e, NCap, s, t)
}

func doVerify(pk *paillier.PublicKey, pf *ProofEnc, K *int2.Int, e *int2.Int, NCap *int2.Int, s *int2.Int, t *int2.Int) bool {
	// Fig 14. Equality Check
	{
		modNSquare := int2.ModInt(pk.NSquare())
		Np1EXPz1 := modNSquare.Exp(pk.Gamma(), pf.Z1)
		z2EXPN := modNSquare.Exp(pf.Z2, pk.N)
		left := modNSquare.Mul(Np1EXPz1, z2EXPN)

		KEXPe := modNSquare.Exp(K, e)
		right := modNSquare.Mul(pf.A, KEXPe)

		if left.Cmp(right) != 0 {
			return false
		}
	}

	{
		modNCap := int2.ModInt(NCap)
		sEXPz1 := modNCap.Exp(s, pf.Z1)
		tEXPz3 := modNCap.Exp(t, pf.Z3)
		left := modNCap.Mul(sEXPz1, tEXPz3)

		SEXPe := modNCap.Exp(pf.S, e)
		right := modNCap.Mul(pf.C, SEXPe)
		if left.Cmp(right) != 0 {
			return false
		}
	}
	return true
}

func (pf *ProofEnc) ValidateBasic() bool {
	return pf.S != nil &&
		pf.A != nil &&
		pf.C != nil &&
		pf.Z1 != nil &&
		pf.Z2 != nil &&
		pf.Z3 != nil
}

func (pf *ProofEnc) Bytes() [ProofEncBytesParts][]byte {
	return [...][]byte{
		pf.S.Bytes(),
		pf.A.Bytes(),
		pf.C.Bytes(),
		pf.Z1.Bytes(),
		pf.Z2.Bytes(),
		pf.Z3.Bytes(),
	}
}
