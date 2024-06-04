// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package resharing_test

import (
	"crypto/ecdsa"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"

	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/ipfs/go-log"
	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	. "github.com/binance-chain/tss-lib/ecdsa/resharing"
	"github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/binance-chain/tss-lib/test"
	"github.com/binance-chain/tss-lib/tss"
)

const (
	testParticipants = test.TestParticipants
	testThreshold    = test.TestThreshold
)

func setUp(level string) {
	if err := log.SetLogLevel("tss-lib", level); err != nil {
		panic(err)
	}
}

func TestKeyRotationE2EConcurrent(t *testing.T) {
	setUp("info")

	_testResharing(t, 0)
}

func TestKeyResharingWithNewMembersE2EConcurrent(t *testing.T) {
	setUp("info")

	_testResharing(t, 1)
}

// func TestKeyResharingWithRemoveMemberE2EConcurrent(t *testing.T) {
// 	setUp("info")
//
// 	_testResharing(t, -1)
// }

func _testResharing(t *testing.T, newParticipants int) {
	threshold, newThreshold := testThreshold, testThreshold

	// PHASE: load keygen fixtures
	oldKeys, oldParties, err := keygen.LoadKeygenTestFixtures(testParticipants, 0)
	assert.NoError(t, err, "should load keygen fixtures")

	// PHASE: resharing
	oldCtx := tss.NewPeerContext(oldParties)
	// init the new parties; re-use the fixture pre-params for speed
	fixtures, _, err := keygen.LoadKeygenTestFixtures(testParticipants)
	if err != nil {
		common.Logger.Info("No test fixtures were found, so the safe primes will be generated from scratch. This may take a while...")
	}

	newParties := tss.SortedPartyIDs{}
	for i := 0; i < testParticipants; i++ {
		newParties = append(newParties, oldParties[i])
	}

	if newParticipants > 0 {
		for i := 0; i < newParticipants; i++ {
			newParties = append(newParties, tss.GenerateTestPartyIDs(testParticipants + newParticipants)[testParticipants+i])
		}
	} else {
		newParties = newParties[0 : testParticipants+newParticipants]
	}

	newParties = sortParties(newParties, oldParties)
	newPCount := len(newParties)
	newCtx := tss.NewPeerContext(newParties)

	oldCommittee := make([]*LocalParty, 0, len(oldParties))
	newCommittee := make([]*LocalParty, 0, newPCount)
	bothCommitteesPax := len(oldCommittee) + len(newCommittee)

	errCh := make(chan *tss.Error, bothCommitteesPax)
	outCh := make(chan tss.Message, bothCommitteesPax)
	endCh := make(chan keygen.LocalPartySaveData, bothCommitteesPax)
	q := big.Wrap(tss.EC().Params().N)
	sessionId := common.GetBigRandomPositiveInt(q, q.BitLen())

	updater := test.SharedPartyUpdater

	oldCommitteeMap := make(map[*tss.PartyID]*LocalParty)

	// init the old parties first
	for j, pID := range oldParties {
		params, _ := tss.NewReSharingParameters(tss.S256(), oldCtx, newCtx, pID, len(oldParties), threshold, newPCount, newThreshold)
		P_, err := NewLocalParty(params, oldKeys[j], outCh, endCh, sessionId) // discard old key data
		if err != nil {
			assert.NoError(t, err, "init the old parties error")
			return
		}
		P := P_.(*LocalParty)
		oldCommitteeMap[pID] = P
		oldCommittee = append(oldCommittee, P)
	}

	// init the new parties
	for j, pID := range newParties {
		params, _ := tss.NewReSharingParameters(tss.S256(), oldCtx, newCtx, pID, len(oldParties), threshold, newPCount, newThreshold)
		var P *LocalParty
		if p, isOld := oldCommitteeMap[pID]; isOld {
			P = p
		} else {
			save := keygen.NewLocalPartySaveData(newPCount)
			if j < len(fixtures) && len(newParties) <= len(fixtures) {
				save.LocalPreParams = fixtures[j].LocalPreParams
			}
			P_, _ := NewLocalParty(params, save, outCh, endCh, sessionId)
			P = P_.(*LocalParty)
		}

		newCommittee = append(newCommittee, P)
	}

	// start the old parties; they will send messages
	for _, P := range oldCommittee {
		go func(P *LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
		}(P)
	}

	// start the new parties; they will wait for messages
	for _, P := range newCommittee {
		if _, ok := oldCommitteeMap[P.PartyID()]; !ok {
			go func(P *LocalParty) {
				if err := P.Start(); err != nil {
					errCh <- err
				}
			}(P)
		}
	}

	newKeys := make([]keygen.LocalPartySaveData, len(newCommittee))
	endedOldCommittee := 0
	var reSharingEnded int32
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case err := <-errCh:
			common.Logger.Errorf("Error: %s", err)
			assert.FailNow(t, err.Error())
			return

		case msg := <-outCh:
			dest := msg.GetTo()
			if dest == nil {
				t.Fatal("did not expect a msg to have a nil destination during resharing")
			}
			if msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, destP := range dest[:len(oldCommittee)] {
					if destP != nil {
						go updater(oldCommittee[destP.Index], msg, errCh)
					}
				}
			}

			if !msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, destP := range dest {
					if destP != nil {
						go updater(newCommittee[destP.Index], msg, errCh)
					}
				}
			}

		case save := <-endCh:
			// old committee members that aren't receiving a share have their Xi zeroed
			if save.Xi != nil {
				index, err := save.OriginalIndex()
				assert.NoErrorf(t, err, "should not be an error getting a party's index from save data")
				newKeys[index] = save
			} else {
				endedOldCommittee++
			}
			atomic.AddInt32(&reSharingEnded, 1)
			if atomic.LoadInt32(&reSharingEnded) == int32(len(oldCommittee)+len(newCommittee)-testParticipants) {
				assert.Equal(t, len(oldCommittee)-testParticipants, endedOldCommittee)
				t.Logf("Resharing done. Reshared %d participants", reSharingEnded)

				// xj tests: BigXj == xj*G
				for j, key := range newKeys {
					// xj test: BigXj == xj*G
					xj := key.Xi
					gXj := crypto.ScalarBaseMult(tss.S256(), xj)
					BigXj := key.BigXj[j]
					assert.True(t, BigXj.Equals(gXj), "ensure BigX_j == g^x_j")
				}

				// more verification of signing is implemented within local_party_test.go of keygen package
				goto signing
			}
		}
	}

signing:
	// PHASE: signing
	signKeys, signPIDs := newKeys, newParties
	signP2pCtx := tss.NewPeerContext(signPIDs)
	signParties := make([]*signing.LocalParty, 0, len(signPIDs))

	signErrCh := make(chan *tss.Error, len(signPIDs))
	signOutCh := make(chan tss.Message, len(signPIDs))
	signEndCh := make(chan common.SignatureData, len(signPIDs))

	for j, signPID := range signPIDs {
		params, _ := tss.NewParameters(tss.S256(), signP2pCtx, signPID, len(signPIDs), newThreshold)
		P_, _ := signing.NewLocalParty(big.NewInt(42), params, signKeys[j], big.NewInt(0), signOutCh, signEndCh, sessionId)
		P := P_.(*signing.LocalParty)
		signParties = append(signParties, P)
		go func(P *signing.LocalParty) {
			if err := P.Start(); err != nil {
				signErrCh <- err
			}
		}(P)
	}

	var signEnded int32
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case err := <-signErrCh:
			common.Logger.Errorf("Error: %s", err)
			assert.FailNow(t, err.Error())
			return

		case msg := <-signOutCh:
			dest := msg.GetTo()
			if dest == nil {
				for _, P := range signParties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, signErrCh)
				}
			} else {
				if dest[0].Index == msg.GetFrom().Index {
					t.Fatalf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
				}
				go updater(signParties[dest[0].Index], msg, signErrCh)
			}

		case signData := <-signEndCh:
			atomic.AddInt32(&signEnded, 1)
			if atomic.LoadInt32(&signEnded) == int32(len(signPIDs)) {
				t.Logf("Signing done. Received sign data from %d participants", signEnded)

				// BEGIN ECDSA verify
				pkX, pkY := signKeys[0].ECDSAPub.X(), signKeys[0].ECDSAPub.Y()
				pk := ecdsa.PublicKey{
					Curve: tss.S256(),
					X:     pkX,
					Y:     pkY,
				}
				ok := ecdsa.Verify(&pk, big.NewInt(42).Bytes(),
					new(big.Int).SetBytes(signData.R),
					new(big.Int).SetBytes(signData.S))

				assert.True(t, ok, "ecdsa verify must pass")
				t.Log("ECDSA signing test done.")
				// END ECDSA verify

				return
			}
		}
	}
}

func TestTooManyParties(t *testing.T) {
	setUp("info")

	pIDs := tss.GenerateTestPartyIDs(MaxParties + 1)
	p2pCtx := tss.NewPeerContext(pIDs)
	oldP2PCtx := tss.NewPeerContext(pIDs)
	params, _ := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, p2pCtx, pIDs[0], MaxParties+1, MaxParties/10,
		len(pIDs), MaxParties/10)
	q := big.Wrap(tss.EC().Params().N)
	sessionId := common.GetBigRandomPositiveInt(q, q.BitLen())

	var err error
	var void keygen.LocalPartySaveData
	_, err = NewLocalParty(params, void, nil, nil, sessionId)
	if !assert.Error(t, err) {
		t.FailNow()
		return
	}

	params, _ = tss.NewReSharingParameters(tss.S256(), tss.NewPeerContext(tss.GenerateTestPartyIDs(MaxParties-1)), p2pCtx,
		pIDs[0], MaxParties+1, MaxParties/10, len(pIDs), MaxParties/10)

	err = nil
	_, err = NewLocalParty(params, void, nil, nil, sessionId)
	if !assert.Error(t, err) {
		t.FailNow()
		return
	}
}

func sortParties(parties tss.SortedPartyIDs, oldParties tss.SortedPartyIDs) tss.SortedPartyIDs {
	newParties := make(tss.SortedPartyIDs, len(parties))
	copy(newParties, oldParties)
	index := len(oldParties)
	for _, party := range parties {
		if !IsParticipant(party, oldParties) {
			newParties[index] = party
			newParties[index].Index = index
			index++
		}
	}
	return newParties
}

func IsParticipant(party *tss.PartyID, parties tss.SortedPartyIDs) bool {
	for _, existingParty := range parties {
		if party.Id == existingParty.Id {
			return true
		}
	}

	return false
}
