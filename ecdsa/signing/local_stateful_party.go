package signing

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/binance-chain/tss-lib/common"
	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto"
	zkpaffg "github.com/binance-chain/tss-lib/crypto/zkp/affg"
	zkpdec "github.com/binance-chain/tss-lib/crypto/zkp/dec"
	zkpenc "github.com/binance-chain/tss-lib/crypto/zkp/enc"
	zkplogstar "github.com/binance-chain/tss-lib/crypto/zkp/logstar"
	zkpmul "github.com/binance-chain/tss-lib/crypto/zkp/mul"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

var _ tss.StatefulParty = (*LocalStatefulParty)(nil)

type (
	LocalStatefulParty struct {
		*LocalParty
		preAdvanceFunc func(tss.StatefulParty, tss.ParsedMessage) (bool, *tss.Error)
	}

	MarshalledLocalTempData struct {
		W     *big.Int
		BigWs []*crypto.ECPoint
		Ki    *big.Int

		BigGammai          *crypto.ECPoint
		K                  *big.Int
		G                  *big.Int
		Rhoi               *big.Int
		Nui                *big.Int
		KeyDerivationDelta *big.Int

		// round 2
		Gammai             *big.Int
		DeltaShareBetas    []*big.Int
		DeltaShareBetaNegs []*big.Int
		DeltaMtASij        []*big.Int
		DeltaMtARij        []*big.Int
		Dji                []*big.Int
		ChiShareBetas      []*big.Int
		DeltaMtAFji        []*big.Int
		ChiMtAF            *big.Int

		// round 3
		BigGamma         *crypto.ECPoint
		DeltaShareAlphas []*big.Int
		ChiShareAlphas   []*big.Int
		Deltai           *big.Int
		Chii             *big.Int
		BigDeltai        *crypto.ECPoint

		// round 4
		M          *big.Int
		BigR       *crypto.ECPoint
		Rx         *big.Int
		SigmaShare *big.Int

		// msg store
		SessionId          *big.Int
		R1msgG             []*big.Int
		R1msgK             []*big.Int
		R1msgPsiij         []*zkpenc.ProofEnc
		R2msgBigGammaShare []*crypto.ECPoint
		R2msgDeltaD        []*big.Int
		R2msgDeltaF        []*big.Int
		R2msgDeltaFjiPki   []*big.Int
		R2msgDeltaProof    []*zkpaffg.ProofAffg
		R2msgChiD          []*big.Int
		R2msgChiF          []*big.Int
		R2msgChiProof      []*zkpaffg.ProofAffg
		R2msgProofLogstar  []*zkplogstar.ProofLogstar
		R3msgSigmaj        []*big.Int
		R3msgDeltaj        []*crypto.ECPoint
		R3msgProofLogstar  []*zkplogstar.ProofLogstar
		R4msgSigmaj        []*big.Int
		R4msgAborting      []bool
		// for identification
		R5msgGammaj      []*big.Int
		R5msgsji         []*big.Int
		R5msgBetaPrimeji []*big.Int

		R6msgH                 []*big.Int
		R6msgProofMul          []*zkpmul.ProofMul
		R6msgProofDec          []*zkpdec.ProofDec
		R6msgDeltaShareEnc     []*big.Int
		R6msgEncryptedValueSum []*big.Int
	}

	MarshalledStatefulPartyData struct {
		TheMarshalledLocalTempData MarshalledLocalTempData
		StartRndNum                int
		Version                    int
	}
)

func NewLocalStatefulParty(
	msg *big.Int,
	params *tss.Parameters,
	key keygen.LocalPartySaveData,
	keyDerivationDelta *big.Int,
	out chan<- tss.Message,
	end chan<- common.SignatureData,
	preAdvanceFunc func(tss.StatefulParty, tss.ParsedMessage) (bool, *tss.Error),
	sessionId *big.Int,
	startRndNums ...int,
) (tss.StatefulParty, error) {
	var party tss.Party
	var err error
	if party, err = NewLocalParty(msg, params, key, keyDerivationDelta, out, end, sessionId, startRndNums...); err != nil {
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

func LocalTempDataToMarshalled(data *localTempData) MarshalledLocalTempData {
	marshalledLocalTempData := MarshalledLocalTempData{}
	marshalledLocalTempData.W = data.w
	marshalledLocalTempData.BigWs = data.BigWs
	marshalledLocalTempData.Ki = data.ki

	marshalledLocalTempData.BigGammai = data.Γi
	marshalledLocalTempData.K = data.K
	marshalledLocalTempData.G = data.G
	marshalledLocalTempData.Rhoi = data.𝜌i
	marshalledLocalTempData.Nui = data.𝜈i
	marshalledLocalTempData.KeyDerivationDelta = data.keyDerivationDelta

	// round 2
	marshalledLocalTempData.Gammai = data.𝛾i
	marshalledLocalTempData.DeltaShareBetas = data.DeltaShareBetas
	marshalledLocalTempData.DeltaShareBetaNegs = data.DeltaShareBetaNegs
	marshalledLocalTempData.DeltaMtASij = data.DeltaMtASij
	marshalledLocalTempData.DeltaMtARij = data.DeltaMtARij
	marshalledLocalTempData.Dji = data.Dji
	marshalledLocalTempData.ChiShareBetas = data.ChiShareBetas
	marshalledLocalTempData.DeltaMtAFji = data.DeltaMtAFji
	marshalledLocalTempData.ChiMtAF = data.ChiMtAF

	// round 3
	marshalledLocalTempData.BigGamma = data.Γ
	marshalledLocalTempData.DeltaShareAlphas = data.DeltaShareAlphas
	marshalledLocalTempData.ChiShareAlphas = data.ChiShareAlphas
	marshalledLocalTempData.Deltai = data.𝛿i
	marshalledLocalTempData.Chii = data.𝜒i
	marshalledLocalTempData.BigDeltai = data.Δi

	// round 4
	marshalledLocalTempData.M = data.m
	marshalledLocalTempData.BigR = data.BigR
	marshalledLocalTempData.Rx = data.Rx
	marshalledLocalTempData.SigmaShare = data.SigmaShare

	marshalledLocalTempData.SessionId = data.sessionId
	marshalledLocalTempData.R1msgG = data.r1msgG
	marshalledLocalTempData.R1msgK = data.r1msgK
	marshalledLocalTempData.R1msgPsiij = data.r1msg𝜓0ij
	marshalledLocalTempData.R2msgBigGammaShare = data.r2msgBigGammaShare
	marshalledLocalTempData.R2msgDeltaD = data.r2msgDeltaD
	marshalledLocalTempData.R2msgDeltaF = data.r2msgDeltaF
	marshalledLocalTempData.R2msgDeltaFjiPki = data.r2msgDeltaFjiPki
	marshalledLocalTempData.R2msgDeltaProof = data.r2msgDeltaProof
	marshalledLocalTempData.R2msgChiD = data.r2msgChiD
	marshalledLocalTempData.R2msgChiF = data.r2msgChiF
	marshalledLocalTempData.R2msgChiProof = data.r2msgChiProof
	marshalledLocalTempData.R2msgProofLogstar = data.r2msgProofLogstar
	marshalledLocalTempData.R3msgSigmaj = data.r3msg𝛿j
	marshalledLocalTempData.R3msgDeltaj = data.r3msgΔj
	marshalledLocalTempData.R3msgProofLogstar = data.r3msgProofLogstar
	marshalledLocalTempData.R4msgSigmaj = data.r4msg𝜎j
	marshalledLocalTempData.R4msgAborting = data.r4msgAborting
	marshalledLocalTempData.R5msgGammaj = data.r5msg𝛾j
	marshalledLocalTempData.R5msgsji = data.r5msgsji
	marshalledLocalTempData.R5msgBetaPrimeji = data.r5msg𝛽ʹji
	marshalledLocalTempData.R6msgH = data.r6msgH
	marshalledLocalTempData.R6msgProofMul = data.r6msgProofMul
	marshalledLocalTempData.R6msgProofDec = data.r6msgProofDec
	marshalledLocalTempData.R6msgDeltaShareEnc = data.r6msgDeltaShareEnc
	marshalledLocalTempData.R6msgEncryptedValueSum = data.r6msgEncryptedValueSum
	return marshalledLocalTempData
}

func MarshalledToLocalTempData(marshalledLocalTempData *MarshalledLocalTempData) *localTempData {
	data := &localTempData{}

	data.w = marshalledLocalTempData.W
	data.BigWs = marshalledLocalTempData.BigWs
	data.ki = marshalledLocalTempData.Ki

	data.Γi = marshalledLocalTempData.BigGammai
	data.K = marshalledLocalTempData.K
	data.G = marshalledLocalTempData.G
	data.𝜌i = marshalledLocalTempData.Rhoi
	data.𝜈i = marshalledLocalTempData.Nui
	data.keyDerivationDelta = marshalledLocalTempData.KeyDerivationDelta

	// round 2
	data.𝛾i = marshalledLocalTempData.Gammai
	data.DeltaShareBetas = marshalledLocalTempData.DeltaShareBetas
	data.DeltaShareBetaNegs = marshalledLocalTempData.DeltaShareBetaNegs
	data.DeltaMtASij = marshalledLocalTempData.DeltaMtASij
	data.DeltaMtARij = marshalledLocalTempData.DeltaMtARij
	data.Dji = marshalledLocalTempData.Dji
	data.ChiShareBetas = marshalledLocalTempData.ChiShareBetas
	data.DeltaMtAFji = marshalledLocalTempData.DeltaMtAFji
	data.ChiMtAF = marshalledLocalTempData.ChiMtAF

	// round 3
	data.Γ = marshalledLocalTempData.BigGamma
	data.DeltaShareAlphas = marshalledLocalTempData.DeltaShareAlphas
	data.ChiShareAlphas = marshalledLocalTempData.ChiShareAlphas
	data.𝛿i = marshalledLocalTempData.Deltai
	data.𝜒i = marshalledLocalTempData.Chii
	data.Δi = marshalledLocalTempData.BigDeltai

	// round 4
	data.m = marshalledLocalTempData.M
	data.BigR = marshalledLocalTempData.BigR
	data.Rx = marshalledLocalTempData.Rx
	data.SigmaShare = marshalledLocalTempData.SigmaShare

	data.sessionId = marshalledLocalTempData.SessionId
	data.r1msgG = marshalledLocalTempData.R1msgG
	data.r1msgK = marshalledLocalTempData.R1msgK
	data.r1msg𝜓0ij = marshalledLocalTempData.R1msgPsiij
	data.r2msgBigGammaShare = marshalledLocalTempData.R2msgBigGammaShare
	data.r2msgDeltaD = marshalledLocalTempData.R2msgDeltaD
	data.r2msgDeltaF = marshalledLocalTempData.R2msgDeltaF
	data.r2msgDeltaFjiPki = marshalledLocalTempData.R2msgDeltaFjiPki
	data.r2msgDeltaProof = marshalledLocalTempData.R2msgDeltaProof
	data.r2msgChiD = marshalledLocalTempData.R2msgChiD
	data.r2msgChiF = marshalledLocalTempData.R2msgChiF
	data.r2msgChiProof = marshalledLocalTempData.R2msgChiProof
	data.r2msgProofLogstar = marshalledLocalTempData.R2msgProofLogstar
	data.r3msg𝛿j = marshalledLocalTempData.R3msgSigmaj
	data.r3msgΔj = marshalledLocalTempData.R3msgDeltaj
	data.r3msgProofLogstar = marshalledLocalTempData.R3msgProofLogstar
	data.r4msg𝜎j = marshalledLocalTempData.R4msgSigmaj
	data.r4msgAborting = marshalledLocalTempData.R4msgAborting
	data.r5msg𝛾j = marshalledLocalTempData.R5msgGammaj
	data.r5msgsji = marshalledLocalTempData.R5msgsji
	data.r5msg𝛽ʹji = marshalledLocalTempData.R5msgBetaPrimeji
	data.r6msgH = marshalledLocalTempData.R6msgH
	data.r6msgProofMul = marshalledLocalTempData.R6msgProofMul
	data.r6msgProofDec = marshalledLocalTempData.R6msgProofDec
	data.r6msgDeltaShareEnc = marshalledLocalTempData.R6msgDeltaShareEnc
	data.r6msgEncryptedValueSum = marshalledLocalTempData.R6msgEncryptedValueSum
	return data
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
	tempData := MarshalledToLocalTempData(&marshalledStatefulPartyData.TheMarshalledLocalTempData)
	p.startRndNum = marshalledStatefulPartyData.StartRndNum
	p.temp = *tempData
	return true, nil
}

func (p *LocalStatefulParty) Dehydrate() (string, *tss.Error) {
	tempData := LocalTempDataToMarshalled(&p.temp)
	marshalledStatefulPartyData := &MarshalledStatefulPartyData{
		TheMarshalledLocalTempData: tempData,
		StartRndNum:                p.Round().RoundNumber(),
	}
	bz, err := json.Marshal(marshalledStatefulPartyData)
	if err != nil {
		return "", p.WrapError(err)
	}
	return string(bz[:]), nil
}

func (p *LocalStatefulParty) Restart(roundNumber int, marshalledPartyState string) *tss.Error {
	p.Lock()
	defer p.Unlock()
	if p.PartyID() == nil || !p.PartyID().ValidateBasic() {
		return p.WrapError(fmt.Errorf("could not start. this party has an invalid PartyID: %+v", p.PartyID()))
	}
	if p.Round() != nil {
		return p.WrapError(errors.New("could not start. this party is in an unexpected state. use the constructor and Start()"))
	}
	if marshalledPartyState != "" {
		_, errH := p.Hydrate(marshalledPartyState)
		if errH != nil {
			return errH
		}
	}
	p.startRndNum = roundNumber
	round := p.FirstRound()
	if err := p.SetRound(round); err != nil {
		return err
	}
	common.Logger.Infof("party %s (%p): %s round %d restarting", p.Round().Params().PartyID(), p, TaskName, roundNumber)
	defer func() {
		common.Logger.Debugf("party %s (%p): %s round %d finished", p.Round().Params().PartyID(), p, TaskName, roundNumber)
	}()
	return p.Round().Start()
}
