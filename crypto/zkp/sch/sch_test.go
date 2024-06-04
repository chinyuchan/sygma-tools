// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpsch

import (
	"testing"

	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/ipfs/go-log"
	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/tss"
)

func TestSchnorrProof(t *testing.T) {
	q := big.Wrap(tss.EC().Params().N)
	u := common.GetRandomPositiveInt(q)
	uG := crypto.ScalarBaseMult(tss.EC(), u)

	proof, err := NewProof(uG, u)

	assert.NoError(t, err, "there should be no error")

	assert.True(t, proof.A.IsOnCurve())
	assert.NotZero(t, proof.A.X())
	assert.NotZero(t, proof.A.Y())
	assert.NotZero(t, proof.Z)
}

func TestSchnorrProofVerify(t *testing.T) {
	q := big.Wrap(tss.EC().Params().N)
	u := common.GetRandomPositiveInt(q)
	X := crypto.ScalarBaseMult(tss.EC(), u)
	nonce := common.GetBigRandomPositiveInt(q, q.BitLen())
	proof, err := NewProofGivenNonce(X, u, nonce)
	assert.NoError(t, err, "there should be no error")

	res := proof.VerifyWithNonce(X, nonce)

	assert.True(t, res, "verify result must be true")
}

func TestSchnorrProofVerifyWithNonce(t *testing.T) {
	q := big.Wrap(tss.EC().Params().N)
	u := common.GetRandomPositiveInt(q)
	X := crypto.ScalarBaseMult(tss.EC(), u)
	nonce := common.GetBigRandomPositiveInt(q, q.BitLen())
	proof, err := NewProofGivenNonce(X, u, nonce)
	assert.NoError(t, err, "there should be no error")

	res := proof.VerifyWithNonce(X, nonce)

	assert.True(t, res, "verify result must be true")
}

func TestSchnorrProofVerifyWithNonceEdwards(t *testing.T) {
	if err := log.SetLogLevel("tss-lib", "debug"); err != nil {
		panic(err)
	}
	curve := tss.Edwards()

	u := big.NewInt(11)
	X := crypto.ScalarBaseMult(curve, u)
	nonce := common.GetBigRandomPositiveInt(big.Wrap(curve.Params().N), big.Wrap(curve.Params().N).BitLen()-1)
	alpha := big.NewInt(90909)
	proof, err := NewProofGivenAlpha(X, u, alpha, nonce)

	// t.Logf("u: %v, X: %v, proof: %v", common.FormatBigInt(u), crypto.FormatECPoint(X), FormatProofSch(proof))
	assert.NoError(t, err, "there should be no error")

	res := proof.VerifyWithNonce(X, nonce)

	assert.True(t, res, "verify result must be true")
}

func TestSchnorrProofVerifyBadX(t *testing.T) {
	q := big.Wrap(tss.EC().Params().N)
	u := common.GetRandomPositiveInt(q)
	u2 := common.GetRandomPositiveInt(q)
	X := crypto.ScalarBaseMult(tss.EC(), u)
	X2 := crypto.ScalarBaseMult(tss.EC(), u2)

	proof, err := NewProof(X2, u2)
	assert.NoError(t, err, "there should be no error")

	res := proof.Verify(X)

	assert.False(t, res, "verify result must be false")
}

func TestZeros(t *testing.T) {
	zero := big.NewInt(0)
	idG := crypto.ScalarBaseMult(tss.EC(), big.NewInt(1))
	proof, err := NewProof(idG, zero)
	assert.Error(t, err, "when x is zero there must be an error")
	assert.Nil(t, proof, "the proof must be nil with the error")

	p2, err2 := NewProofGivenAlpha(idG, zero, zero, zero)
	assert.Error(t, err2, "when alpha is zero there must be an error")
	assert.Nil(t, p2, "the proof must be nil with the error")
}

func TestBadVerify(t *testing.T) {
	q := big.Wrap(tss.EC().Params().N)
	u := common.GetRandomPositiveInt(q)
	X := crypto.ScalarBaseMult(tss.EC(), u)
	nonce := common.GetRandomPositiveInt(q)
	proof, _ := NewProof(X, u)
	proof.Z = big.NewInt(0)
	res := proof.Verify(X)
	assert.False(t, res, "verify result must be false")

	proof.Z = nil
	res2 := proof.VerifyWithNonce(X, nonce)
	assert.False(t, res2, "verify result must be false")
}
