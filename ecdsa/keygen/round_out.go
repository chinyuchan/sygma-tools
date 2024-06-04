// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package keygen

import (
	"errors"
	"sync"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/tss"
)

func newRoundout(params *tss.Parameters, save *LocalPartySaveData, temp *localTempData, out chan<- tss.Message, end chan<- LocalPartySaveData) tss.Round {
	return &roundout{&round4{&round3{&round2{&round1{
		&base{params, save, temp, out, end, make([]bool, len(params.Parties().IDs())), false, 5}}}}}}
}

func (round *roundout) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 5
	round.started = true
	round.resetOK()

	i := round.PartyID().Index
	round.ok[i] = true

	wg := sync.WaitGroup{}
	errChs := make(chan *tss.Error, len(round.Parties().IDs())-1)
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}

		wg.Add(1)
		go func(j int, Pj *tss.PartyID) {
			defer wg.Done()
			if round.temp.sessionId == nil {
				errChs <- round.WrapError(errors.New("sessionId not set"))
			}
			if round.temp.r4msgAbortingj[j] {
				common.Logger.Errorf("party %v, reporting party: %v, alleged culprit:%v, 𝜇: %v, C^i_j: %v"+
					", x^i_j: %v",
					round.PartyID(),
					Pj, round.Parties().IDs()[round.temp.r4msgCulpritPj[j]], round.temp.r4msg𝜇j[j].String(),
					round.temp.r4msgCji[j].String(), round.temp.r4msgxji[j].String(),
				)
				errChs <- round.WrapError(errors.New("g^(x^i_j) != X^i_j, equality required -- verify 𝜇"),
					round.Parties().IDs()[round.temp.r4msgCulpritPj[j]])
			}
		}(j, Pj)
	}
	wg.Wait()
	close(errChs)
	culprits := make([]*tss.PartyID, 0)
	for err := range errChs {
		culprits = append(culprits, err.Culprits()...)
	}
	if len(culprits) > 0 {
		return round.WrapError(errors.New("round_out: error"), culprits...)
	}
	round.save.ECDSAPub = round.temp.ecdsaPubKey
	round.end <- *round.save

	return nil
}

func (round *roundout) CanAccept(msg tss.ParsedMessage) bool {
	// not expecting any incoming messages in this round
	return false
}

func (round *roundout) Update() (bool, *tss.Error) {
	// not expecting any incoming messages in this round
	return false, nil
}

func (round *roundout) NextRound() tss.Round {
	return nil // finished!
}
