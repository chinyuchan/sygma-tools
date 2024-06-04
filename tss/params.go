// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package tss

import (
	"crypto/elliptic"
	"errors"
	"time"

	"github.com/binance-chain/tss-lib/common"
)

type (
	Parameters struct {
		ec                      elliptic.Curve
		partyID                 *PartyID
		parties                 *PeerContext
		partyCount              int
		threshold               int
		safePrimeGenTimeout     time.Duration
		unsafeKGIgnoreH1H2Dupes bool
	}

	ReSharingParameters struct {
		*Parameters
		newParties    *PeerContext
		newPartyCount int
		newThreshold  int
	}
)

const (
	defaultSafePrimeGenTimeout = 5 * time.Minute
)

func NewParameters(ec elliptic.Curve, ctx *PeerContext, partyID *PartyID, partyCount, threshold int, optionalSafePrimeGenTimeout ...time.Duration) (*Parameters, error) {
	var safePrimeGenTimeout time.Duration
	if 0 < len(optionalSafePrimeGenTimeout) {
		if 1 < len(optionalSafePrimeGenTimeout) {
			return nil, errors.New("GeneratePreParams: expected 0 or 1 item in `optionalSafePrimeGenTimeout`")
		}
		safePrimeGenTimeout = optionalSafePrimeGenTimeout[0]
	} else {
		safePrimeGenTimeout = defaultSafePrimeGenTimeout
	}
	params := &Parameters{
		ec:                  ec,
		parties:             ctx,
		partyID:             partyID,
		partyCount:          partyCount,
		threshold:           threshold,
		safePrimeGenTimeout: safePrimeGenTimeout,
	}
	return params, params.Validate()
}

func (params *Parameters) Validate() error {
	if params.threshold >= params.partyCount {
		return errors.New("TSS Parameters: threshold >= partyCount (dishonest majority assumption)")
	}
	if params.partyCount < 2 {
		return errors.New("TSS Parameters: partyCount < 2")
	}
	if params.threshold < 1 {
		return errors.New("TSS Parameters: threshold < 1")
	}
	return nil
}

func (params *Parameters) EC() elliptic.Curve {
	return params.ec
}

func (params *Parameters) Parties() *PeerContext {
	return params.parties
}

func (params *Parameters) PartyID() *PartyID {
	return params.partyID
}

func (params *Parameters) PartyCount() int {
	return params.partyCount
}

func (params *Parameters) Threshold() int {
	return params.threshold
}

func (params *Parameters) SafePrimeGenTimeout() time.Duration {
	return params.safePrimeGenTimeout
}

// Getter. The H1, H2 dupe check is disabled during some benchmarking scenarios to allow reuse of pre-params.
func (params *Parameters) UNSAFE_KGIgnoreH1H2Dupes() bool {
	return params.unsafeKGIgnoreH1H2Dupes
}

// Setter. The H1, H2 dupe check is disabled during some benchmarking scenarios to allow reuse of pre-params.
func (params *Parameters) UNSAFE_setKGIgnoreH1H2Dupes(unsafeKGIgnoreH1H2Dupes bool) {
	if unsafeKGIgnoreH1H2Dupes {
		common.Logger.Warn("UNSAFE_setKGIgnoreH1H2Dupes() has been called; do not use these shares in production.")
	}
	params.unsafeKGIgnoreH1H2Dupes = unsafeKGIgnoreH1H2Dupes
}

// ----- //

// Exported, used in `tss` client
func NewReSharingParameters(ec elliptic.Curve, ctx, newCtx *PeerContext, partyID *PartyID, partyCount, threshold, newPartyCount, newThreshold int) (*ReSharingParameters, error) {
	params, _ := NewParameters(ec, ctx, partyID, partyCount, threshold)
	rsParams := &ReSharingParameters{
		Parameters:    params,
		newParties:    newCtx,
		newPartyCount: newPartyCount,
		newThreshold:  newThreshold,
	}
	return rsParams, rsParams.Validate()
}

func (rsParams *ReSharingParameters) Validate() error {
	if err := rsParams.Parameters.Validate(); err != nil {
		return err
	}
	if rsParams.newThreshold >= rsParams.newPartyCount {
		return errors.New("TSS Parameters: newThreshold >= newPartyCount (dishonest majority assumption)")
	}
	if rsParams.newPartyCount < 2 {
		return errors.New("TSS ReSharingParameters: newPartyCount < 2")
	}
	if rsParams.newThreshold < 1 {
		return errors.New("TSS ReSharingParameters: newThreshold < 1")
	}
	return nil
}

func (rsParams *ReSharingParameters) OldParties() *PeerContext {
	return rsParams.Parties() // wr use the original method for old parties
}

func (rsParams *ReSharingParameters) OldPartyCount() int {
	return rsParams.partyCount
}

func (rsParams *ReSharingParameters) NewParties() *PeerContext {
	return rsParams.newParties
}

func (rsParams *ReSharingParameters) NewPartyCount() int {
	return rsParams.newPartyCount
}

func (rsParams *ReSharingParameters) NewThreshold() int {
	return rsParams.newThreshold
}

func (rsParams *ReSharingParameters) OldAndNewParties() []*PartyID {
	return append(rsParams.OldParties().IDs(), rsParams.NewParties().IDs()...)
}

func (rsParams *ReSharingParameters) OldAndNewPartyCount() int {
	return rsParams.OldPartyCount() + rsParams.NewPartyCount()
}

func (rsParams *ReSharingParameters) IsOldCommittee() bool {
	partyID := rsParams.partyID
	for _, Pj := range rsParams.parties.IDs() {
		if partyID.KeyInt().Cmp(Pj.KeyInt()) == 0 {
			return true
		}
	}
	return false
}

func (rsParams *ReSharingParameters) IsNewCommittee() bool {
	partyID := rsParams.partyID
	for _, Pj := range rsParams.newParties.IDs() {
		if partyID.KeyInt().Cmp(Pj.KeyInt()) == 0 {
			return true
		}
	}
	return false
}
