package keygen

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/binance-chain/tss-lib/common"
	int2 "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/vss"
	"github.com/binance-chain/tss-lib/test"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/stretchr/testify/assert"
)

func TestSaveState(t *testing.T) {
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
	parties := make([]*LocalStatefulParty, 0, len(pIDs))
	stateParties := make([]string, len(pIDs))
	errCh1 := make(chan *tss.Error, len(pIDs))
	outCh1 := make(chan tss.Message, len(pIDs))
	endCh := make(chan LocalPartySaveData, len(pIDs))

	updater := test.SharedPartyUpdater

	startGR := runtime.NumGoroutine()

	q := int2.Wrap(tss.EC().Params().N)
	sessionId := common.GetBigRandomPositiveInt(q, q.BitLen())

	// Save and reload party
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
	for i := 0; i < len(pIDs); i++ {
		var P *LocalStatefulParty
		params, _ := tss.NewParameters(tss.EC(), p2pCtx, pIDs[i], len(pIDs), threshold)
		if i < len(fixtures) {
			P_, _ := NewLocalStatefulParty(params, outCh1, endCh, preAdvanceFunc, sessionId, fixtures[i].LocalPreParams)
			P, _ = P_.(*LocalStatefulParty)
		} else {
			P_, _ := NewLocalStatefulParty(params, outCh1, endCh, preAdvanceFunc, sessionId)
			P, _ = P_.(*LocalStatefulParty)
		}
		parties = append(parties, P)
		go func(P *LocalStatefulParty) {
			if err := P.Start(); err != nil {
				errCh1 <- err
			}
		}(P)
	}

	var endedFirstPart int32

keygenFirstPart:
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case errC := <-errCh1:
			if strings.Compare("_force_party_stop_", errC.Cause().Error()) == 0 {
				atomic.AddInt32(&endedFirstPart, 1)
				if atomic.LoadInt32(&endedFirstPart) == int32(len(pIDs)) {
					break keygenFirstPart
				} else {
					continue
				}
			}
			common.Logger.Errorf("Error: %s", errC)
			assert.FailNow(t, errC.Error())
			break keygenFirstPart

		case msg := <-outCh1:
			dest := msg.GetTo()
			if dest == nil { // broadcast!
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, errCh1)
				}
			} else { // point-to-point!
				if dest[0].Index == msg.GetFrom().Index {
					t.Fatalf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
				}
				go updater(parties[dest[0].Index], msg, errCh1)
			}
		}
	}

	time.Sleep(3 * time.Second)
	// Second part
	noActionFunc := func(p tss.StatefulParty, msg tss.ParsedMessage) (bool, *tss.Error) {
		return true, nil
	}

	parties = make([]*LocalStatefulParty, 0, len(pIDs))
	errCh2 := make(chan *tss.Error, len(pIDs))
	outCh2 := make(chan tss.Message, len(pIDs))
	// init the parties
	for i := 0; i < len(pIDs); i++ {
		var P *LocalStatefulParty
		params, _ := tss.NewParameters(tss.EC(), p2pCtx, pIDs[i], len(pIDs), threshold)
		if i < len(fixtures) {
			P_, _ := NewLocalStatefulParty(params, outCh2, endCh, noActionFunc, sessionId, fixtures[i].LocalPreParams)
			P, _ = P_.(*LocalStatefulParty)
		} else {
			P_, _ := NewLocalStatefulParty(params, outCh2, endCh, noActionFunc, sessionId)
			P, _ = P_.(*LocalStatefulParty)
		}
		_, errH := P.Hydrate(stateParties[i])
		if errH != nil {
			assert.NoError(t, errH, "there should be no error hydrating")
		}
		parties = append(parties, P)
		go func(P *LocalStatefulParty) {
			if errR := P.Restart(4, ""); errR != nil {
				errCh2 <- errR
			}
		}(P)
	}

	var ended int32
keygenSecondPart:
	for {
		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
		select {
		case errC2 := <-errCh2:
			common.Logger.Errorf("Error: %s", errC2)
			assert.FailNow(t, errC2.Error())
			break keygenSecondPart

		case msg := <-outCh2:
			dest := msg.GetTo()
			if dest == nil { // broadcast!
				for _, P := range parties {
					if P.PartyID().Index == msg.GetFrom().Index {
						continue
					}
					go updater(P, msg, errCh2)
				}
			} else { // point-to-point!
				if dest[0].Index == msg.GetFrom().Index {
					t.Fatalf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
				}
				go updater(parties[dest[0].Index], msg, errCh2)
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
				u := int2.NewInt(0)
				for j, Pj := range parties {
					pShares := make(vss.Shares, 0)
					for j2, P := range parties {
						var share *int2.Int
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
						badShares[len(badShares)-1].Share.Set(int2.NewInt(0))
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
					u = new(int2.Int).Add(u, uj)
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
					assert.NotNil(t, Pj.data.ECDSAPub, "ECDSAPub must not be nil")
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

				break keygenSecondPart
			}
		}
	}
}
