// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package keygen

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"

	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/ipfs/go-log"
	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/vss"
	"github.com/binance-chain/tss-lib/test"
	"github.com/binance-chain/tss-lib/tss"
)

func setUp(level string) {
	if err := log.SetLogLevel("tss-lib", level); err != nil {
		panic(err)
	}
}

func TestStartRound1Paillier(t *testing.T) {
	setUp("info")

	pIDs := tss.GenerateTestPartyIDs(2)
	p2pCtx := tss.NewPeerContext(pIDs)
	threshold := 1
	params, _ := tss.NewParameters(tss.EC(), p2pCtx, pIDs[0], len(pIDs), threshold)

	fixtures, pIDs, err := LoadKeygenTestFixtures(testParticipants)
	if err != nil {
		common.Logger.Info("No test fixtures were found, so the safe primes will be generated from scratch. This may take a while...")
		pIDs = tss.GenerateTestPartyIDs(testParticipants)
	}
	q := big.Wrap(tss.EC().Params().N)
	sessionId := common.GetBigRandomPositiveInt(q, q.BitLen())
	var lp *LocalParty
	out := make(chan tss.Message, len(pIDs))
	if 0 < len(fixtures) {
		lp_, _ := NewLocalParty(params, out, nil, sessionId, fixtures[0].LocalPreParams)
		lp = lp_.(*LocalParty)
	} else {
		lp_, _ := NewLocalParty(params, out, nil, sessionId)
		lp = lp_.(*LocalParty)
	}
	if err := lp.Start(); err != nil {
		assert.FailNow(t, err.Error())
	}
	<-out

	// Paillier modulus 2048 (two 1024-bit primes)
	// round up to 256, it was used to be flaky, sometimes comes back with 1 byte less
	len1 := len(lp.data.PaillierSK.LambdaN.Bytes())
	len2 := len(lp.data.PaillierSK.PublicKey.N.Bytes())
	if len1%2 != 0 {
		len1 = len1 + (256 - (len1 % 256))
	}
	if len2%2 != 0 {
		len2 = len2 + (256 - (len2 % 256))
	}
	assert.Equal(t, 2048/8, len1)
	assert.Equal(t, 2048/8, len2)
}

func TestFinishAndSaveH1H2(t *testing.T) {
	setUp("info")

	pIDs := tss.GenerateTestPartyIDs(2)
	p2pCtx := tss.NewPeerContext(pIDs)
	threshold := 1
	params, _ := tss.NewParameters(tss.EC(), p2pCtx, pIDs[0], len(pIDs), threshold)

	fixtures, pIDs, err := LoadKeygenTestFixtures(testParticipants)
	if err != nil {
		common.Logger.Info("No test fixtures were found, so the safe primes will be generated from scratch. This may take a while...")
		pIDs = tss.GenerateTestPartyIDs(testParticipants)
	}
	q := big.Wrap(tss.EC().Params().N)
	sessionId := common.GetBigRandomPositiveInt(q, q.BitLen())
	var lp *LocalParty
	out := make(chan tss.Message, len(pIDs))
	if 0 < len(fixtures) {
		lp_, _ := NewLocalParty(params, out, nil, sessionId, fixtures[0].LocalPreParams)
		lp = lp_.(*LocalParty)
	} else {
		lp_, _ := NewLocalParty(params, out, nil, sessionId)
		lp = lp_.(*LocalParty)
	}
	if err := lp.Start(); err != nil {
		assert.FailNow(t, err.Error())
	}

	// RSA modulus 2048 (two 1024-bit primes)
	// round up to 256
	len1 := len(lp.data.H1j[0].Bytes())
	len2 := len(lp.data.H2j[0].Bytes())
	len3 := len(lp.data.NTildej[0].Bytes())
	if len1%2 != 0 {
		len1 = len1 + (256 - (len1 % 256))
	}
	if len2%2 != 0 {
		len2 = len2 + (256 - (len2 % 256))
	}
	if len3%2 != 0 {
		len3 = len3 + (256 - (len3 % 256))
	}
	// 256 bytes = 2048 bits
	assert.Equal(t, 256, len1, "h1 should be correct len")
	assert.Equal(t, 256, len2, "h2 should be correct len")
	assert.Equal(t, 256, len3, "n-tilde should be correct len")
	assert.NotZero(t, lp.data.H1i, "h1 should be non-zero")
	assert.NotZero(t, lp.data.H2i, "h2 should be non-zero")
	assert.NotZero(t, lp.data.NTildei, "n-tilde should be non-zero")
}

func TestBadMessageCulprits(t *testing.T) {
	setUp("info")

	pIDs := tss.GenerateTestPartyIDs(2)
	p2pCtx := tss.NewPeerContext(pIDs)
	params, _ := tss.NewParameters(tss.S256(), p2pCtx, pIDs[0], len(pIDs), 1)

	fixtures, pIDs, err := LoadKeygenTestFixtures(testParticipants)
	if err != nil {
		common.Logger.Info("No test fixtures were found, so the safe primes will be generated from scratch. This may take a while...")
		pIDs = tss.GenerateTestPartyIDs(testParticipants)
	}
	q := big.Wrap(tss.EC().Params().N)
	sessionId := common.GetRandomPositiveInt(q)
	var lp *LocalParty
	out := make(chan tss.Message, len(pIDs))
	if 0 < len(fixtures) {
		lp_, _ := NewLocalParty(params, out, nil, sessionId, fixtures[0].LocalPreParams)
		lp = lp_.(*LocalParty)
	} else {
		lp_, _ := NewLocalParty(params, out, nil, sessionId)
		lp = lp_.(*LocalParty)
	}
	if err := lp.Start(); err != nil {
		assert.FailNow(t, err.Error())
	}

	// badMsg := NewKGRound1Message(pIDs[1], zero, &paillier.PublicKey{N: zero}, zero, zero, zero)
	badMsg := NewKGRound1Message(sessionId, pIDs[1], zero, zero, zero, zero)
	ok, err2 := lp.Update(badMsg)
	t.Log(err2)
	assert.False(t, ok)
	if !assert.Error(t, err2) {
		return
	}
	assert.Equal(t, 1, len(err2.Culprits()))
	assert.Equal(t, pIDs[1], err2.Culprits()[0])
	assert.Regexpf(t, `^task ecdsa-keygen, party.+round 1, culprits.+message failed ValidateBasic.+KGRound1Message`, err2.Error(), "unexpected culprit error message")
	assert.Regexpf(t, `^task ecdsa-keygen, party.+round 1, culprits.+1,.*2.+message failed ValidateBasic.+KGRound1Message`, err2.Error(), "unexpected culprit error message")

	// expected: "task ecdsa-keygen, party {0,P[1]}, round 1, culprits [{1,2}]: message failed ValidateBasic: Type: binance.tsslib.ecdsa.keygen.KGRound1Message, From: {1,2}",
	// or "[...] culprits [{1,P[2]}]: message failed[...]"

}

func TestE2EConcurrentAndSaveFixtures(t *testing.T) {
	setUp("debug")

	// tss.SetCurve(elliptic.P256())

	threshold := testThreshold
	fixtures, pIDs, err := LoadKeygenTestFixtures(testParticipants)
	if err != nil {
		common.Logger.Info("No test fixtures were found, so the safe primes will be generated from scratch. This may take a while...",
			err)
		pIDs = tss.GenerateTestPartyIDs(testParticipants)
	}

	p2pCtx := tss.NewPeerContext(pIDs)
	parties := make([]*LocalParty, 0, len(pIDs))

	errCh := make(chan *tss.Error, len(pIDs))
	outCh := make(chan tss.Message, len(pIDs))
	endCh := make(chan LocalPartySaveData, len(pIDs))

	updater := test.SharedPartyUpdater

	startGR := runtime.NumGoroutine()

	parties, errCh = InitTheParties(pIDs, p2pCtx, threshold, fixtures, outCh, endCh, parties, errCh)

	// PHASE: keygen
	var ended int32
keygen:
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case err := <-errCh:
			common.Logger.Errorf("Error: %s", err)
			assert.FailNow(t, err.Error())
			break keygen

		case msg := <-outCh:
			if HandleMessage(t, msg, parties, updater, errCh) {
				return
			}

		case save := <-endCh:
			// SAVE a test fixture file for this P (if it doesn't already exist)
			// .. here comes a workaround to recover this party's index (it was removed from save data)
			index, err := save.OriginalIndex()
			assert.NoErrorf(t, err, "should not be an error getting a party's index from save data")
			TryWriteTestFixtureFile(t, index, save)

			atomic.AddInt32(&ended, 1)
			if atomic.LoadInt32(&ended) == int32(len(pIDs)) {
				t.Logf("Done. Received save data from %d participants", ended)

				// combine shares for each Pj to get u
				u := big.NewInt(0)
				for j, Pj := range parties {
					pShares := make(vss.Shares, 0)
					for j2, P := range parties {
						var share *big.Int
						P.Lock()
						if j2 == j {
							share = P.temp.shares[j].Share
						} else {
							share = P.temp.r3msgxij[j]
						}
						// vssMsgs := P.temp.kgRound3Messages
						// share := vssMsgs[j].Content().(*KGRound3Message).Share
						shareStruct := &vss.Share{
							Threshold: threshold,
							ID:        P.PartyID().KeyInt(),
							Share:     share, // new(big.Int).SetBytes(share),
						}
						pShares = append(pShares, shareStruct)
						P.Unlock()
					}
					Pj.Lock()
					uj, errRec := pShares[:threshold+1].ReConstruct(tss.EC())
					assert.NoError(t, errRec, "vss.ReConstruct should not throw error")

					// uG test: u*G[j] == V[0]
					assert.Equal(t, uj.Cmp(Pj.temp.ui), 0)
					uG := crypto.ScalarBaseMult(tss.EC(), uj)
					V0 := Pj.temp.vs[0]
					if Pj.temp.r2msgVss[j] != nil {
						V0 = Pj.temp.r2msgVss[j][0]
					}
					assert.True(t, uG.Equals(V0), "ensure u*G[j] == V_0")

					// xj tests: BigXj == xj*G
					xj := Pj.data.Xi
					gXj := crypto.ScalarBaseMult(tss.EC(), xj)
					BigXj := Pj.data.BigXj[j]
					assert.True(t, BigXj.Equals(gXj), "ensure BigX_j == g^x_j")

					// fails if threshold cannot be satisfied (bad share)
					{
						badShares := pShares[:threshold]
						badShares[len(badShares)-1].Share.Set(big.NewInt(0))
						uj, err := pShares[:threshold+1].ReConstruct(tss.S256())
						assert.NoError(t, err)
						assert.NotEqual(t, parties[j].temp.ui, uj)
						BigXjX, BigXjY := tss.EC().ScalarBaseMult(uj.Bytes())
						V_0 := Pj.temp.vs[0]
						if Pj.temp.r2msgVss[j] != nil {
							V_0 = Pj.temp.r2msgVss[j][0]
						}
						assert.NotEqual(t, BigXjX, V_0.X())
						assert.NotEqual(t, BigXjY, V_0.Y())
					}
					u = new(big.Int).Add(u, uj)
					Pj.Unlock()
				}

				// build ecdsa key pair
				pkX, pkY := save.ECDSAPub.X(), save.ECDSAPub.Y()
				pk := ecdsa.PublicKey{
					Curve: tss.EC(),
					X:     pkX,
					Y:     pkY,
				}
				sk := ecdsa.PrivateKey{
					PublicKey: pk,
					D:         u,
				}
				// test pub key, should be on curve and match pkX, pkY
				assert.True(t, sk.IsOnCurve(pkX, pkY), "public key must be on curve")

				// public key tests
				assert.NotZero(t, u, "u should not be zero")
				ourPk := crypto.ScalarBaseMult(tss.EC(), u)

				assert.Equal(t, pkX, ourPk.X(), "pkX should match expected pk derived from u")
				assert.Equal(t, pkY, ourPk.Y(), "pkY should match expected pk derived from u")
				t.Log("Public key tests done.")

				// make sure everyone has the same ECDSA public key
				for _, Pj := range parties {
					assert.Equal(t, pkX, Pj.data.ECDSAPub.X())
					assert.Equal(t, pkY, Pj.data.ECDSAPub.Y())
				}
				t.Log("Public key distribution test done.")

				// test sign/verify
				data := make([]byte, 32)
				for i := range data {
					data[i] = byte(i)
				}
				r, s, err := ecdsa.Sign(rand.Reader, &sk, data)
				assert.NoError(t, err, "sign should not throw an error")
				ok := ecdsa.Verify(&pk, data, r, s)
				assert.True(t, ok, "signature should be ok")
				t.Log("ECDSA signing test done.")

				t.Logf("Start goroutines: %d, End goroutines: %d", startGR, runtime.NumGoroutine())

				break keygen
			}
		}
	}
}

func TestTooManyParties(t *testing.T) {
	setUp("info")

	pIDs := tss.GenerateTestPartyIDs(MaxParties + 1)
	p2pCtx := tss.NewPeerContext(pIDs)
	params, _ := tss.NewParameters(tss.S256(), p2pCtx, pIDs[0], len(pIDs), MaxParties/100)
	q := big.Wrap(tss.EC().Params().N)
	sessionId := common.GetBigRandomPositiveInt(q, q.BitLen())
	out := make(chan tss.Message, len(pIDs))
	var err error
	_, err = NewLocalParty(params, out, nil, sessionId)
	if !assert.Error(t, err) {
		t.FailNow()
		return
	}
}
