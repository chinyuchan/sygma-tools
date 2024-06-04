package resharing_test

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/binance-chain/tss-lib/common"
	int2 "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	. "github.com/binance-chain/tss-lib/ecdsa/resharing"
	"github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/binance-chain/tss-lib/test"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/stretchr/testify/assert"
)

func TestSaveState(t *testing.T) {
	setUp("info")

	threshold, newThreshold := testThreshold, testThreshold

	// PHASE: load keygen fixtures
	firstPartyIdx, extraParties := 0, 1 // extra can be 0 to N-first
	oldKeys, oldPIDs, err := keygen.LoadKeygenTestFixtures(testThreshold+1+extraParties+firstPartyIdx, firstPartyIdx)
	assert.NoError(t, err, "should load keygen fixtures")

	// PHASE: resharing
	oldP2PCtx := tss.NewPeerContext(oldPIDs)
	// init the new parties; re-use the fixture pre-params for speed
	fixtures, _, err := keygen.LoadKeygenTestFixtures(testParticipants)
	if err != nil {
		common.Logger.Info("No test fixtures were found, so the safe primes will be generated from scratch. This may take a while...")
	}
	newPIDs := tss.GenerateTestPartyIDs(testParticipants)
	newP2PCtx := tss.NewPeerContext(newPIDs)
	newPCount := len(newPIDs)

	oldCommittee := make([]*LocalStatefulParty, 0, len(oldPIDs))
	newCommittee := make([]*LocalStatefulParty, 0, newPCount)
	bothCommitteesPax := len(oldCommittee) + len(newCommittee)

	stateOldCommittee := make(map[tss.PartyID]string, len(oldPIDs))
	stateNewCommittee := make(map[tss.PartyID]string, len(oldPIDs))
	oldPartySignalled := make(map[tss.PartyID]bool, len(oldPIDs))
	newPartySignalled := make(map[tss.PartyID]bool, len(oldPIDs))

	// Save and reload party
	mutexOld, mutexNew := sync.RWMutex{}, sync.RWMutex{}
	q := int2.Wrap(tss.EC().Params().N)
	sessionId := common.GetBigRandomPositiveInt(q, q.BitLen())

	preAdvanceOldFunc := func(p tss.StatefulParty, msg tss.ParsedMessage) (bool, *tss.Error) {
		var state string
		var errF *tss.Error

		if state, errF = p.Dehydrate(); errF != nil {
			common.Logger.Errorf("error: %v", errF)
			return false, p.WrapError(errF)
		}
		mutexOld.Lock()
		_, partySeen := oldPartySignalled[*p.PartyID()]
		stateOldCommittee[*p.PartyID()] = state
		mutexOld.Unlock()
		// Stop all parties after round 1
		if p.Round().RoundNumber() >= 1 && !partySeen {
			mutexOld.Lock()
			oldPartySignalled[*p.PartyID()] = true
			mutexOld.Unlock()
			return false, p.WrapError(errors.New("_force_party_stop_"))
		}
		if p.Round().RoundNumber() >= 2 {
			time.Sleep(1 * time.Second)
			// do nothing
			return false, p.WrapError(errors.New("_silent_nop_"))
		}
		return false, nil
	}

	preAdvanceNewFunc := func(p tss.StatefulParty, msg tss.ParsedMessage) (bool, *tss.Error) {
		var state string
		var errF *tss.Error

		if state, errF = p.Dehydrate(); errF != nil {
			common.Logger.Errorf("error: %v", errF)
			return false, p.WrapError(errF)
		}
		mutexNew.Lock()
		_, partySeen := newPartySignalled[*p.PartyID()]
		stateNewCommittee[*p.PartyID()] = state
		mutexNew.Unlock()
		// Stop all parties after round 2
		if p.Round().RoundNumber() >= 2 && !partySeen {
			mutexNew.Lock()
			newPartySignalled[*p.PartyID()] = true
			mutexNew.Unlock()
			return false, p.WrapError(errors.New("_force_party_stop_"))
		}
		if p.Round().RoundNumber() >= 3 {
			time.Sleep(1 * time.Second)
			// do nothing
			return false, p.WrapError(errors.New("_silent_nop_"))
		}
		return false, nil
	}

	errCh1 := make(chan *tss.Error, bothCommitteesPax)
	outCh1 := make(chan tss.Message, bothCommitteesPax)
	endCh := make(chan keygen.LocalPartySaveData, bothCommitteesPax)

	var save keygen.LocalPartySaveData

	updater := test.SharedPartyUpdater

	common.Logger.Debug("init the old parties")
	// init the old parties first
	for j, pID := range oldPIDs {
		pID.Moniker = pID.Moniker + "_old"
		params, _ := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, testParticipants, threshold, newPCount, newThreshold)
		P_, _ := NewLocalStatefulParty(params, oldKeys[j], outCh1, endCh, preAdvanceOldFunc, sessionId) // discard old key data
		P := P_.(*LocalStatefulParty)
		common.Logger.Debugf("old party: %v, keyInt:%v", pID, common.FormatBigInt(pID.KeyInt()))
		oldCommittee = append(oldCommittee, P)
	}
	common.Logger.Debug("init the new parties")
	// init the new parties
	for j, pID := range newPIDs {
		pID.Moniker = pID.Moniker + "_new"
		params, _ := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, testParticipants, threshold, newPCount, newThreshold)
		save = keygen.NewLocalPartySaveData(newPCount)
		if j < len(fixtures) && len(newPIDs) <= len(fixtures) {
			save.LocalPreParams = fixtures[j].LocalPreParams
		}
		P_, _ := NewLocalStatefulParty(params, save, outCh1, endCh, preAdvanceNewFunc, sessionId)
		P := P_.(*LocalStatefulParty)
		common.Logger.Debugf("new party: %v, keyInt:%v", pID, common.FormatBigInt(pID.KeyInt()))
		newCommittee = append(newCommittee, P)
	}

	common.Logger.Debug("start the new parties; they will wait for messages")
	// start the new parties; they will wait for messages
	for _, P := range newCommittee {
		go func(P *LocalStatefulParty) {
			if err := P.Start(); err != nil {
				errCh1 <- err
			}
		}(P)
	}
	common.Logger.Debug("start the old parties; they will send messages")
	// start the old parties; they will send messages
	for _, P := range oldCommittee {
		go func(P *LocalStatefulParty) {
			if err := P.Start(); err != nil {
				errCh1 <- err
			}
		}(P)
	}

	newKeys := make([]keygen.LocalPartySaveData, len(newCommittee))
	endedOldCommittee := 0
	var reSharingEndedFirstPart int32

resharingFirstPart:
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case errC := <-errCh1:
			if strings.Compare("_force_party_stop_", errC.Cause().Error()) == 0 {
				atomic.AddInt32(&reSharingEndedFirstPart, 1)
				if atomic.LoadInt32(&reSharingEndedFirstPart) == int32(len(oldCommittee)+len(newCommittee)) {
					break resharingFirstPart
				} else {
					continue
				}
			} else if strings.Compare("_silent_nop_", errC.Cause().Error()) == 0 {
				// common.Logger.Debug("_silent_nop_")
			} else {
				common.Logger.Errorf("Error: %s", errC)
				assert.FailNow(t, errC.Error())
				break resharingFirstPart
			}

		case msg := <-outCh1:
			dest := msg.GetTo()
			if dest == nil {
				t.Fatal("did not expect a msg to have a nil destination during resharing")
			}
			if msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, destP := range dest[:len(oldCommittee)] {
					go updater(oldCommittee[destP.Index], msg, errCh1)
				}
			}
			if !msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, destP := range dest {
					go updater(newCommittee[destP.Index], msg, errCh1)
				}
			}
		}
	}

	common.Logger.Debug("I'll sleep -------------------------------------------------------")
	time.Sleep(2 * time.Second)
	common.Logger.Debug("Second part -------------------------------------------------------")
	// Second part
	noActionFunc := func(p tss.StatefulParty, msg tss.ParsedMessage) (bool, *tss.Error) {
		return true, nil
	}

	oldCommittee = make([]*LocalStatefulParty, 0, len(oldPIDs))
	newCommittee = make([]*LocalStatefulParty, 0, newPCount)
	errCh2 := make(chan *tss.Error, bothCommitteesPax)
	outCh2 := make(chan tss.Message, bothCommitteesPax)

	// init the parties again
	// init the old parties first
	common.Logger.Debug("init the parties again - init the old parties first")
	for j, pID := range oldPIDs {
		// pID.Moniker = pID.Moniker + "*"
		params, _ := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, testParticipants, threshold, newPCount, newThreshold)
		P_, _ := NewLocalStatefulParty(params, oldKeys[j], outCh2, endCh, noActionFunc, sessionId) // discard old key data
		P := P_.(*LocalStatefulParty)
		common.Logger.Debugf("old party*: %v, keyInt:%v", pID, common.FormatBigInt(pID.KeyInt()))
		oldCommittee = append(oldCommittee, P)
	}

	common.Logger.Debug("init the parties again - init the new parties")
	// init the new parties
	for j, pID := range newPIDs {
		// pID.Moniker = pID.Moniker + "*"
		params, _ := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, testParticipants, threshold, newPCount, newThreshold)
		save = keygen.NewLocalPartySaveData(newPCount)
		if j < len(fixtures) && len(newPIDs) <= len(fixtures) {
			save.LocalPreParams = fixtures[j].LocalPreParams
		}
		P_, _ := NewLocalStatefulParty(params, save, outCh2, endCh, noActionFunc, sessionId)
		P := P_.(*LocalStatefulParty)
		common.Logger.Debugf("new party*: %v, keyInt:%v", pID, common.FormatBigInt(pID.KeyInt()))
		newCommittee = append(newCommittee, P)
	}

	common.Logger.Debug("restart the old parties; they will send messages")
	// restart the old parties; they will send messages
	for _, P := range oldCommittee {
		go func(P *LocalStatefulParty) {
			mutexOld.Lock()
			if errR := P.Restart(3, stateOldCommittee[*P.PartyID()]); errR != nil {
				errCh1 <- errR
			}
			mutexOld.Unlock()
		}(P)
	}
	common.Logger.Debug("restart the new parties; they will wait for messages")
	// restart the new parties; they will wait for messages
	for _, P := range newCommittee {
		go func(P *LocalStatefulParty) {
			mutexNew.Lock()
			if errR := P.Restart(3, stateNewCommittee[*P.PartyID()]); errR != nil {
				errCh2 <- errR
			}
			mutexNew.Unlock()
		}(P)
	}

	endedOldCommittee = 0
	var reSharingEndedSecondPart int32

resharingSecondPart:
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case errC2 := <-errCh2:
			common.Logger.Errorf("Error: %s", errC2)
			assert.FailNow(t, errC2.Error())
			return

		case msg := <-outCh2:
			dest := msg.GetTo()
			if dest == nil {
				t.Fatal("did not expect a msg to have a nil destination during resharing")
			}
			if msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, destP := range dest[:len(oldCommittee)] {
					go updater(oldCommittee[destP.Index], msg, errCh2)
				}
			}
			if !msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, destP := range dest {
					go updater(newCommittee[destP.Index], msg, errCh2)
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
			atomic.AddInt32(&reSharingEndedSecondPart, 1)
			if atomic.LoadInt32(&reSharingEndedSecondPart) == int32(len(oldCommittee)+len(newCommittee)) {
				assert.Equal(t, len(oldCommittee), endedOldCommittee)
				t.Logf("Resharing done. Reshared %d participants", reSharingEndedSecondPart)

				// xj tests: BigXj == xj*G
				for j, key := range newKeys {
					// xj test: BigXj == xj*G
					xj := key.Xi
					gXj := crypto.ScalarBaseMult(tss.S256(), xj)
					BigXj := key.BigXj[j]
					assert.True(t, BigXj.Equals(gXj), "ensure BigX_j == g^x_j")
				}

				// more verification of signing is implemented within local_party_test.go of keygen package
				break resharingSecondPart
			}
		}
	}

	common.Logger.Debug("signing ----------------------------------------------")

	// PHASE: signing
	signKeys, signPIDs := newKeys, newPIDs
	signP2pCtx := tss.NewPeerContext(signPIDs)
	signParties := make([]*signing.LocalParty, 0, len(signPIDs))

	signErrCh := make(chan *tss.Error, len(signPIDs))
	signOutCh := make(chan tss.Message, len(signPIDs))
	signEndCh := make(chan common.SignatureData, len(signPIDs))

	for j, signPID := range signPIDs {
		params, _ := tss.NewParameters(tss.S256(), signP2pCtx, signPID, len(signPIDs), newThreshold)
		P_, _ := signing.NewLocalParty(int2.NewInt(42), params, signKeys[j], int2.NewInt(0), signOutCh, signEndCh, sessionId)
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
					new(int2.Int).SetBytes(signData.R),
					new(int2.Int).SetBytes(signData.S))

				assert.True(t, ok, "ecdsa verify must pass")
				t.Log("ECDSA signing test done.")
				// END ECDSA verify

				return
			}
		}
	}
}
