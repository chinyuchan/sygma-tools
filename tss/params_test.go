package tss

import (
	"crypto/elliptic"
	"testing"
)

func TestNewParameters(t *testing.T) {
	type args struct {
		ec         elliptic.Curve
		ctx        *PeerContext
		partyID    *PartyID
		partyCount int
		threshold  int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name:    "Good parameters",
		args:    args{ec, &PeerContext{}, &PartyID{}, 2, 1},
		wantErr: false,
	}, {
		name:    "Bad parameters: low partyCount",
		args:    args{ec, &PeerContext{}, &PartyID{}, 1, 1},
		wantErr: true,
	}, {
		name:    "Bad parameters: low threshold",
		args:    args{ec, &PeerContext{}, &PartyID{}, 5, 0},
		wantErr: true,
	}, {
		name:    "Bad parameters: threshold + 1 > partyCount",
		args:    args{ec, &PeerContext{}, &PartyID{}, 2, 2},
		wantErr: true,
	}, {
		name:    "Bad parameters: negative threshold",
		args:    args{ec, &PeerContext{}, &PartyID{}, 2, -1},
		wantErr: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewParameters(tt.args.ec, tt.args.ctx, tt.args.partyID, tt.args.partyCount, tt.args.threshold)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNewReSharingParameters(t *testing.T) {
	type args struct {
		ec            elliptic.Curve
		ctx           *PeerContext
		newCtx        *PeerContext
		partyID       *PartyID
		partyCount    int
		threshold     int
		newPartyCount int
		newThreshold  int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name:    "Good regular parameters",
		args:    args{ec, &PeerContext{}, &PeerContext{}, &PartyID{}, 2, 1, 3, 2},
		wantErr: false,
	}, {
		name:    "Good re-share parameters",
		args:    args{ec, &PeerContext{}, &PeerContext{}, &PartyID{}, 3, 2, 2, 1},
		wantErr: false,
	}, {
		name:    "Bad regular parameters: low partyCount",
		args:    args{ec, &PeerContext{}, &PeerContext{}, &PartyID{}, 1, 1, 3, 2},
		wantErr: true,
	}, {
		name:    "Bad regular parameters: low threshold",
		args:    args{ec, &PeerContext{}, &PeerContext{}, &PartyID{}, 5, 0, 3, 2},
		wantErr: true,
	}, {
		name:    "Bad regular parameters: threshold + 1 > partyCount",
		args:    args{ec, &PeerContext{}, &PeerContext{}, &PartyID{}, 2, 2, 3, 2},
		wantErr: true,
	}, {
		name:    "Bad re-share parameters: low newPartyCount",
		args:    args{ec, &PeerContext{}, &PeerContext{}, &PartyID{}, 3, 2, 1, 1},
		wantErr: true,
	}, {
		name:    "Bad re-share parameters: low newThreshold",
		args:    args{ec, &PeerContext{}, &PeerContext{}, &PartyID{}, 3, 2, 5, 0},
		wantErr: true,
	}, {
		name:    "Bad re-share parameters: newThreshold + 1 > newPartyCount",
		args:    args{ec, &PeerContext{}, &PeerContext{}, &PartyID{}, 3, 2, 2, 2},
		wantErr: true,
	}, {
		name:    "Bad re-share parameters: negative newThreshold",
		args:    args{ec, &PeerContext{}, &PeerContext{}, &PartyID{}, 3, 2, 2, -1},
		wantErr: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReSharingParameters(tt.args.ec, tt.args.ctx, tt.args.newCtx, tt.args.partyID, tt.args.partyCount, tt.args.threshold, tt.args.newPartyCount, tt.args.newThreshold)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReSharingParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
