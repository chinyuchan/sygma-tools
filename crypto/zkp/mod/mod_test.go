// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpmod_test

import (
	"testing"
	"time"

	"github.com/binance-chain/tss-lib/common"
	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/tss"

	. "github.com/binance-chain/tss-lib/crypto/zkp/mod"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/stretchr/testify/assert"
)

func TestMod(test *testing.T) {
	preParams, err := keygen.GeneratePreParams(time.Minute*20, 8)
	assert.NoError(test, err)

	p, q, N := preParams.P, preParams.Q, preParams.NTildei
	// p2, q2 := new(big.Int).Mul(p, big.NewInt(2)), new(big.Int).Mul(q, big.NewInt(2))
	p2, q2 := new(big.Int).Lsh(p, 1), new(big.Int).Lsh(q, 1)
	P, Q := new(big.Int).Add(p2, big.NewInt(1)), new(big.Int).Add(q2, big.NewInt(1))
	nonce := common.GetBigRandomPositiveInt(q, q.BitLen())
	order := big.Wrap(tss.S256().Params().N)

	proof, err := NewProof(order, N, P, Q, nonce)
	assert.NoError(test, err)

	proofBzs := proof.Bytes()
	proof, err = NewProofFromBytes(proofBzs[:])
	assert.NoError(test, err)

	ok := proof.Verify(order, N, nonce)
	assert.True(test, ok, "proof must verify")

	nonce = common.GetBigRandomPositiveInt(q, q.BitLen())
	proof2, err2 := NewProof(order, N, P, Q, nonce)
	assert.NoError(test, err2)

	ok2 := proof2.Verify(order, N, nonce)
	assert.True(test, ok2, "proof must verify")
}

func TestBadW(test *testing.T) {
	preParams, err := keygen.GeneratePreParams(time.Minute*20, 8)
	if !assert.NoError(test, err) {
		return
	}

	p, q, N := preParams.P, preParams.Q, preParams.NTildei
	// p2, q2 := new(big.Int).Mul(p, big.NewInt(2)), new(big.Int).Mul(q, big.NewInt(2))
	p2, q2 := new(big.Int).Lsh(p, 1), new(big.Int).Lsh(q, 1)
	P, Q := new(big.Int).Add(p2, big.NewInt(1)), new(big.Int).Add(q2, big.NewInt(1))
	nonce := common.GetRandomPrimeInt(q.BitLen())
	order := big.Wrap(tss.S256().Params().N)

	pr, err := NewProof(order, N, P, Q, nonce)
	if !assert.NoError(test, err) {
		return
	}
	pr.W = nil
	ok := pr.Verify(order, N, nonce)
	assert.False(test, ok, "proof with nil W must not verify")

	pr.W = big.NewInt(0)
	ok2 := pr.Verify(order, N, nonce)
	assert.False(test, ok2, "proof must not verify")

	pr.W = GetRandomNonQuadraticNonResidue(N)
	ok3 := pr.Verify(order, N, nonce)
	assert.False(test, ok3, "proof must not verify")
}
