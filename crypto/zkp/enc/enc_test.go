// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpenc

import (
	"testing"
	"time"

	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/tss"
)

// Using a modulus length of 2048 is recommended in the GG18 spec
const (
	testSafePrimeBits = 1024
)

func TestEnc(test *testing.T) {
	ec := tss.EC()
	q := big.Wrap(ec.Params().N)

	sk, pk, err := paillier.GenerateKeyPair(testSafePrimeBits*2, time.Minute*15)
	assert.NoError(test, err)

	k := common.GetRandomPositiveInt(q)
	K, rho, err := sk.EncryptAndReturnRandomness(k)
	assert.NoError(test, err)

	primes := [2]*big.Int{common.GetRandomPrimeInt(testSafePrimeBits), common.GetRandomPrimeInt(testSafePrimeBits)}
	NCap, s, t, err := crypto.GenerateNTildei(primes)
	assert.NoError(test, err)
	proof, err := NewProof(ec, pk, K, NCap, s, t, k, rho)
	assert.NoError(test, err)

	ok := proof.Verify(ec, pk, NCap, s, t, K)
	assert.True(test, ok, "proof must verify")

	// with nonce
	nonce := common.GetBigRandomPositiveInt(q, q.BitLen())
	proof2, err := NewProofGivenNonce(ec, pk, K, NCap, s, t, k, rho, nonce)
	assert.NoError(test, err)

	ok2 := proof2.VerifyWithNonce(ec, pk, NCap, s, t, K, nonce)
	assert.True(test, ok2, "proof must verify")
}
