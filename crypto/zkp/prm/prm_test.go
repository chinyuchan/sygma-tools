// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpprm_test

import (
	"testing"
	"time"

	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/stretchr/testify/assert"

	. "github.com/binance-chain/tss-lib/crypto/zkp/prm"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
)

func TestPrm(test *testing.T) {
	preParams, err := keygen.GeneratePreParams(time.Minute*20, 8)
	assert.NoError(test, err)

	s, t, lambda, P, Q, N := preParams.H1i, preParams.H2i, preParams.Beta, preParams.P, preParams.Q, preParams.NTildei
	P2, Q2 := new(big.Int).Lsh(P, 1), new(big.Int).Lsh(Q, 1)
	Phi := new(big.Int).Mul(P2, Q2)

	proof, err := NewProof(s, t, N, Phi, lambda)
	assert.NoError(test, err)

	proofBzs := proof.Bytes()
	proof, err = NewProofFromBytes(proofBzs[:])
	assert.NoError(test, err)

	ok := proof.Verify(s, t, N)
	assert.True(test, ok, "proof must verify")
}

func TestPrmWithNonce(test *testing.T) {
	preParams, err := keygen.GeneratePreParams(time.Minute*10, 8)
	assert.NoError(test, err)
	secret := common.GetBigRandomPositiveInt(big.Wrap(tss.EC().Params().N), big.Wrap(tss.EC().Params().N).BitLen()-1)

	s, t, lambda, P, Q, N := preParams.H1i, preParams.H2i, preParams.Beta, preParams.P, preParams.Q, preParams.NTildei
	P2, Q2 := new(big.Int).Lsh(P, 1), new(big.Int).Lsh(Q, 1)
	Phi := new(big.Int).Mul(P2, Q2)

	proof, err := NewProofWithNonce(s, t, N, Phi, lambda, secret)
	assert.NoError(test, err)

	ok := proof.VerifyWithNonce(s, t, N, secret)
	assert.True(test, ok, "proof must verify")
}

func TestA1(test *testing.T) {
	preParams, err := keygen.GeneratePreParams(time.Minute*10, 8)
	assert.NoError(test, err)

	s, t, lambda, P, Q, N := preParams.H1i, preParams.H2i, preParams.Beta, preParams.P, preParams.Q, preParams.NTildei
	P2, Q2 := new(big.Int).Lsh(P, 1), new(big.Int).Lsh(Q, 1)
	Phi := new(big.Int).Mul(P2, Q2)

	proof, err := NewProof(s, t, N, Phi, lambda)
	assert.NoError(test, err)
	proof.A[1] = big.NewInt(1)

	ok := proof.Verify(s, t, N)
	assert.False(test, ok, "proof must verify")
}
