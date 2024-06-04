package keygen

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/binance-chain/tss-lib/common"
	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/vss"
	"github.com/binance-chain/tss-lib/test"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/stretchr/testify/assert"
)

const (
	testParticipants = TestParticipants
	testThreshold    = TestThreshold
)

func HandleMessage(t *testing.T, msg tss.Message, parties []*LocalParty, updater func(party tss.Party, msg tss.Message, errCh chan<- *tss.Error), errCh chan *tss.Error) bool {
	dest := msg.GetTo()
	if dest == nil { // broadcast!
		for _, P := range parties {
			if P.PartyID().Index == msg.GetFrom().Index {
				continue
			}
			go updater(P, msg, errCh)
		}
	} else { // point-to-point!
		if dest[0].Index == msg.GetFrom().Index {
			t.Fatalf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
			return true
		}
		go updater(parties[dest[0].Index], msg, errCh)
	}
	return false
}

func InitTheParties(pIDs tss.SortedPartyIDs, p2pCtx *tss.PeerContext, threshold int, fixtures []LocalPartySaveData, outCh chan tss.Message, endCh chan LocalPartySaveData, parties []*LocalParty, errCh chan *tss.Error) ([]*LocalParty, chan *tss.Error) {
	// q := big.Wrap(tss.EC().Params().N)
	// sessionId := common.GetBigRandomPositiveInt(q, q.BitLen())
	// try a small sessionId
	sessionId := new(big.Int).SetInt64(1)
	// init the parties
	for i := 0; i < len(pIDs); i++ {
		var P *LocalParty
		params, _ := tss.NewParameters(tss.EC(), p2pCtx, pIDs[i], len(pIDs), threshold)
		if i < len(fixtures) {
			P_, _ := NewLocalParty(params, outCh, endCh, sessionId, fixtures[i].LocalPreParams)
			P, _ = P_.(*LocalParty)
		} else {
			P_, _ := NewLocalParty(params, outCh, endCh, sessionId)
			P, _ = P_.(*LocalParty)
		}
		parties = append(parties, P)
		go func(P *LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
		}(P)
	}
	return parties, errCh
}

func TryWriteTestFixtureFile(t *testing.T, index int, data LocalPartySaveData) {
	fixtureFileName := makeTestFixtureFilePath(index)

	// fixture file does not already exist?
	// if it does, we won't re-create it here
	fi, err := os.Stat(fixtureFileName)
	if !(err == nil && fi != nil && !fi.IsDir()) {
		fd, err := os.OpenFile(fixtureFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			assert.NoErrorf(t, err, "unable to open fixture file %s for writing", fixtureFileName)
		}
		bz, err := json.Marshal(&data)
		if err != nil {
			t.Fatalf("unable to marshal save data for fixture file %s", fixtureFileName)
		}
		_, err = fd.Write(bz)
		if err != nil {
			t.Fatalf("unable to write to fixture file %s", fixtureFileName)
		}
		t.Logf("Saved a test fixture file for party %d: %s", index, fixtureFileName)
	} else {
		t.Logf("Fixture file already exists for party %d; not re-creating: %s", index, fixtureFileName)
	}
	//
}
func GenECDSA(threshold int, participants int) {
	// tss.SetCurve(elliptic.P256())
	t := &testing.T{}
	fixtures, pIDs, err := LoadKeygenTestFixtures(participants)
	if err != nil {
		common.Logger.Info("No test fixtures were found, so the safe primes will be generated from scratch. This may take a while...",
			err)
		pIDs = tss.GenerateTestPartyIDs(participants)
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
