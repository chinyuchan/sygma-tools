package signing

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/binance-chain/tss-lib/common"
	int2 "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/test"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/stretchr/testify/assert"
)

func TestSaveState(t *testing.T) {
	setUp("debug")
	threshold := testThreshold

	// PHASE: load keygen fixtures
	keys, signPIDs, err := keygen.LoadKeygenTestFixturesRandomSet(testThreshold+1, testParticipants)
	assert.NoError(t, err, "should load keygen fixtures")
	assert.Equal(t, testThreshold+1, len(keys))
	assert.Equal(t, testThreshold+1, len(signPIDs))

	// PHASE: signing
	// use a shuffled selection of the list of parties for this test
	p2pCtx := tss.NewPeerContext(signPIDs)
	parties := make([]*LocalStatefulParty, 0, len(signPIDs))
	stateParties := make([]string, len(signPIDs))

	errCh1 := make(chan *tss.Error, len(signPIDs))
	outCh1 := make(chan tss.Message, len(signPIDs))
	endCh := make(chan common.SignatureData, len(signPIDs))

	// dumpCh := make(chan tss.Message, len(signPIDs))
	q := int2.Wrap(tss.EC().Params().N)
	sessionId := common.GetBigRandomPositiveInt(q, q.BitLen())

	updater := test.SharedPartyUpdater
	keyDerivationDelta := int2.NewInt(0)

	// Save party state
	preAdvanceFunc := func(p tss.StatefulParty, msg tss.ParsedMessage) (bool, *tss.Error) {
		var state string
		var errF *tss.Error

		if state, errF = p.Dehydrate(); errF != nil {
			common.Logger.Errorf("error: %v", errF)
			return false, p.WrapError(errF)
		}
		stateParties[p.PartyID().Index] = state
		// Stop all parties after round 3
		if p.Round().RoundNumber() >= 3 {
			common.Logger.Debugf("party:%v (%p), post-update test intervention", p.PartyID(), &p)
			return false, p.WrapError(errors.New("_force_party_stop_"))
		}
		return false, nil
	}

	// init the parties
	for i := 0; i < len(signPIDs); i++ {
		params, _ := tss.NewParameters(tss.S256(), p2pCtx, signPIDs[i], len(signPIDs), threshold)
		P_, errP := NewLocalStatefulParty(int2.NewInt(42), params, keys[i], keyDerivationDelta, outCh1, endCh,
			preAdvanceFunc, sessionId)
		if errP != nil {
			t.Errorf("error %v", errP)
			t.FailNow()
		}
		P := P_.(*LocalStatefulParty)
		parties = append(parties, P)
		go func(P *LocalStatefulParty) {
			if errS := P.Start(); errS != nil {
				errCh1 <- errS
			}
		}(P)
	}

	var ended int32
	var endedFirstPart int32

signingFirstPart:
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case err0 := <-errCh1:
			if strings.Compare("_force_party_stop_", err0.Cause().Error()) == 0 {
				atomic.AddInt32(&endedFirstPart, 1)
				if atomic.LoadInt32(&endedFirstPart) == int32(len(signPIDs)) {
					break signingFirstPart
				} else {
					continue
				}
			}
			common.Logger.Errorf("Error: %s", err0)
			assert.FailNow(t, err0.Error())
			break signingFirstPart

		case msg := <-outCh1:
			dest := msg.GetTo()
			if dest == nil {
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, errCh1)
				}
			} else {
				if dest[0].Index == msg.GetFrom().Index {
					t.Fatalf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
				}
				go updater(parties[dest[0].Index], msg, errCh1)
			}
		}
	}

	time.Sleep(3 * time.Second)
	common.Logger.Debug("Second part of the unit test ----------------------------------------------")
	// Second part

	parties = make([]*LocalStatefulParty, 0, len(signPIDs))
	errCh2 := make(chan *tss.Error, len(signPIDs))
	outCh2 := make(chan tss.Message, len(signPIDs))

	nilAdvanceFunc := func(p tss.StatefulParty, msg tss.ParsedMessage) (bool, *tss.Error) {
		return false, nil
	}

	// init the parties
	for i := 0; i < len(signPIDs); i++ {
		params, _ := tss.NewParameters(tss.S256(), p2pCtx, signPIDs[i], len(signPIDs), threshold)

		P_, errP := NewLocalStatefulParty(int2.NewInt(42), params, keys[i], keyDerivationDelta, outCh2, endCh,
			nilAdvanceFunc, sessionId)
		if errP != nil {
			t.Errorf("error %v", errP)
			t.FailNow()
		}
		P := P_.(*LocalStatefulParty)
		_, errH := P.Hydrate(stateParties[i])
		if errH != nil {
			assert.NoError(t, errH, "there should be no error hydrating")
		}
		parties = append(parties, P)
		go func(P *LocalStatefulParty) {
			if errS := P.Restart(4, ""); errS != nil {
				errCh2 <- errS
			}
		}(P)
	}

signingSecondPart:
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case err2 := <-errCh2:
			common.Logger.Errorf("Error: %s", err2)
			assert.FailNow(t, err2.Error())
			break signingSecondPart

		case msg := <-outCh2:
			dest := msg.GetTo()
			if dest == nil {
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, errCh2)
				}
			} else {
				if dest[0].Index == msg.GetFrom().Index {
					t.Fatalf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
				}
				go updater(parties[dest[0].Index], msg, errCh2)
			}
		case <-endCh:
			atomic.AddInt32(&ended, 1)
			if atomic.LoadInt32(&ended) == int32(len(signPIDs)) {
				t.Logf("Done. Received signature data from %d participants", ended)
				R := parties[0].temp.BigR
				// r := parties[0].temp.Rx
				// fmt.Printf("sign result: R(%s, %s), r=%s\n", R.X().String(), R.Y().String(), r.String())

				modN := int2.ModInt(int2.Wrap(tss.S256().Params().N))

				// BEGIN check s correctness
				sumS := int2.NewInt(0)
				for _, p := range parties {
					sumS = modN.Add(sumS, p.temp.SigmaShare)
				}
				// fmt.Printf("S: %s\n", sumS.String())
				// END check s correctness

				// BEGIN ECDSA verify
				pkX, pkY := keys[0].ECDSAPub.X(), keys[0].ECDSAPub.Y()
				pk := ecdsa.PublicKey{
					Curve: tss.EC(),
					X:     pkX,
					Y:     pkY,
				}
				ok := ecdsa.Verify(&pk, int2.NewInt(42).Bytes(), R.X(), sumS)
				assert.True(t, ok, "ecdsa verify must pass")
				t.Log("ECDSA signing test done.")
				// END ECDSA verify

				break signingSecondPart
			}
		}
	}
}
