// Copyright © 2021 Swingby

package signing

import (
	"crypto/elliptic"
	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/ckd"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg"
)

func UpdatePublicKeyAndAdjustBigXj(keyDerivationDelta *big.Int, keys []keygen.LocalPartySaveData,
	extendedChildPk *btcec.PublicKey, ec elliptic.Curve) error {
	var err error
	gDelta := crypto.ScalarBaseMult(ec, keyDerivationDelta)
	for k := range keys {
		keys[k].ECDSAPub, err = crypto.NewECPoint(ec, big.Wrap(extendedChildPk.X()), big.Wrap(extendedChildPk.Y()))
		if err != nil {
			common.Logger.Errorf("error creating new extended child public key")
			return err
		}
		// Suppose X_j has shamir shares X_j0,     X_j1,     ..., X_jn
		// So X_j + D has shamir shares  X_j0 + D, X_j1 + D, ..., X_jn + D
		for j := range keys[k].BigXj {
			keys[k].BigXj[j], err = keys[k].BigXj[j].Add(gDelta)
			if err != nil {
				common.Logger.Errorf("error in delta operation")
				return err
			}
		}
	}
	return nil
}

func derivingPubkeyFromPath(masterPub *crypto.ECPoint, chainCode []byte, path []uint32, ec elliptic.Curve) (*big.Int, *ckd.ExtendedKey, error) {
	// build ecdsa key pair
	pk := masterPub.ToBtcecPubKey()
	net := &chaincfg.MainNetParams
	extendedParentPk := &ckd.ExtendedKey{
		PublicKey:  pk,
		Depth:      0,
		ChildIndex: 0,
		ChainCode:  chainCode[:],
		ParentFP:   []byte{0x00, 0x00, 0x00, 0x00},
		Version:    net.HDPrivateKeyID[:],
	}
	return ckd.DeriveChildKeyFromHierarchy(path, extendedParentPk, big.Wrap(ec.Params().N), ec)
}