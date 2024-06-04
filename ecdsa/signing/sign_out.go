// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package signing

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"

	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/btcsuite/btcd/btcec/v2"

	"github.com/binance-chain/tss-lib/common"
	int2 "github.com/binance-chain/tss-lib/common/int"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
)

func newRound5(params *tss.Parameters, key *keygen.LocalPartySaveData, data *common.SignatureData, temp *localTempData, out chan<- tss.Message, end chan<- common.SignatureData) tss.Round {
	return &signout{&sign4{&presign3{&presign2{&presign1{
		&base{params, key, data, temp, out, end, make([]bool, len(params.Parties().IDs())), false, 5}}}}, false}}
}

func (round *signout) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 5
	round.started = true
	round.resetOK()

	// Fig 8. Output. combine signature shares verify and output
	Sigma := round.temp.SigmaShare
	modN := int2.ModInt(big.Wrap(round.Params().EC().Params().N))
	for j := range round.Parties().IDs() {
		round.ok[j] = true
		if j == round.PartyID().Index {
			continue
		}
		Sigma = modN.Add(Sigma, round.temp.r4msg𝜎j[j])
	}
	recid := 0
	// byte v = if(R.X > curve.N) then 2 else 0) | (if R.Y.IsEven then 0 else 1);
	if round.temp.Rx.Cmp(big.Wrap(round.Params().EC().Params().N)) > 0 {
		recid = 2
	}
	if round.temp.BigR.Y().Bit(0) != 0 {
		recid |= 1
	}

	// This is copied from:
	// https://github.com/btcsuite/btcd/blob/c26ffa870fd817666a857af1bf6498fabba1ffe3/btcec/signature.go#L442-L444
	// This is needed because of tendermint checks here:
	// https://github.com/tendermint/tendermint/blob/d9481e3648450cb99e15c6a070c1fb69aa0c255b/crypto/secp256k1/secp256k1_nocgo.go#L43-L47
	halfN := new(big.Int).Rsh(big.Wrap(round.Params().EC().Params().N), 1)
	if Sigma.Cmp(halfN) > 0 {
		Sigma.Sub(big.Wrap(round.Params().EC().Params().N), Sigma)
		recid ^= 1
	}

	// save the signature for final output
	bitSizeInBytes := round.Params().EC().Params().BitSize / 8
	round.data.R = common.PadToLengthBytesInPlace(round.temp.Rx.Bytes(), bitSizeInBytes)
	round.data.S = common.PadToLengthBytesInPlace(Sigma.Bytes(), bitSizeInBytes)
	round.data.Signature = append(round.data.R, round.data.S...)
	round.data.SignatureRecovery = []byte{byte(recid)}
	round.data.M = round.temp.m.Bytes()

	// self-test ECDSA verify
	pk1 := round.key.ECDSAPub.ToBtcecPubKey()
	ok := ecdsa.Verify(pk1.ToECDSA(), round.temp.m.Bytes(), round.temp.Rx, Sigma)
	if !ok {
		return round.WrapError(fmt.Errorf("signature verification failed"))
	}

	// self-test EC recovery
	m, r, s, v := round.temp.m.Bytes(), round.data.R, round.data.S, round.data.SignatureRecovery
	expPub, gotPub, err := selfTestECRecovery(m, r, s, v, pk1)
	if err != nil {
		return round.WrapError(err)
	}
	if !bytes.Equal(expPub, gotPub) {
		return round.WrapError(fmt.Errorf("EC recovery self-test failed"))
	}
	round.end <- *round.data
	round.temp.G = nil
	round.temp.𝜈i = nil
	return nil
}

func selfTestECRecovery(msg, r, s, v []byte, pk *btcec.PublicKey) ([]byte, []byte, error) {
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = v[0] & 0x01
	expPub := pk.SerializeUncompressed()
	gotPub, err := crypto.Ecrecover(msg, sig)
	return expPub, gotPub, err
}

func (round *signout) CanAccept(msg tss.ParsedMessage) bool {
	// not expecting any incoming messages in this round
	return false
}

func (round *signout) Update() (bool, *tss.Error) {
	// not expecting any incoming messages in this round
	return false, nil
}

func (round *signout) NextRound() tss.Round {
	return nil // finished!
}
