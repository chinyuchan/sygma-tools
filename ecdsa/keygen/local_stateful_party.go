package keygen

import (
	"encoding/json"
	"fmt"

	"github.com/binance-chain/tss-lib/common"
	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/vss"
	zkpfac "github.com/binance-chain/tss-lib/crypto/zkp/fac"
	zkpmod "github.com/binance-chain/tss-lib/crypto/zkp/mod"
	zkpprm "github.com/binance-chain/tss-lib/crypto/zkp/prm"
	zkpsch "github.com/binance-chain/tss-lib/crypto/zkp/sch"
	"github.com/binance-chain/tss-lib/tss"
)

var _ tss.StatefulParty = (*LocalStatefulParty)(nil)

type (
	LocalStatefulParty struct {
		*LocalParty
		preAdvanceFunc func(tss.StatefulParty, tss.ParsedMessage) (bool, *tss.Error)
	}

	MarshalledLocalTempData struct {
		TheLocalPartySaveData    LocalPartySaveData
		Ui                       *big.Int
		Ridi                     *big.Int
		Sid                      *big.Int
		Rid                      *big.Int
		Shares                   vss.Shares
		Vs                       vss.Vs
		AiKeygen                 *crypto.ECPoint
		XiKeygen                 *crypto.ECPoint
		TauKeygen, TauKeyRefresh *big.Int
		SessionId                *big.Int
		EcdsaPubKey              *crypto.ECPoint

		// key refresh:
		Xni           []*big.Int
		Taujs         []*big.Int
		AiRefreshList []*crypto.ECPoint
		XiRefreshList []*crypto.ECPoint
		BigYi         *crypto.ECPoint
		Smallyi       *big.Int
		Psii          *zkpprm.ProofPrm
		Rhoi          *big.Int
		Bi            *crypto.ECPoint
		Ssid          *big.Int
		Rho           *big.Int

		// msg store
		R1msgVjKeygen []*big.Int

		// Refresh:
		Rref1msgVjKeyRefresh []*big.Int
		Rref1msgSid          []*big.Int
		Rref1msgSsid         []*big.Int

		// Keygen:
		R2msgSid      []*big.Int
		R2msgRidj     []*big.Int
		R2msgUj       []*big.Int
		R2msgVss      [][]*crypto.ECPoint
		R2msgAKeygenj []*crypto.ECPoint
		R2msgXKeygenj []*crypto.ECPoint

		// Refresh:
		Rref2msgSsid                       []*big.Int
		Rref2msgXj                         [][]*crypto.ECPoint // first index: owner. Second index: recipient.
		Rref2msgAj                         [][]*crypto.ECPoint
		Rref2msgYj                         []*crypto.ECPoint
		Rref2msgBj                         []*crypto.ECPoint
		Rref2msgNj, Rref2msgsj, Rref2msgtj []*big.Int
		Rref2msgpfPsij                     []*zkpprm.ProofPrm
		Rref2msgRhoj                       []*big.Int

		// Keygen:
		R3msgSid    []*big.Int
		R3msgpfPsij []*zkpsch.ProofSch
		R3msgxij    []*big.Int

		// Refresh:
		Rref3msgSsid    []*big.Int
		Rref3msgpfPsij  []*zkpmod.ProofMod
		Rref3msgpfPhiji []*zkpfac.ProofFac
		Rref3msgpfPii   []*zkpsch.ProofSch

		Rref3msgCzeroji, Rref3msgRandomnessCzeroji []*big.Int
		Rref3msgpfPsiij                            []*zkpsch.ProofSch

		R4msgSid       []*big.Int
		R4msgMuj       []*big.Int
		R4msgAbortingj []bool
		R4msgCulpritPj []int // alleged
		R4msgCji       []*big.Int
		R4msgxji       []*big.Int
	}

	MarshalledStatefulPartyData struct {
		TheMarshalledLocalTempData MarshalledLocalTempData
	}
)

func NewLocalStatefulParty(
	params *tss.Parameters,
	out chan<- tss.Message,
	end chan<- LocalPartySaveData,
	preAdvanceFunc func(tss.StatefulParty, tss.ParsedMessage) (bool, *tss.Error),
	sessionId *big.Int,
	optionalPreParams ...LocalPreParams,
) (tss.StatefulParty, error) {
	var party tss.Party
	var err error
	if party, err = NewLocalParty(params, out, end, sessionId, optionalPreParams...); err != nil {
		return nil, err
	}
	return &LocalStatefulParty{party.(*LocalParty), preAdvanceFunc}, nil
}

func (p *LocalStatefulParty) Update(msg tss.ParsedMessage) (ok bool, err *tss.Error) {
	f := func(_p tss.Party) (bool, *tss.Error) {
		p2 := _p.(*LocalStatefulParty)
		return p2.preAdvanceFunc(p2, msg)
	}
	ok, err = tss.BaseUpdate(p, msg, TaskName, f)
	if err != nil {
		return false, err
	}
	return
}

func LocalTempDataToMarshalled(data *localTempData, saveData *LocalPartySaveData) MarshalledLocalTempData {
	marshalledLocalTempData := MarshalledLocalTempData{}

	marshalledLocalTempData.TheLocalPartySaveData = *saveData

	marshalledLocalTempData.Ui = data.ui
	marshalledLocalTempData.Ridi = data.ridi
	marshalledLocalTempData.Sid = data.sid
	marshalledLocalTempData.Rid = data.rid
	marshalledLocalTempData.Shares = data.shares
	marshalledLocalTempData.Vs = data.vs
	marshalledLocalTempData.AiKeygen = data.AiKeygen
	marshalledLocalTempData.XiKeygen = data.XiKeygen
	marshalledLocalTempData.TauKeygen = data.τKeygen
	marshalledLocalTempData.TauKeyRefresh = data.𝜏KeyRefresh
	marshalledLocalTempData.SessionId = data.sessionId
	marshalledLocalTempData.EcdsaPubKey = data.ecdsaPubKey

	// key refresh:
	marshalledLocalTempData.Xni = data.xⁿᵢ
	marshalledLocalTempData.Taujs = data.𝜏js
	marshalledLocalTempData.AiRefreshList = data.AiRefreshList
	marshalledLocalTempData.XiRefreshList = data.XiRefreshList
	marshalledLocalTempData.BigYi = data.Yᵢ
	marshalledLocalTempData.Smallyi = data.yᵢ
	marshalledLocalTempData.Psii = data.𝜓ᵢ
	marshalledLocalTempData.Rhoi = data.𝜌ᵢ
	marshalledLocalTempData.Bi = data.Bᵢ
	marshalledLocalTempData.Ssid = data.ssid
	marshalledLocalTempData.Rho = data.𝜌

	// msg store
	marshalledLocalTempData.R1msgVjKeygen = data.r1msgVjKeygen

	// Refresh:
	marshalledLocalTempData.Rref1msgVjKeyRefresh = data.rref1msgVjKeyRefresh
	marshalledLocalTempData.Rref1msgSid = data.rref1msgSid
	marshalledLocalTempData.Rref1msgSsid = data.rref1msgSsid

	// Keygen:
	marshalledLocalTempData.R2msgSid = data.r2msgSid
	marshalledLocalTempData.R2msgRidj = data.r2msgRidj
	marshalledLocalTempData.R2msgUj = data.r2msgUj
	marshalledLocalTempData.R2msgVss = data.r2msgVss
	marshalledLocalTempData.R2msgAKeygenj = data.r2msgAKeygenj
	marshalledLocalTempData.R2msgXKeygenj = data.r2msgXKeygenj

	// Refresh:
	marshalledLocalTempData.Rref2msgSsid = data.rref2msgSsid
	marshalledLocalTempData.Rref2msgXj = data.rref2msgXj
	marshalledLocalTempData.Rref2msgAj = data.rref2msgAj
	marshalledLocalTempData.Rref2msgYj = data.rref2msgYj
	marshalledLocalTempData.Rref2msgBj = data.rref2msgBj
	marshalledLocalTempData.Rref2msgNj = data.rref2msgNj
	marshalledLocalTempData.Rref2msgsj = data.rref2msgsj
	marshalledLocalTempData.Rref2msgtj = data.rref2msgtj
	marshalledLocalTempData.Rref2msgpfPsij = data.rref2msgpf𝜓j
	marshalledLocalTempData.Rref2msgRhoj = data.rref2msg𝜌j

	// Keygen:
	marshalledLocalTempData.R3msgSid = data.r3msgSid
	marshalledLocalTempData.R3msgpfPsij = data.r3msgpf𝜓j
	marshalledLocalTempData.R3msgxij = data.r3msgxij

	// Refresh:
	marshalledLocalTempData.Rref3msgSsid = data.rref3msgSsid
	marshalledLocalTempData.Rref3msgpfPsij = data.rref3msgpf𝜓j
	marshalledLocalTempData.Rref3msgpfPhiji = data.rref3msgpf𝜙ji
	marshalledLocalTempData.Rref3msgpfPii = data.rref3msgpfᴨᵢ

	marshalledLocalTempData.Rref3msgCzeroji = data.rref3msgCzeroji
	marshalledLocalTempData.Rref3msgRandomnessCzeroji = data.rref3msgRandomnessCzeroji
	marshalledLocalTempData.Rref3msgpfPsiij = data.rref3msgpf𝜓ⁱⱼ

	marshalledLocalTempData.R4msgSid = data.r4msgSid
	marshalledLocalTempData.R4msgMuj = data.r4msg𝜇j
	marshalledLocalTempData.R4msgAbortingj = data.r4msgAbortingj
	marshalledLocalTempData.R4msgCulpritPj = data.r4msgCulpritPj
	marshalledLocalTempData.R4msgCji = data.r4msgCji
	marshalledLocalTempData.R4msgxji = data.r4msgxji
	return marshalledLocalTempData
}

func MarshalledToLocalTempData(marshalledLocalTempData *MarshalledLocalTempData) (*localTempData, *LocalPartySaveData) {
	data := &localTempData{}
	saveData := &LocalPartySaveData{}

	data.ui = marshalledLocalTempData.Ui
	data.ridi = marshalledLocalTempData.Ridi
	data.sid = marshalledLocalTempData.Sid
	data.rid = marshalledLocalTempData.Rid
	data.shares = marshalledLocalTempData.Shares
	data.vs = marshalledLocalTempData.Vs
	data.AiKeygen = marshalledLocalTempData.AiKeygen
	data.XiKeygen = marshalledLocalTempData.XiKeygen
	data.τKeygen = marshalledLocalTempData.TauKeygen
	data.𝜏KeyRefresh = marshalledLocalTempData.TauKeyRefresh
	data.sessionId = marshalledLocalTempData.SessionId
	data.ecdsaPubKey = marshalledLocalTempData.EcdsaPubKey

	// key refresh:
	data.xⁿᵢ = marshalledLocalTempData.Xni
	data.𝜏js = marshalledLocalTempData.Taujs
	data.AiRefreshList = marshalledLocalTempData.AiRefreshList
	data.XiRefreshList = marshalledLocalTempData.XiRefreshList
	data.Yᵢ = marshalledLocalTempData.BigYi
	data.yᵢ = marshalledLocalTempData.Smallyi
	data.𝜓ᵢ = marshalledLocalTempData.Psii
	data.𝜌ᵢ = marshalledLocalTempData.Rhoi
	data.Bᵢ = marshalledLocalTempData.Bi
	data.ssid = marshalledLocalTempData.Ssid
	data.𝜌 = marshalledLocalTempData.Rho

	// msg store
	data.r1msgVjKeygen = marshalledLocalTempData.R1msgVjKeygen

	// Refresh:
	data.rref1msgVjKeyRefresh = marshalledLocalTempData.Rref1msgVjKeyRefresh
	data.rref1msgSid = marshalledLocalTempData.Rref1msgSid
	data.rref1msgSsid = marshalledLocalTempData.Rref1msgSsid

	// Keygen:
	data.r2msgSid = marshalledLocalTempData.R2msgSid
	data.r2msgRidj = marshalledLocalTempData.R2msgRidj
	data.r2msgUj = marshalledLocalTempData.R2msgUj
	data.r2msgVss = marshalledLocalTempData.R2msgVss
	data.r2msgAKeygenj = marshalledLocalTempData.R2msgAKeygenj
	data.r2msgXKeygenj = marshalledLocalTempData.R2msgXKeygenj

	// Refresh:
	data.rref2msgSsid = marshalledLocalTempData.Rref2msgSsid
	data.rref2msgXj = marshalledLocalTempData.Rref2msgXj
	data.rref2msgAj = marshalledLocalTempData.Rref2msgAj
	data.rref2msgYj = marshalledLocalTempData.Rref2msgYj
	data.rref2msgBj = marshalledLocalTempData.Rref2msgBj
	data.rref2msgNj = marshalledLocalTempData.Rref2msgNj
	data.rref2msgsj = marshalledLocalTempData.Rref2msgsj
	data.rref2msgtj = marshalledLocalTempData.Rref2msgtj
	data.rref2msgpf𝜓j = marshalledLocalTempData.Rref2msgpfPsij
	data.rref2msg𝜌j = marshalledLocalTempData.Rref2msgRhoj

	// Keygen:
	data.r3msgSid = marshalledLocalTempData.R3msgSid
	data.r3msgpf𝜓j = marshalledLocalTempData.R3msgpfPsij
	data.r3msgxij = marshalledLocalTempData.R3msgxij

	// Refresh:
	data.rref3msgSsid = marshalledLocalTempData.Rref3msgSsid
	data.rref3msgpf𝜓j = marshalledLocalTempData.Rref3msgpfPsij
	data.rref3msgpf𝜙ji = marshalledLocalTempData.Rref3msgpfPhiji
	data.rref3msgpfᴨᵢ = marshalledLocalTempData.Rref3msgpfPii

	data.rref3msgCzeroji = marshalledLocalTempData.Rref3msgCzeroji
	data.rref3msgRandomnessCzeroji = marshalledLocalTempData.Rref3msgRandomnessCzeroji
	data.rref3msgpf𝜓ⁱⱼ = marshalledLocalTempData.Rref3msgpfPsiij

	data.r4msgSid = marshalledLocalTempData.R4msgSid
	data.r4msg𝜇j = marshalledLocalTempData.R4msgMuj
	data.r4msgAbortingj = marshalledLocalTempData.R4msgAbortingj
	data.r4msgCulpritPj = marshalledLocalTempData.R4msgCulpritPj
	data.r4msgCji = marshalledLocalTempData.R4msgCji
	data.r4msgxji = marshalledLocalTempData.R4msgxji
	saveData = &marshalledLocalTempData.TheLocalPartySaveData
	return data, saveData
}

func StringToMarshalledLocalTempData(serializedPartyState string) (MarshalledStatefulPartyData, error) {
	var blob = []byte(serializedPartyState)
	var marshalledStatefulPartyData MarshalledStatefulPartyData
	if err := json.Unmarshal(blob, &marshalledStatefulPartyData); err != nil {
		return marshalledStatefulPartyData, err
	}
	return marshalledStatefulPartyData, nil
}

func (p *LocalStatefulParty) Hydrate(marshalledPartyState string) (bool, *tss.Error) {
	marshalledStatefulPartyData, err := StringToMarshalledLocalTempData(marshalledPartyState)
	if err != nil {
		return false, p.WrapError(err)
	}
	tempData, saveData := MarshalledToLocalTempData(&marshalledStatefulPartyData.TheMarshalledLocalTempData)
	p.temp = *tempData
	p.data = *saveData
	return true, nil
}

func (p *LocalStatefulParty) Dehydrate() (string, *tss.Error) {
	tempData := LocalTempDataToMarshalled(&p.temp, &p.data)
	marshalledStatefulPartyData := &MarshalledStatefulPartyData{
		TheMarshalledLocalTempData: tempData,
	}
	// common.Logger.Debugf("party:%v, Dehydrate, StartRndNum: %v (real is -1)", p.PartyID(), marshalledStatefulPartyData.StartRndNum)
	bz, err := json.Marshal(marshalledStatefulPartyData)
	if err != nil {
		return "", p.WrapError(err)
	}
	return string(bz[:]), nil
}

func (p *LocalStatefulParty) RoundByNumber(roundNumber int) tss.Round {
	newRound := []interface{}{newRound1, newRound2, newRound3, newRound4, newRoundout}
	return newRound[roundNumber-1].(func(*tss.Parameters, *LocalPartySaveData, *localTempData, chan<- tss.Message, chan<- LocalPartySaveData) tss.Round)(p.params, &p.data, &p.temp, p.out, p.end)
}

func (p *LocalStatefulParty) Restart(roundNumber int, marshalledPartyState string) *tss.Error {
	p.Lock()
	defer p.Unlock()
	if p.PartyID() == nil || !p.PartyID().ValidateBasic() {
		return p.WrapError(fmt.Errorf("could not start. this party has an invalid PartyID: %+v", p.PartyID()))
	}
	var round tss.Round
	if marshalledPartyState != "" {
		_, errH := p.Hydrate(marshalledPartyState)
		if errH != nil {
			return errH
		}
	}
	if p.Round() == nil {
		round = p.RoundByNumber(roundNumber)
		if err := p.SetRound(round); err != nil {
			return err
		}
	} else {
		round = p.Round()
	}
	common.Logger.Infof("party %s (%p): %s round %d restarting", round.Params().PartyID(), p, TaskName, roundNumber)
	defer func() {
		common.Logger.Debugf("party %s (%p): %s round %d finished", round.Params().PartyID(), p, TaskName, roundNumber)
	}()
	return round.Start()
}
