package resharing

import (
	"encoding/json"
	"fmt"

	"github.com/binance-chain/tss-lib/common"
	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/vss"
	ecdsautils "github.com/binance-chain/tss-lib/ecdsa"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

var _ tss.StatefulParty = (*LocalStatefulParty)(nil)

type (
	LocalStatefulParty struct {
		*LocalParty
		preAdvanceFunc func(tss.StatefulParty, tss.ParsedMessage) (bool, *tss.Error)
	}

	BaseMarshalledMessages struct {
		Routing tss.MessageRouting
		Wire    *tss.MessageWrapper
	}

	MarshalledDgRound1Messages struct {
		BaseMarshalledMessages
		Content DGRound1Message
	}

	MarshalledDGRound2Message1 struct {
		BaseMarshalledMessages
		Content DGRound2Message1
	}

	MarshalledDGRound2Message2 struct {
		BaseMarshalledMessages
		Content DGRound2Message2
	}

	MarshalledDGRound3Message1 struct {
		BaseMarshalledMessages
		Content DGRound3Message1
	}

	MarshalledDGRound3Message2 struct {
		BaseMarshalledMessages
		Content DGRound3Message2
	}

	MarshalledDGRound4Message struct {
		BaseMarshalledMessages
		Content DGRound4Message
	}

	MarshalledLocalTempData struct {

		// localMessageStore
		DgRound1Messages  []MarshalledDgRound1Messages
		DgRound2Message1s []MarshalledDGRound2Message1
		DgRound2Message2s []MarshalledDGRound2Message2
		DgRound3Message1s []MarshalledDGRound3Message1
		DgRound3Message2s []MarshalledDGRound3Message2
		DgRound4Messages  []MarshalledDGRound4Message

		ECDSAPub *crypto.ECPoint

		// localTempData
		NewVs     vss.Vs
		NewShares vss.Shares
		VD        []*big.Int
		SessionId *big.Int

		NewXi         *big.Int
		NewKs         []*big.Int
		NewBigXjs     []*crypto.ECPoint // Xj to save in round 5
		AbortTriggers []ecdsautils.AbortTrigger
	}

	// MarshalledStatefulPartyData struct {
	//	TheMarshalledLocalTempData MarshalledLocalTempData
	// }
)

func NewLocalStatefulParty(
	params *tss.ReSharingParameters,
	key keygen.LocalPartySaveData,
	out chan<- tss.Message,
	end chan<- keygen.LocalPartySaveData,
	preAdvanceFunc func(tss.StatefulParty, tss.ParsedMessage) (bool, *tss.Error),
	sessionId *big.Int,
) (tss.StatefulParty, error) {
	var party tss.Party
	var err error
	if party, err = NewLocalParty(params, key, out, end, sessionId); err != nil {
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

func LocalTempDataToMarshalled(data *localTempData, ECDSAPub *crypto.ECPoint) MarshalledLocalTempData {
	marshalledLocalTempData := MarshalledLocalTempData{}

	marshalledLocalTempData.NewVs = data.NewVs
	marshalledLocalTempData.NewShares = data.NewShares
	if data.VD != nil {
		marshalledLocalTempData.VD = make([]*big.Int, len(data.VD))
		for i, v := range data.VD {
			marshalledLocalTempData.VD[i] = new(big.Int).SetBytes(v.Bytes())
		}
	}
	marshalledLocalTempData.SessionId = data.sessionId

	marshalledLocalTempData.NewXi = data.newXi
	marshalledLocalTempData.NewKs = data.newKs
	marshalledLocalTempData.NewBigXjs = data.newBigXjs
	marshalledLocalTempData.AbortTriggers = data.abortTriggers

	// localMessageStore
	marshalledLocalTempData.DgRound1Messages = ParsedMessageToDGRound1Message(data.dgRound1Messages)
	marshalledLocalTempData.DgRound2Message1s = ParsedMessageToDGRound2Message1(data.dgRound2Message1s)
	marshalledLocalTempData.DgRound2Message2s = ParsedMessageToDGRound2Message2(data.dgRound2Message2s)
	marshalledLocalTempData.DgRound3Message1s = ParsedMessageToDGRound3Message1(data.dgRound3Message1s)
	marshalledLocalTempData.DgRound3Message2s = ParsedMessageToDGRound3Message2(data.dgRound3Message2s)
	marshalledLocalTempData.DgRound4Messages = ParsedMessageToDGRound4Message(data.dgRound4Messages)
	if ECDSAPub != nil {
		marshalledLocalTempData.ECDSAPub = ECDSAPub
	}
	return marshalledLocalTempData
}

func ParsedMessageToDGRound1Message(pArray []tss.ParsedMessage) []MarshalledDgRound1Messages {
	result := make([]MarshalledDgRound1Messages, len(pArray))
	for i, v := range pArray {
		if v != nil {
			m := v.(*tss.MessageImpl)
			content := m.Content()
			result[i] = MarshalledDgRound1Messages{BaseMarshalledMessages{m.MessageRouting, m.WireMsg()},
				*(content.(*DGRound1Message))}
		}
	}
	return result
}

func ParsedMessageToDGRound2Message1(pArray []tss.ParsedMessage) []MarshalledDGRound2Message1 {
	result := make([]MarshalledDGRound2Message1, len(pArray))
	for i, v := range pArray {
		if v != nil {
			m := v.(*tss.MessageImpl)
			content := m.Content()
			result[i] = MarshalledDGRound2Message1{BaseMarshalledMessages{m.MessageRouting, m.WireMsg()},
				*(content.(*DGRound2Message1))}
		}
	}
	return result
}

func ParsedMessageToDGRound2Message2(pArray []tss.ParsedMessage) []MarshalledDGRound2Message2 {
	result := make([]MarshalledDGRound2Message2, len(pArray))
	for i, v := range pArray {
		if v != nil {
			m := v.(*tss.MessageImpl)
			content := m.Content()
			result[i] = MarshalledDGRound2Message2{BaseMarshalledMessages{m.MessageRouting, m.WireMsg()},
				*(content.(*DGRound2Message2))}
		}
	}
	return result
}

func ParsedMessageToDGRound3Message1(pArray []tss.ParsedMessage) []MarshalledDGRound3Message1 {
	result := make([]MarshalledDGRound3Message1, len(pArray))
	for i, v := range pArray {
		if v != nil {
			m := v.(*tss.MessageImpl)
			content := m.Content()
			result[i] = MarshalledDGRound3Message1{BaseMarshalledMessages{m.MessageRouting, m.WireMsg()},
				*(content.(*DGRound3Message1))}
		}
	}
	return result
}

func ParsedMessageToDGRound3Message2(pArray []tss.ParsedMessage) []MarshalledDGRound3Message2 {
	result := make([]MarshalledDGRound3Message2, len(pArray))
	for i, v := range pArray {
		if v != nil {
			m := v.(*tss.MessageImpl)
			content := m.Content()
			result[i] = MarshalledDGRound3Message2{BaseMarshalledMessages{m.MessageRouting, m.WireMsg()},
				*(content.(*DGRound3Message2))}

		}
	}
	return result
}

func ParsedMessageToDGRound4Message(pArray []tss.ParsedMessage) []MarshalledDGRound4Message {
	result := make([]MarshalledDGRound4Message, len(pArray))
	for i, v := range pArray {
		if v != nil {
			m := v.(*tss.MessageImpl)
			content := m.Content()
			result[i] = MarshalledDGRound4Message{BaseMarshalledMessages{m.MessageRouting, m.WireMsg()},
				*(content.(*DGRound4Message))}
		}
	}
	return result
}

func DGRound1MessageToParsedMessage(mArray []MarshalledDgRound1Messages, sessionId *big.Int, count int,
	params *tss.ReSharingParameters) []tss.ParsedMessage {
	if len(mArray) == 0 {
		return nil
	}
	if nilMarshalledMessages(mArray[0].BaseMarshalledMessages, mArray[len(mArray)-1].BaseMarshalledMessages) {
		return nil
	}
	result := make([]tss.ParsedMessage, count)
	for i, v := range mArray {
		ECDSAPub, err := v.Content.UnmarshalECDSAPub(params.EC())
		if err != nil {
			return nil
		}
		result[i] = NewDGRound1Message(sessionId, v.Routing.To, v.Routing.From, ECDSAPub, v.Content.UnmarshalVCommitment())
	}
	return result
}

func DGRound2Message1ToParsedMessage(mArray []MarshalledDGRound2Message1, sessionId *big.Int, count int) []tss.ParsedMessage {
	if len(mArray) == 0 {
		return nil
	}
	if nilMarshalledMessages(mArray[0].BaseMarshalledMessages, mArray[len(mArray)-1].BaseMarshalledMessages) {
		return nil
	}
	result := make([]tss.ParsedMessage, count)
	for i, v := range mArray {
		result[i] = NewDGRound2Message1(sessionId,
			v.Routing.To, v.Routing.From,
			v.Content.UnmarshalPaillierPK(), v.Content.UnmarshalPaillierProof(),
			new(big.Int).SetBytes(v.Content.NTilde),
			new(big.Int).SetBytes(v.Content.H1),
			new(big.Int).SetBytes(v.Content.H2))
	}
	return result
}

func nilMarshalledMessages(a1, an BaseMarshalledMessages) bool {
	if a1.Routing.From != nil || (a1.Wire != nil && a1.Wire.From != nil) {
		return false
	}
	if an.Routing.From != nil || (an.Wire != nil && an.Wire.From != nil) {
		return false
	}
	return true
}

func DGRound2Message2ToParsedMessage(mArray []MarshalledDGRound2Message2, count int) []tss.ParsedMessage {
	result := make([]tss.ParsedMessage, count)
	for i, v := range mArray {
		if v.Wire != nil {
			msg := tss.NewMessage(v.Routing, &v.Content, v.Wire)
			result[i] = msg
		}
	}
	return result
}

func DGRound3Message1ToParsedMessage(mArray []MarshalledDGRound3Message1, count int) []tss.ParsedMessage {
	result := make([]tss.ParsedMessage, count)
	for i, v := range mArray {
		if v.Wire != nil {
			msg := tss.NewMessage(v.Routing, &v.Content, v.Wire)
			result[i] = msg
		}
	}
	return result
}

func DGRound3Message2ToParsedMessage(mArray []MarshalledDGRound3Message2, count int) []tss.ParsedMessage {
	result := make([]tss.ParsedMessage, count)
	for i, v := range mArray {
		if v.Wire != nil {
			msg := tss.NewMessage(v.Routing, &v.Content, v.Wire)
			result[i] = msg
		}
	}
	return result
}

func DGRound4MessageToParsedMessage(mArray []MarshalledDGRound4Message, count int) []tss.ParsedMessage {
	result := make([]tss.ParsedMessage, count)
	for i, v := range mArray {
		if v.Wire != nil {
			msg := tss.NewMessage(v.Routing, &v.Content, v.Wire)
			result[i] = msg
		}
	}
	return result
}

func MarshalledToLocalTempData(marshalledLocalTempData *MarshalledLocalTempData, tempData *localTempData,
	saveData *keygen.LocalPartySaveData, partyID *tss.PartyID, params *tss.ReSharingParameters,
	sessionId *big.Int) {

	oldPartyCount := len(params.OldParties().IDs())
	newPartyCount := params.NewPartyCount()

	tempData.NewVs = marshalledLocalTempData.NewVs
	tempData.NewShares = marshalledLocalTempData.NewShares
	if len(marshalledLocalTempData.VD) > 0 {
		tempData.VD = make([]*big.Int, len(marshalledLocalTempData.VD))
		for i, v := range marshalledLocalTempData.VD {
			tempData.VD[i] = v
		}
	}
	tempData.sessionId = marshalledLocalTempData.SessionId

	tempData.newXi = marshalledLocalTempData.NewXi
	tempData.newKs = marshalledLocalTempData.NewKs
	tempData.newBigXjs = marshalledLocalTempData.NewBigXjs
	tempData.abortTriggers = marshalledLocalTempData.AbortTriggers

	somethingIsNotNil := func(array []tss.ParsedMessage) bool {
		for _, v := range array {
			if v != nil {
				return true
			}
		}
		return false
	}
	// localMessageStore
	var array []tss.ParsedMessage
	array = DGRound1MessageToParsedMessage(marshalledLocalTempData.DgRound1Messages, sessionId, oldPartyCount, params)
	if array != nil && len(array) > 0 && somethingIsNotNil(array) {
		copy(tempData.dgRound1Messages, array)
	}
	array = DGRound2Message1ToParsedMessage(marshalledLocalTempData.DgRound2Message1s, sessionId, newPartyCount)
	if array != nil && len(array) > 0 && somethingIsNotNil(array) {
		copy(tempData.dgRound2Message1s, array)
	}

	array = DGRound2Message2ToParsedMessage(marshalledLocalTempData.DgRound2Message2s, newPartyCount)
	if array != nil && len(array) > 0 && somethingIsNotNil(array) {
		copy(tempData.dgRound2Message2s, array)
	}
	array = DGRound3Message1ToParsedMessage(marshalledLocalTempData.DgRound3Message1s, oldPartyCount)
	if array != nil && len(array) > 0 && somethingIsNotNil(array) {
		copy(tempData.dgRound3Message1s, array)
	}
	array = DGRound3Message2ToParsedMessage(marshalledLocalTempData.DgRound3Message2s, oldPartyCount)
	if array != nil && len(array) > 0 && somethingIsNotNil(array) {
		copy(tempData.dgRound3Message2s, array)
	}
	array = DGRound4MessageToParsedMessage(marshalledLocalTempData.DgRound4Messages, newPartyCount)
	if array != nil && len(array) > 0 && somethingIsNotNil(array) {
		copy(tempData.dgRound4Messages, array)
	}
	saveData.ECDSAPub = marshalledLocalTempData.ECDSAPub
}

func StringToMarshalledLocalTempData(serializedPartyState string) (MarshalledLocalTempData, error) {
	var blob = []byte(serializedPartyState)
	var marshalledStatefulPartyData MarshalledLocalTempData
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
	MarshalledToLocalTempData(&marshalledStatefulPartyData, &p.temp, &p.save, p.PartyID(), p.params, p.temp.sessionId)
	return true, nil
}

func (p *LocalStatefulParty) Dehydrate() (string, *tss.Error) {
	tempData := LocalTempDataToMarshalled(&p.temp, p.save.ECDSAPub)
	bz, err := json.Marshal(tempData)
	if err != nil {
		return "", p.WrapError(err)
	}
	return string(bz[:]), nil
}

func (p *LocalStatefulParty) RoundByNumber(roundNumber int) tss.Round {
	newRound := []interface{}{newRound1, newRound2, newRound3, newRound4, newRound5}
	return newRound[roundNumber-1].(func(*tss.ReSharingParameters, *keygen.LocalPartySaveData, *keygen.LocalPartySaveData, *localTempData, chan<- tss.Message, chan<- keygen.LocalPartySaveData) tss.Round)(p.params, &p.input, &p.save, &p.temp, p.out, p.end)
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
