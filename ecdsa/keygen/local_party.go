// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package keygen

import (
	"errors"
	"fmt"

	"github.com/binance-chain/tss-lib/common"
	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto"
	zkpfac "github.com/binance-chain/tss-lib/crypto/zkp/fac"

	// cmt "github.com/binance-chain/tss-lib/crypto/commitments"
	"github.com/binance-chain/tss-lib/crypto/vss"
	zkpmod "github.com/binance-chain/tss-lib/crypto/zkp/mod"
	zkpprm "github.com/binance-chain/tss-lib/crypto/zkp/prm"
	zkpsch "github.com/binance-chain/tss-lib/crypto/zkp/sch"
	"github.com/binance-chain/tss-lib/tss"
)

// Implements Party
// Implements Stringer
var _ tss.Party = (*LocalParty)(nil)
var _ fmt.Stringer = (*LocalParty)(nil)

const MaxParties = 10000

type (
	LocalParty struct {
		*tss.BaseParty
		params *tss.Parameters

		temp localTempData
		data LocalPartySaveData

		// outbound messaging
		out chan<- tss.Message
		end chan<- LocalPartySaveData
	}

	localTempData struct {
		// temp data (thrown away after keygen)
		ui                   *big.Int // used for tests
		ridi                 *big.Int
		sid                  *big.Int
		rid                  *big.Int
		shares               vss.Shares
		vs                   vss.Vs
		AiKeygen             *crypto.ECPoint
		XiKeygen             *crypto.ECPoint
		τKeygen, 𝜏KeyRefresh *big.Int
		sessionId            *big.Int
		ecdsaPubKey          *crypto.ECPoint

		// key refresh:
		xⁿᵢ           []*big.Int
		𝜏js           []*big.Int
		AiRefreshList []*crypto.ECPoint
		XiRefreshList []*crypto.ECPoint
		Yᵢ            *crypto.ECPoint
		yᵢ            *big.Int
		𝜓ᵢ            *zkpprm.ProofPrm
		𝜌ᵢ            *big.Int
		Bᵢ            *crypto.ECPoint
		ssid          *big.Int
		𝜌             *big.Int

		r1msgVjKeygen []*big.Int

		// Refresh:
		rref1msgVjKeyRefresh []*big.Int
		rref1msgSid          []*big.Int
		rref1msgSsid         []*big.Int

		// Keygen:
		r2msgSid      []*big.Int
		r2msgRidj     []*big.Int
		r2msgUj       []*big.Int
		r2msgVss      [][]*crypto.ECPoint
		r2msgAKeygenj []*crypto.ECPoint
		r2msgXKeygenj []*crypto.ECPoint

		// Refresh:
		rref2msgSsid                       []*big.Int
		rref2msgXj                         [][]*crypto.ECPoint // first index: owner. Second index: recipient.
		rref2msgAj                         [][]*crypto.ECPoint
		rref2msgYj                         []*crypto.ECPoint
		rref2msgBj                         []*crypto.ECPoint
		rref2msgNj, rref2msgsj, rref2msgtj []*big.Int
		rref2msgpf𝜓j                       []*zkpprm.ProofPrm
		rref2msg𝜌j                         []*big.Int

		// Keygen:
		r3msgSid  []*big.Int
		r3msgpf𝜓j []*zkpsch.ProofSch
		r3msgxij  []*big.Int

		// Refresh:
		rref3msgSsid  []*big.Int
		rref3msgpf𝜓j  []*zkpmod.ProofMod
		rref3msgpf𝜙ji []*zkpfac.ProofFac
		rref3msgpfᴨᵢ  []*zkpsch.ProofSch

		rref3msgCzeroji, rref3msgRandomnessCzeroji []*big.Int
		rref3msgpf𝜓ⁱⱼ                              []*zkpsch.ProofSch

		r4msgSid       []*big.Int
		r4msg𝜇j        []*big.Int
		r4msgAbortingj []bool
		r4msgCulpritPj []int // alleged
		r4msgCji       []*big.Int
		r4msgxji       []*big.Int
	}
)

// Exported, used in `tss` client
func NewLocalParty(
	params *tss.Parameters,
	out chan<- tss.Message,
	end chan<- LocalPartySaveData,
	sessionId *big.Int,
	optionalPreParams ...LocalPreParams,
) (tss.Party, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	partyCount := params.PartyCount()
	data := NewLocalPartySaveData(partyCount)
	// when `optionalPreParams` is provided we'll use the pre-computed primes instead of generating them from scratch
	if partyCount > MaxParties {
		return nil, fmt.Errorf("keygen.NewLocalParty expected at most %d parties", MaxParties)
	}
	if 0 < len(optionalPreParams) {
		if 1 < len(optionalPreParams) {
			panic(errors.New("keygen.NewLocalParty expected 0 or 1 item in `optionalPreParams`"))
		}
		if !optionalPreParams[0].Validate() {
			panic(errors.New("keygen.NewLocalParty: `optionalPreParams` failed to validate"))
		}
		data.LocalPreParams = optionalPreParams[0]
	}
	p := &LocalParty{
		BaseParty: new(tss.BaseParty),
		params:    params,
		temp:      localTempData{},
		data:      data,
		out:       out,
		end:       end,
	}
	// msgs data init
	p.temp.rref1msgSid = make([]*big.Int, partyCount)
	p.temp.r1msgVjKeygen = make([]*big.Int, partyCount)
	// Refresh:
	p.temp.rref1msgVjKeyRefresh = make([]*big.Int, partyCount)
	p.temp.rref1msgSsid = make([]*big.Int, partyCount)

	p.temp.r2msgVss = make([][]*crypto.ECPoint, partyCount)
	p.temp.r2msgAKeygenj = make([]*crypto.ECPoint, partyCount)
	p.temp.r2msgXKeygenj = make([]*crypto.ECPoint, partyCount)
	p.temp.r2msgRidj = make([]*big.Int, partyCount)
	p.temp.r2msgSid = make([]*big.Int, partyCount)
	p.temp.r2msgUj = make([]*big.Int, partyCount)

	// Refresh:
	p.temp.rref2msgSsid = make([]*big.Int, partyCount)
	p.temp.rref2msgXj = make([][]*crypto.ECPoint, partyCount)
	p.temp.rref2msgAj = make([][]*crypto.ECPoint, partyCount)
	p.temp.rref2msgYj = make([]*crypto.ECPoint, partyCount)
	p.temp.rref2msgBj = make([]*crypto.ECPoint, partyCount)
	p.temp.rref2msgNj = make([]*big.Int, partyCount)
	p.temp.rref2msgsj = make([]*big.Int, partyCount)
	p.temp.rref2msgtj = make([]*big.Int, partyCount)
	p.temp.rref2msgpf𝜓j = make([]*zkpprm.ProofPrm, partyCount)
	p.temp.rref2msg𝜌j = make([]*big.Int, partyCount)

	// Keygen:
	p.temp.r3msgSid = make([]*big.Int, partyCount)
	p.temp.r3msgpf𝜓j = make([]*zkpsch.ProofSch, partyCount)
	p.temp.r3msgxij = make([]*big.Int, partyCount)

	// Refresh:
	p.temp.rref3msgSsid = make([]*big.Int, partyCount)
	p.temp.rref3msgpf𝜓j = make([]*zkpmod.ProofMod, partyCount)
	p.temp.rref3msgpf𝜙ji = make([]*zkpfac.ProofFac, partyCount)
	p.temp.rref3msgpfᴨᵢ = make([]*zkpsch.ProofSch, partyCount)
	p.temp.rref3msgCzeroji = make([]*big.Int, partyCount)

	p.temp.rref3msgRandomnessCzeroji = make([]*big.Int, partyCount)
	p.temp.rref3msgpf𝜓ⁱⱼ = make([]*zkpsch.ProofSch, partyCount)

	p.temp.r4msgSid = make([]*big.Int, partyCount)
	p.temp.r4msg𝜇j = make([]*big.Int, partyCount)
	p.temp.r4msgAbortingj = make([]bool, partyCount)
	for j := 0; j < partyCount; j++ {
		p.temp.r4msgAbortingj[j] = false
	}
	p.temp.r4msgCulpritPj = make([]int, partyCount)
	p.temp.r4msgCji = make([]*big.Int, partyCount)
	p.temp.r4msgxji = make([]*big.Int, partyCount)

	// hash the sessionID to make sure it's of the expected length when used as a nonce
	p.temp.sessionId = tss.ExpandSessionID(sessionId, len(p.params.EC().Params().N.Bytes()))
	return p, nil
}

func (p *LocalParty) FirstRound() tss.Round {
	return newRound1(p.params, &p.data, &p.temp, p.out, p.end)
}

func (p *LocalParty) Start() *tss.Error {
	return tss.BaseStart(p, TaskName)
}

func (p *LocalParty) Update(msg tss.ParsedMessage) (ok bool, err *tss.Error) {
	return tss.BaseUpdate(p, msg, TaskName)
}

func (p *LocalParty) UpdateFromBytes(wireBytes []byte, from *tss.PartyID, isBroadcast bool, sessionId *big.Int) (bool, *tss.Error) {
	sessionId = tss.ExpandSessionID(sessionId, len(p.params.EC().Params().N.Bytes()))
	msg, err := tss.ParseWireMessage(wireBytes, from, isBroadcast, sessionId)
	if err != nil {
		return false, p.WrapError(err)
	}
	return p.Update(msg)
}

func (p *LocalParty) ValidateMessage(msg tss.ParsedMessage) (bool, *tss.Error) {
	if ok, err := p.BaseParty.ValidateMessage(msg); !ok || err != nil {
		return ok, err
	}
	// check that the message's "from index" will fit into the array
	if maxFromIdx := p.params.PartyCount() - 1; maxFromIdx < msg.GetFrom().Index {
		return false, p.WrapError(fmt.Errorf("received msg with a sender index too great (%d <= %d)",
			p.params.PartyCount(), msg.GetFrom().Index), msg.GetFrom())
	}
	if p.temp.sessionId == nil || msg.GetSessionId() == nil || p.temp.sessionId.Cmp(msg.GetSessionId()) != 0 {
		return false, p.WrapError(fmt.Errorf("wrong session in message (%s)",
			msg.GetSessionId().String()), msg.GetFrom())
	}
	return true, nil
}

func (p *LocalParty) StoreMessage(msg tss.ParsedMessage) (bool, *tss.Error) {
	// ValidateBasic is cheap; double-check the message here in case the public StoreMessage was called externally
	if ok, err := p.ValidateMessage(msg); !ok || err != nil {
		return ok, err
	}
	fromPIdx := msg.GetFrom().Index

	// switch/case is necessary to store any messages beyond current round
	// this does not handle message replays. we expect the caller to apply replay and spoofing protection.
	switch msg.Content().(type) {
	case *KGRound1Message:
		// p.temp.kgRound1Messages[fromPIdx] = msg // TODO remove
		r1msg := msg.Content().(*KGRound1Message)
		p.temp.r1msgVjKeygen[fromPIdx] = r1msg.UnmarshalViKeygen()
		p.temp.rref1msgVjKeyRefresh[fromPIdx] = r1msg.UnmarshalViKeyRefresh()
		p.temp.rref1msgSid[fromPIdx] = r1msg.UnmarshalSid()
		p.temp.rref1msgSsid[fromPIdx] = r1msg.UnmarshalSsid()
	case *KGRound2Message:
		// p.temp.kgRound2Messages[fromPIdx] = msg
		var err error
		r2msg, ok := msg.Content().(*KGRound2Message)
		p.temp.r2msgSid[fromPIdx] = r2msg.UnmarshalSid()
		p.temp.r2msgRidj[fromPIdx] = r2msg.UnmarshalRidi()
		p.temp.r2msgUj[fromPIdx] = r2msg.UnmarshalUi()
		if !ok {
			return false, p.WrapError(fmt.Errorf("error with KGRound2Message (%d)", fromPIdx))
		}
		p.data.PaillierPKs[fromPIdx] = r2msg.UnmarshalPaillierPK() // used in round 4
		p.data.NTildej[fromPIdx] = r2msg.UnmarshalNi()
		p.data.H1j[fromPIdx], p.data.H2j[fromPIdx] = r2msg.UnmarshalSi(), r2msg.UnmarshalTi()
		p.temp.r2msgVss[fromPIdx], err = r2msg.UnmarshalVs(p.params.EC())
		if err != nil {
			return false, p.WrapError(err)
		}
		p.temp.r2msgAKeygenj[fromPIdx], err = r2msg.UnmarshalAiKeygen(p.params.EC())
		if err != nil {
			return false, p.WrapError(err)
		}
		p.temp.r2msgXKeygenj[fromPIdx], err = r2msg.UnmarshalXiKeygen(p.params.EC())
		if err != nil {
			return false, p.WrapError(err)
		}
		p.data.PaillierPKs[fromPIdx] = r2msg.UnmarshalPaillierPK() // used in round 4

		// Refresh:
		p.temp.rref2msgSsid[fromPIdx] = r2msg.UnmarshalSsid()
		p.temp.rref2msgXj[fromPIdx], err = r2msg.UnmarshalXiRefresh(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}
		p.temp.rref2msgAj[fromPIdx], err = r2msg.UnmarshalAiRefresh(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}
		p.temp.rref2msgYj[fromPIdx], err = r2msg.UnmarshalYi(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}
		p.temp.rref2msgBj[fromPIdx], err = r2msg.UnmarshalBi(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}
		p.temp.rref2msgNj[fromPIdx] = r2msg.UnmarshalNi()
		p.temp.rref2msgsj[fromPIdx] = r2msg.UnmarshalSi()
		p.temp.rref2msgtj[fromPIdx] = r2msg.UnmarshalTi()
		p.temp.rref2msgpf𝜓j[fromPIdx], err = r2msg.Unmarshal𝜓i()
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}
		p.temp.rref2msg𝜌j[fromPIdx] = r2msg.Unmarshal𝜌i()
	case *KGRound3Message:
		// p.temp.kgRound3Messages[fromPIdx] = msg
		r3msg := msg.Content().(*KGRound3Message)
		// Keygen:
		p.temp.r3msgSid[fromPIdx] = r3msg.UnmarshalSid()
		𝜓Schj, err := r3msg.Unmarshal𝜓SchProof(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}
		p.temp.r3msgpf𝜓j[fromPIdx] = 𝜓Schj
		xij, err := p.data.PaillierSK.Decrypt(r3msg.UnmarshalCvssji())
		p.temp.r3msgxij[fromPIdx] = xij

		// Refresh:
		p.temp.rref3msgSsid[fromPIdx] = r3msg.UnmarshalSsid()
		p.temp.rref3msgpf𝜓j[fromPIdx], err = r3msg.Unmarshal𝜓ModProof()
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}
		p.temp.rref3msgpf𝜙ji[fromPIdx], err = r3msg.Unmarshal𝜙ji()
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}
		p.temp.rref3msgpfᴨᵢ[fromPIdx], err = r3msg.Unmarshalᴨi(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}

		p.temp.rref3msgCzeroji[fromPIdx] = r3msg.UnmarshalCzeroji()
		p.temp.rref3msgRandomnessCzeroji[fromPIdx] = r3msg.UnmarshalRandomnessCzeroji()
		p.temp.rref3msgpf𝜓ⁱⱼ[fromPIdx], err = r3msg.Unmarshal𝜓ji(p.params.EC())
		if err != nil {
			return false, p.WrapError(err, p.params.Parties().IDs()[fromPIdx])
		}
	case *KGRound4Message:
		// p.temp.kgRound4Messages[fromPIdx] = msg
		r4msg := msg.Content().(*KGRound4Message)
		p.temp.r4msgSid[fromPIdx] = r4msg.UnmarshalSid()
		p.temp.r4msg𝜇j[fromPIdx] = r4msg.UnmarshalMu()
		p.temp.r4msgAbortingj[fromPIdx] = r4msg.GetAbort()
		p.temp.r4msgCulpritPj[fromPIdx] = int(r4msg.GetCulpritPj())
		p.temp.r4msgCji[fromPIdx] = r4msg.UnmarshalCji()
		p.temp.r4msgxji[fromPIdx] = r4msg.UnmarshalXji()

	default: // unrecognised message, just ignore!
		common.Logger.Warnf("unrecognised message ignored: %v", msg)
		return false, nil
	}
	return true, nil
}

// recovers a party's original index in the set of parties during keygen
func (save LocalPartySaveData) OriginalIndex() (int, error) {
	index := -1
	ki := save.ShareID
	for j, kj := range save.Ks {
		if kj.Cmp(ki) != 0 {
			continue
		}
		index = j
		break
	}
	if index < 0 {
		return -1, errors.New("a party index could not be recovered from Ks")
	}
	return index, nil
}

func (p *LocalParty) Params() *tss.Parameters {
	return p.params
}

func (p *LocalParty) PartyID() *tss.PartyID {
	return p.params.PartyID()
}

func (p *LocalParty) String() string {
	return fmt.Sprintf("id: %s, %s", p.PartyID(), p.BaseParty.String())
}
