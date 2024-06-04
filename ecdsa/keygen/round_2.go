// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package keygen

import (
	"errors"

	"github.com/binance-chain/tss-lib/common/hash"
	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/tss"
)

func newRound2(params *tss.Parameters, save *LocalPartySaveData, temp *localTempData, out chan<- tss.Message, end chan<- LocalPartySaveData) tss.Round {
	return &round2{&round1{
		&base{params, save, temp, out, end, make([]bool, len(params.Parties().IDs())), false, 2}}}
}

func (round *round2) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 2
	round.started = true
	round.resetOK()

	Pi := round.PartyID()
	i := Pi.Index
	round.ok[i] = true

	{
		xi := new(big.Int).Set(round.temp.shares[i].Share)
		XiKeygen := crypto.ScalarBaseMult(round.EC(), xi)
		sid := hash.SHA256i(append(round.Parties().IDs().Keys(), big.Wrap(tss.EC().Params().N),
			big.Wrap(tss.EC().Params().P),
			big.Wrap(tss.EC().Params().B), big.Wrap(tss.EC().Params().Gx), big.Wrap(tss.EC().Params().Gy))...)
		msg, err := NewKGRound2Message(round.temp.sessionId, round.PartyID(), round.temp.vs, &round.save.PaillierSK.PublicKey,
			sid, round.temp.ridi, XiKeygen, round.temp.AiKeygen, round.temp.ui,
			// key refresh:
			round.temp.ssid,
			round.temp.XiRefreshList,
			round.temp.AiRefreshList, round.temp.Yᵢ,
			round.temp.Bᵢ,
			round.save.LocalPreParams.NTildei, round.save.LocalPreParams.H1i, round.save.LocalPreParams.H2i,
			round.temp.𝜓ᵢ,
			round.temp.𝜌ᵢ)
		if err != nil {
			return round.WrapError(errors.New("msg error"))
		}
		round.out <- msg
	}

	return nil
}

func (round *round2) CanAccept(msg tss.ParsedMessage) bool {
	if _, ok := msg.Content().(*KGRound2Message); ok {
		return msg.IsBroadcast()
	}
	return false
}

func (round *round2) Update() (bool, *tss.Error) {
	for j, msg := range round.temp.rref2msg𝜌j {
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

func (round *round2) NextRound() tss.Round {
	round.started = false
	return &round3{round}
}
