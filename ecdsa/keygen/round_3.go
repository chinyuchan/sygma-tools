// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package keygen

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/binance-chain/tss-lib/common/hash"
	big "github.com/binance-chain/tss-lib/common/int"
	zkpprm "github.com/binance-chain/tss-lib/crypto/zkp/prm"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	zkpfac "github.com/binance-chain/tss-lib/crypto/zkp/fac"
	zkpmod "github.com/binance-chain/tss-lib/crypto/zkp/mod"
	zkpsch "github.com/binance-chain/tss-lib/crypto/zkp/sch"
	"github.com/binance-chain/tss-lib/tss"
)

const (
	paillierModulusLen = 2048
	NBitLen            = 2048
)

func newRound3(params *tss.Parameters, save *LocalPartySaveData, temp *localTempData, out chan<- tss.Message, end chan<- LocalPartySaveData) tss.Round {
	return &round3{&round2{&round1{
		&base{params, save, temp, out, end, make([]bool, len(params.Parties().IDs())), false, 3}}}}
}

func (round *round3) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 3
	round.started = true
	round.resetOK()

	i := round.PartyID().Index
	round.ok[i] = true

	var err error

	errsCh := make(chan *tss.Error, (len(round.Parties().IDs())-1)*10) // sufficient buffer
	rid := round.temp.ridi
	wg := sync.WaitGroup{}
	q := big.Wrap(round.EC().Params().N)
	modQ := big.ModInt(q)
	𝜅 := uint(128)
	twoTo8𝜅 := new(big.Int).Lsh(big.NewInt(1), 8*𝜅)
	sid := hash.SHA256i(append(round.Parties().IDs().Keys(), big.Wrap(tss.EC().Params().N),
		big.Wrap(tss.EC().Params().P), big.Wrap(tss.EC().Params().B),
		big.Wrap(tss.EC().Params().Gx), big.Wrap(tss.EC().Params().Gy))...)

	idG := crypto.ScalarBaseMult(round.EC(), big.NewInt(1))
	𝜌 := round.temp.𝜌ᵢ

	concurrency := runtime.NumCPU()
	if concurrency > len(round.Parties().IDs()) {
		concurrency = len(round.Parties().IDs())
	}
	common.Logger.Debugf(
		"%s Setting up PRM verification with concurrency level %d",
		round.PartyID(),
		concurrency,
	)
	prmVerifier, errV := zkpprm.NewPrmProofVerifier(concurrency)
	if errV != nil {
		return round.WrapError(fmt.Errorf("zkpprm init"))
	}

	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}
		rid = modQ.Add(rid, round.temp.r2msgRidj[j])
		𝜌 = big.NewInt(0).Add(𝜌, round.temp.rref2msg𝜌j[j])

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()
			// Fig 5. Round 3.1
			keygenListToHash, errF := crypto.FlattenECPoints(round.temp.r2msgVss[j])
			if errF != nil {
				errsCh <- round.WrapError(errF, Pj)
				return
			}
			keygenListToHash = append(keygenListToHash, []*big.Int{round.temp.r2msgSid[j], big.NewInt(uint64(j)),
				round.temp.r2msgRidj[j],
				round.temp.r2msgXKeygenj[j].X(), round.temp.r2msgXKeygenj[j].Y(),
				round.temp.r2msgAKeygenj[j].X(), round.temp.r2msgAKeygenj[j].Y(), round.temp.r2msgUj[j]}...)

			VjKeygen := hash.SHA256i(keygenListToHash...)
			if VjKeygen.Cmp(round.temp.r1msgVjKeygen[j]) != 0 {
				errsCh <- round.WrapError(errors.New("verify hash failed"), Pj)
				return
			}
		}(j, Pj)

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()
			if round.save.NTildej[j].BitLen() < NBitLen {
				errsCh <- round.WrapError(errors.New("N too small"), Pj)
				return
			}
		}(j, Pj)

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()
			if round.save.PaillierPKs[j].N.BitLen() < paillierModulusLen {
				errsCh <- round.WrapError(errors.New("paillier modulus too small"), Pj)
				return
			}
		}(j, Pj)

		// Refresh:
		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()
			if round.temp.rref2msgNj[j].Cmp(twoTo8𝜅) == -1 {
				errsCh <- round.WrapError(errors.New(" Nj is too small"), Pj)
				return
			}
		}(j, Pj)

		Xkj := round.temp.rref2msgXj[j]
		ᴨkXkj := crypto.NewECPointNoCurveCheck(round.EC(), idG.X(), idG.Y())
		for _, X := range Xkj { // for each k
			if ᴨkXkj, err = ᴨkXkj.Add(X); err != nil {
				errsCh <- round.WrapError(errors.New(" Xj product"), Pj)
			}
		}

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			Nj, sj, tj := round.temp.rref2msgNj[j], round.temp.rref2msgsj[j], round.temp.rref2msgtj[j]
			ssid := hash.SHA256i([]*big.Int{sid /*round.temp.r2msgRidj[j],*/, Nj, sj, tj, round.temp.sessionId}...)
			nonce := big.NewInt(0).Add(ssid, big.NewInt(uint64(j)))
			if nonce.BitLen() < zkpprm.MinBitLen {
				nonce = new(big.Int).Lsh(nonce, uint(zkpprm.MinBitLen-nonce.BitLen()))
			}
			prmVerifier.VerifyWithNonce(round.temp.rref2msgpf𝜓j[j], sj, tj, Nj, nonce, func(isValid bool) {
				defer wg.Done()
				if !isValid {
					/* common.Logger.Debugf("party %v r3 err Pj: %v, proof: %v, Ni: %v, si: %v, nonce: %v", round.PartyID(),
						Pj, zkpprm.FormatProofPrm(round.temp.rref2msgpf𝜓j[j]), common.FormatBigInt(Nj),
						common.FormatBigInt(sj), common.FormatBigInt(nonce),
					) */
					errsCh <- round.WrapError(errors.New("failed prm proof"), Pj)
					return
				}
			})

		}(j, Pj)

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()
			Nj, sj, tj := round.temp.rref2msgNj[j], round.temp.rref2msgsj[j], round.temp.rref2msgtj[j]
			𝜓array := round.temp.rref2msgpf𝜓j[j].ToIntArray()
			XjPoints, errX := crypto.FlattenECPoints(round.temp.rref2msgXj[j])
			if errX != nil {
				errsCh <- round.WrapError(errors.New("flattening error"), Pj)
				return
			}
			AjPoints, errA := crypto.FlattenECPoints(round.temp.rref2msgAj[j])
			if errA != nil {
				errsCh <- round.WrapError(errors.New("flattening error"), Pj)
				return
			}

			h := append([]*big.Int{round.temp.rref2msgSsid[j], big.NewInt(uint64(j)), round.temp.rref2msgYj[j].X(),
				round.temp.rref2msgYj[j].Y(),
				round.temp.rref2msgBj[j].X(), round.temp.rref2msgBj[j].Y(), Nj, sj, tj,
				round.temp.rref2msg𝜌j[j], round.temp.r2msgUj[j]}, 𝜓array...)
			h = append(h, XjPoints...)
			h = append(h, AjPoints...)
			Vj := hash.SHA256i(h...)
			if same := round.temp.rref1msgVjKeyRefresh[j].Cmp(Vj) == 0; !same {
				errsCh <- round.WrapError(errors.New("different V hashes"), Pj)
				return
			}
		}(j, Pj)
	}

	wg.Wait()
	culprits := make([]*tss.PartyID, 0)
outer:
	for {
		select {
		case err := <-errsCh:
			culprits = append(culprits, err.Culprits()...)
		default:
			break outer
		}
	}
	if len(culprits) > 0 {
		return round.WrapError(errors.New("round3: failed stage 3.1"), culprits...)
	}

	// Fig 5. Round 3.2
	xi := new(big.Int).Set(round.temp.shares[i].Share)
	Xi := crypto.ScalarBaseMult(round.EC(), xi)
	sidirid := modQ.Add(modQ.Add(round.temp.sid, big.NewInt(uint64(i))), rid)
	nonceKG := modQ.Add(sidirid, round.temp.sessionId)
	if nonceKG.BitLen() < round.EC().Params().N.BitLen() {
		nonceKG = new(big.Int).Lsh(nonceKG, uint(round.EC().Params().N.BitLen()-nonceKG.BitLen()))
	}
	𝜓Schi, err := zkpsch.NewProofGivenAlpha(Xi, xi, round.temp.τKeygen, nonceKG)
	if err != nil {
		return round.WrapError(errors.New("create proofSch failed"))
	}
	/* common.Logger.Debugf("party %v r3 sch 𝜓[i=%v]: %v, nonceKG: %v", round.PartyID(),
		i, zkpsch.FormatProofSch(𝜓Schi), common.FormatBigInt(nonceKG),
	) */

	// Refresh:
	modN := big.ModInt(big.Wrap(round.EC().Params().N))
	nonce := modN.Add(modN.Add(round.temp.ssid, 𝜌), big.NewInt(uint64(i)))
	if nonce.BitLen() < round.EC().Params().N.BitLen() {
		nonce = new(big.Int).Lsh(nonce, uint(round.EC().Params().N.BitLen()-nonce.BitLen()))
	}

	𝜓Modi, errP := zkpmod.NewProof(q, round.save.LocalPreParams.NTildei,
		common.PrimeToSafePrime(round.save.LocalPreParams.P),
		common.PrimeToSafePrime(round.save.LocalPreParams.Q), nonce)
	if errP != nil {
		return round.WrapError(fmt.Errorf("zkpmod failed"))
	}

	/* common.Logger.Debugf("party %v r3 KR mod 𝜓[i=%v]: %v, nonce: %v", round.PartyID(),
		i, zkpmod.FormatProofMod(𝜓Modi), common.FormatBigInt(nonce),
	) */

	ᴨi, errPi := zkpsch.NewProofGivenAlpha(round.temp.Yᵢ, round.temp.yᵢ, round.temp.𝜏KeyRefresh, nonce)
	if errPi != nil {
		return round.WrapError(fmt.Errorf("zkpsch failed"))
	}
	/* common.Logger.Debugf("party %v r3 ᴨ[i=%v]: %v, nonce: %v", round.PartyID(),
		i, zkpsch.FormatProofSch(ᴨi), common.FormatBigInt(nonce),
	) */

	errsCh = make(chan *tss.Error, len(round.Parties().IDs())-1)
	wg = sync.WaitGroup{}
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}
		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()

			𝜙ji, errF := zkpfac.NewProofGivenNonce(round.EC(), &round.save.PaillierSK.PublicKey, round.save.LocalPreParams.NTildei,
				round.save.LocalPreParams.H1i, round.save.LocalPreParams.H2i,
				common.PrimeToSafePrime(round.save.LocalPreParams.P), common.PrimeToSafePrime(round.save.LocalPreParams.Q), nonce)
			if errF != nil {
				errsCh <- round.WrapError(errors.New("create proofPrm failed"))
				return
			}
			/* verif := 𝜙ji.Verify(round.EC(), &round.save.PaillierSK.PublicKey, round.save.LocalPreParams.NTildei,
				round.save.LocalPreParams.H1i, round.save.LocalPreParams.H2i, nonce)
			common.Logger.Debugf("party:%v r3, Pj: %v, 𝜙_[j=%v],[i=%v]: %v, nonce[%v]: %v, ssid: %v, 𝜌: %v"+
				", verif? %v",
				round.PartyID(), Pj,
				j, i, zkpfac.FormatProofFac(𝜙ji),
				i, common.FormatBigInt(nonce),
				common.FormatBigInt(round.temp.ssid), common.FormatBigInt(𝜌), verif) */

			// "vss" as in Feldman's verifiable secret sharing
			Cvssji, randomnessCvssji, errEv := round.save.PaillierPKs[j].EncryptAndReturnRandomness(round.temp.shares[j].Share)
			if errEv != nil {
				errsCh <- round.WrapError(errors.New("encryption error"), Pj)
				return
			}
			// "zero" as un zero sum, per Figure 6, Round 1.
			Czeroji, randomnessCzeroji, errE0 := round.save.PaillierPKs[j].EncryptAndReturnRandomness(round.temp.xⁿᵢ[j])
			if errE0 != nil {
				errsCh <- round.WrapError(errors.New("encryption error"), Pj)
				return
			}
			𝜓jᵢ, errS := zkpsch.NewProofGivenAlpha(round.temp.XiRefreshList[j], round.temp.xⁿᵢ[j], round.temp.𝜏js[j], nonce)
			if errS != nil {
				errsCh <- round.WrapError(fmt.Errorf("error with zkpsch"))
				return
			}
			/* common.Logger.Debugf("party:%v r3, Pj: %v, 𝜓^[j=%v]_[i=%v]: %v, X^[j=%v]_[i=%v]: %v, nonce[%v]: %v"+
			", ssid: %v, 𝜌: %v",
			round.PartyID(), Pj,
			j, i, zkpsch.FormatProofSch(𝜓jᵢ),
			j, i, crypto.FormatECPoint(round.temp.XiRefreshList[j]),
			i, common.FormatBigInt(nonce),
			common.FormatBigInt(round.temp.ssid), common.FormatBigInt(𝜌))
			*/
			r3msg := NewKGRound3Message(round.temp.sessionId, Pj, round.PartyID(), round.temp.sid, 𝜓Schi,
				Cvssji, randomnessCvssji,
				// refresh:
				round.temp.ssid, 𝜓Modi, 𝜙ji, ᴨi, Czeroji, randomnessCzeroji, 𝜓jᵢ)
			round.out <- r3msg
		}(j, Pj)
	}
	wg.Wait()
	select {
	case err := <-errsCh:
		return err
	default:
		break
	}
	round.temp.𝜌 = 𝜌
	round.temp.rid = rid
	return nil
}

func (round *round3) CanAccept(msg tss.ParsedMessage) bool {
	if _, ok := msg.Content().(*KGRound3Message); ok {
		return !msg.IsBroadcast()
	}
	return false
}

func (round *round3) Update() (bool, *tss.Error) {
	for j, msg := range round.temp.r3msgxij {
		if round.ok[j] {
			continue
		}
		if msg == nil {
			return false, nil
		}
		round.ok[j] = true
	}
	return true, nil
}

func (round *round3) NextRound() tss.Round {
	round.started = false
	return &round4{round}
}
