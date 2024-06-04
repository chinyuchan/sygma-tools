// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package keygen

import (
	"crypto/elliptic"
	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/crypto/vss"
	zkpfac "github.com/binance-chain/tss-lib/crypto/zkp/fac"
	zkpmod "github.com/binance-chain/tss-lib/crypto/zkp/mod"
	zkpprm "github.com/binance-chain/tss-lib/crypto/zkp/prm"
	zkpsch "github.com/binance-chain/tss-lib/crypto/zkp/sch"
	"github.com/binance-chain/tss-lib/tss"
)

// These messages were generated from Protocol Buffers definitions into ecdsa-keygen.pb.go
// The following messages are registered on the Protocol Buffers "wire"

var (
	// Ensure that keygen messages implement ValidateBasic
	_ = []tss.MessageContent{
		(*KGRound1Message)(nil),
		(*KGRound2Message)(nil),
		(*KGRound3Message)(nil),
		(*KGRound4Message)(nil),
	}
)

// ----- //

func NewKGRound1Message(
	sessionId *big.Int,
	from *tss.PartyID,
	sid *big.Int,
	ViKeygen *big.Int,
	ssid *big.Int,
	ViKeyRefresh *big.Int,
) tss.ParsedMessage {
	meta := tss.MessageRouting{
		From:        from,
		IsBroadcast: true,
	}
	content := &KGRound1Message{
		Sid:      sid.Bytes(),
		ViKeygen: ViKeygen.Bytes(),
		// Refresh:
		Ssid:         ssid.Bytes(),
		ViKeyRefresh: ViKeyRefresh.Bytes(),
	}
	msg := tss.NewMessageWrapper(meta, content, sessionId)
	return tss.NewMessage(meta, content, msg)
}

func (m *KGRound1Message) ValidateBasic(_ elliptic.Curve) bool {
	return m != nil &&
		common.NonEmptyBytes(m.GetViKeygen()) &&
		common.NonEmptyBytes(m.GetViKeyRefresh()) &&
		common.NonEmptyBytes(m.GetSid()) &&
		common.NonEmptyBytes(m.GetSsid())
}

func (m *KGRound1Message) UnmarshalViKeygen() *big.Int {
	return new(big.Int).SetBytes(m.GetViKeygen())
}

func (m *KGRound1Message) UnmarshalViKeyRefresh() *big.Int {
	return new(big.Int).SetBytes(m.GetViKeyRefresh())
}

func (m *KGRound1Message) UnmarshalSid() *big.Int {
	return new(big.Int).SetBytes(m.GetSid())
}

func (m *KGRound1Message) UnmarshalSsid() *big.Int {
	return new(big.Int).SetBytes(m.GetSsid())
}

// ----- //

func NewKGRound2Message(
	sessionId *big.Int,
	from *tss.PartyID,
	vs vss.Vs,
	paillierPK *paillier.PublicKey,
	sid *big.Int, ridi *big.Int,
	XiKeygen *crypto.ECPoint, AiKeygen *crypto.ECPoint,
	ui *big.Int,
	ssid *big.Int,
	XiRefresh []*crypto.ECPoint, AiRefresh []*crypto.ECPoint, Yi *crypto.ECPoint,
	Bi *crypto.ECPoint, Ni *big.Int, si *big.Int, ti *big.Int,
	𝜓i *zkpprm.ProofPrm, 𝜌i *big.Int) (tss.ParsedMessage, error) {
	meta := tss.MessageRouting{
		From:        from,
		IsBroadcast: true,
	}
	vs_flat, err := crypto.FlattenECPoints(vs)
	if err != nil {
		return nil, err
	}
	vsbzs := make([][]byte, len(vs_flat))
	for i, item := range vs_flat {
		vsbzs[i] = item.Bytes()
	}
	AiKeygenBytes := AiKeygen.Bytes()
	XiKeygenBytes := XiKeygen.Bytes()

	AiRefreshPoints, err := crypto.FlattenECPoints(AiRefresh)
	if err != nil {
		return nil, err
	}
	AiRefreshBytes := make([][]byte, len(AiRefreshPoints))
	for k, p := range AiRefreshPoints {
		AiRefreshBytes[k] = p.Bytes()
	}
	XiRefreshPoints, err := crypto.FlattenECPoints(XiRefresh)
	if err != nil {
		return nil, err
	}
	XiRefreshBytes := make([][]byte, len(XiRefreshPoints))
	for k, p := range XiRefreshPoints {
		XiRefreshBytes[k] = p.Bytes()
	}
	YiBytes := Yi.Bytes()
	BiBytes := Bi.Bytes()
	𝜓iBytes := 𝜓i.Bytes()
	content := &KGRound2Message{
		Sid:       sid.Bytes(),
		Ridi:      ridi.Bytes(),
		Ui:        ui.Bytes(),
		PaillierN: paillierPK.N.Bytes(),
		AiKeygen:  AiKeygenBytes[:],
		XiKeygen:  XiKeygenBytes[:],
		Vs:        vsbzs[:],

		// Refresh:
		Ssid:      ssid.Bytes(),
		XiRefresh: XiRefreshBytes[:],
		AiRefresh: AiRefreshBytes[:],
		Yi:        YiBytes[:],
		Bi:        BiBytes[:],
		Ni:        Ni.Bytes(),
		Si:        si.Bytes(),
		Ti:        ti.Bytes(),
		PsiiProof: 𝜓iBytes[:],
		Rhoi:      𝜌i.Bytes(),
	}
	msg := tss.NewMessageWrapper(meta, content, sessionId)
	return tss.NewMessage(meta, content, msg), nil
}

func (m *KGRound2Message) ValidateBasic(ec elliptic.Curve) bool {
	if _, err := m.UnmarshalAiKeygen(ec); err != nil {
		return false
	}
	if _, err := m.UnmarshalAiRefresh(ec); err != nil {
		return false
	}
	if _, err := m.UnmarshalXiKeygen(ec); err != nil {
		return false
	}
	if _, err := m.UnmarshalXiRefresh(ec); err != nil {
		return false
	}
	return m != nil &&
		common.NonEmptyBytes(m.GetSid()) &&
		common.NonEmptyBytes(m.GetRidi()) &&
		common.NonEmptyBytes(m.GetUi()) &&
		common.NonEmptyBytes(m.GetPaillierN()) &&
		common.NonEmptyMultiBytes(m.GetXiKeygen()) &&
		common.NonEmptyMultiBytes(m.GetAiKeygen()) &&
		common.NonEmptyMultiBytes(m.GetVs()) &&
		// Refresh:
		common.NonEmptyBytes(m.GetSsid()) &&
		common.NonEmptyMultiBytes(m.GetXiRefresh()) &&
		common.NonEmptyMultiBytes(m.GetAiRefresh()) &&
		common.NonEmptyMultiBytes(m.GetYi()) &&
		common.NonEmptyMultiBytes(m.GetBi()) &&
		common.NonEmptyBytes(m.GetNi()) &&
		common.NonEmptyBytes(m.GetSi()) &&
		common.NonEmptyBytes(m.GetTi()) &&
		common.NonEmptyMultiBytes(m.GetPsiiProof(), zkpprm.ProofPrmBytesParts) &&
		common.NonEmptyBytes(m.GetRhoi())
}

func (m *KGRound2Message) UnmarshalSid() *big.Int {
	return new(big.Int).SetBytes(m.GetSid())
}

func (m *KGRound2Message) UnmarshalRidi() *big.Int {
	return new(big.Int).SetBytes(m.GetRidi())
}

func (m *KGRound2Message) UnmarshalUi() *big.Int {
	return new(big.Int).SetBytes(m.GetUi())
}

func (m *KGRound2Message) UnmarshalPaillierPK() *paillier.PublicKey {
	return &paillier.PublicKey{N: new(big.Int).SetBytes(m.GetPaillierN())}
}

func (m *KGRound2Message) UnmarshalAiKeygen(ec elliptic.Curve) (*crypto.ECPoint, error) {
	return crypto.NewECPointFromBytes(ec, m.GetAiKeygen())
}

func (m *KGRound2Message) UnmarshalXiKeygen(ec elliptic.Curve) (*crypto.ECPoint, error) {
	return crypto.NewECPointFromBytes(ec, m.GetXiKeygen())
}

func (m *KGRound2Message) UnmarshalVs(ec elliptic.Curve) ([]*crypto.ECPoint, error) {
	bzs := m.GetVs()
	vs_points := make([]*big.Int, len(bzs))
	for i, item := range m.GetVs() {
		vs_points[i] = new(big.Int).SetBytes(item)
	}
	vs, err := crypto.UnFlattenECPoints(ec, vs_points)
	if err != nil {
		return nil, err
	}
	return vs, nil
}

func (m *KGRound2Message) UnmarshalSsid() *big.Int {
	return new(big.Int).SetBytes(m.GetSsid())
}

func (m *KGRound2Message) UnmarshalXiRefresh(ec elliptic.Curve) ([]*crypto.ECPoint, error) {
	in := make([]*big.Int, len(m.GetXiRefresh()))
	for i, buff := range m.GetXiRefresh() {
		a := big.NewInt(0).SetBytes(buff)
		in[i] = a
	}
	points, err := crypto.UnFlattenECPoints(ec, in, false)
	if err != nil {
		return nil, err
	}
	return points, nil
}

func (m *KGRound2Message) UnmarshalAiRefresh(ec elliptic.Curve) ([]*crypto.ECPoint, error) {
	in := make([]*big.Int, len(m.GetAiRefresh()))
	for i, buff := range m.GetAiRefresh() {
		a := big.NewInt(0).SetBytes(buff)
		in[i] = a
	}
	points, err := crypto.UnFlattenECPoints(ec, in, false)
	if err != nil {
		return nil, err
	}
	return points, nil
}

func (m *KGRound2Message) UnmarshalYi(ec elliptic.Curve) (*crypto.ECPoint, error) {
	return crypto.NewECPointFromBytes(ec, m.GetYi())
}

func (m *KGRound2Message) UnmarshalBi(ec elliptic.Curve) (*crypto.ECPoint, error) {
	return crypto.NewECPointFromBytes(ec, m.GetBi())
}

func (m *KGRound2Message) UnmarshalNi() *big.Int {
	return new(big.Int).SetBytes(m.GetNi())
}

func (m *KGRound2Message) UnmarshalSi() *big.Int {
	return new(big.Int).SetBytes(m.GetSi())
}

func (m *KGRound2Message) UnmarshalTi() *big.Int {
	return new(big.Int).SetBytes(m.GetTi())
}

func (m *KGRound2Message) Unmarshal𝜓i() (*zkpprm.ProofPrm, error) {
	return zkpprm.NewProofFromBytes(m.GetPsiiProof())
}

func (m *KGRound2Message) Unmarshal𝜌i() *big.Int {
	return new(big.Int).SetBytes(m.GetRhoi())
}

// ----- //

func NewKGRound3Message(
	sessionId *big.Int,
	to, from *tss.PartyID,
	sid *big.Int,
	𝜓Schi *zkpsch.ProofSch,
	Cvssji, randomnessCvssji *big.Int,
	// refresh:
	ssid *big.Int,
	𝜓Modi *zkpmod.ProofMod, 𝜙ji *zkpfac.ProofFac,
	ᴨi *zkpsch.ProofSch,
	Czeroji, randomnessCzeroji *big.Int, 𝜓ji *zkpsch.ProofSch,
) tss.ParsedMessage {
	meta := tss.MessageRouting{
		From:        from,
		To:          []*tss.PartyID{to},
		IsBroadcast: false,
	}
	𝜓SchiBytes := 𝜓Schi.Bytes()
	𝜙jiBytes := 𝜙ji.Bytes()
	𝜓ModiBytes := 𝜓Modi.Bytes()
	ᴨiBytes := ᴨi.Bytes()
	𝜓jiBytes := 𝜓ji.Bytes()
	content := &KGRound3Message{
		Sid:          sid.Bytes(),
		PsiiSchProof: 𝜓SchiBytes[:],
		Cvssji:       Cvssji.Bytes(),
		RandCvssji:   randomnessCvssji.Bytes(),
		// Refresh:
		Ssid:         ssid.Bytes(),
		PsiiModProof: 𝜓ModiBytes[:],
		PhijiProof:   𝜙jiBytes[:],
		PiiProof:     ᴨiBytes[:],
		Czeroji:      Czeroji.Bytes(),
		RandCzeroji:  randomnessCzeroji.Bytes(),
		PsijiProof:   𝜓jiBytes[:],
	}
	msg := tss.NewMessageWrapper(meta, content, sessionId)
	return tss.NewMessage(meta, content, msg)
}

func (m *KGRound3Message) ValidateBasic(_ elliptic.Curve) bool {
	return m != nil &&
		common.NonEmptyBytes(m.GetSid()) &&
		common.NonEmptyMultiBytes(m.PsiiSchProof, zkpsch.ProofSchBytesParts) &&
		// Refresh:
		common.NonEmptyBytes(m.GetSsid()) &&
		common.AnyNonEmptyMultiByte(m.GetPsiiModProof(), zkpmod.ProofModBytesParts) &&
		common.NonEmptyMultiBytes(m.GetPhijiProof(), zkpfac.ProofFacBytesParts) &&
		common.NonEmptyMultiBytes(m.GetPiiProof(), zkpsch.ProofSchBytesParts) &&
		common.NonEmptyBytes(m.GetCvssji()) &&
		common.NonEmptyBytes(m.GetRandCvssji()) &&
		common.NonEmptyBytes(m.GetCzeroji()) &&
		common.NonEmptyBytes(m.GetRandCzeroji()) &&
		common.NonEmptyMultiBytes(m.GetPsijiProof(), zkpsch.ProofSchBytesParts)
}

func (m *KGRound3Message) UnmarshalSid() *big.Int {
	return new(big.Int).SetBytes(m.GetSid())
}

func (m *KGRound3Message) Unmarshal𝜓SchProof(ec elliptic.Curve) (*zkpsch.ProofSch, error) {
	return zkpsch.NewProofFromBytes(ec, m.GetPsiiSchProof())
}

func (m *KGRound3Message) UnmarshalSsid() *big.Int {
	return new(big.Int).SetBytes(m.GetSsid())
}

func (m *KGRound3Message) Unmarshal𝜓ModProof() (*zkpmod.ProofMod, error) {
	return zkpmod.NewProofFromBytes(m.GetPsiiModProof())
}

func (m *KGRound3Message) Unmarshal𝜙ji() (*zkpfac.ProofFac, error) {
	return zkpfac.NewProofFromBytes(m.GetPhijiProof())
}

func (m *KGRound3Message) Unmarshalᴨi(ec elliptic.Curve) (*zkpsch.ProofSch, error) {
	return zkpsch.NewProofFromBytes(ec, m.GetPiiProof())
}

func (m *KGRound3Message) UnmarshalCzeroji() *big.Int {
	return new(big.Int).SetBytes(m.GetCzeroji())
}

func (m *KGRound3Message) UnmarshalRandomnessCzeroji() *big.Int {
	return new(big.Int).SetBytes(m.GetRandCzeroji())
}

func (m *KGRound3Message) UnmarshalCvssji() *big.Int {
	return new(big.Int).SetBytes(m.GetCvssji())
}

func (m *KGRound3Message) UnmarshalRandomnessCvssji() *big.Int {
	return new(big.Int).SetBytes(m.GetRandCvssji())
}

func (m *KGRound3Message) Unmarshal𝜓ji(ec elliptic.Curve) (*zkpsch.ProofSch, error) {
	return zkpsch.NewProofFromBytes(ec, m.GetPsijiProof())
}

// ----- //

func NewKGRound4Message(
	sessionId *big.Int,
	from *tss.PartyID,
	sid *big.Int,
	abort bool,
	mu *big.Int,
	culpritIndex int,
	Cji, xji *big.Int,
) tss.ParsedMessage {
	meta := tss.MessageRouting{
		From:        from,
		IsBroadcast: true,
	}

	content := &KGRound4Message{
		Sid:       sid.Bytes(),
		Abort:     abort,
		Mu:        mu.Bytes(),
		CulpritPj: int32(culpritIndex),
		Cji:       Cji.Bytes(),
		Xji:       xji.Bytes(),
	}
	msg := tss.NewMessageWrapper(meta, content, sessionId)
	return tss.NewMessage(meta, content, msg)
}

func (m *KGRound4Message) ValidateBasic(_ elliptic.Curve) bool {
	return m != nil &&
		common.NonEmptyBytes(m.GetSid()) &&
		common.NonEmptyBytes(m.GetMu()) &&
		common.NonEmptyBytes(m.GetCji()) &&
		common.NonEmptyBytes(m.GetXji())
}

func (m *KGRound4Message) UnmarshalSid() *big.Int {
	return new(big.Int).SetBytes(m.GetSid())
}

func (m *KGRound4Message) UnmarshalMu() *big.Int {
	return new(big.Int).SetBytes(m.GetMu())
}

func (m *KGRound4Message) UnmarshalCji() *big.Int {
	return new(big.Int).SetBytes(m.GetCji())
}

func (m *KGRound4Message) UnmarshalXji() *big.Int {
	return new(big.Int).SetBytes(m.GetXji())
}
