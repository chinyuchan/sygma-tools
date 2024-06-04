// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package common

import (
	"crypto/elliptic"
	"math/big"
)

func (x *ECPoint) ValidateBasic(ec elliptic.Curve) bool {
	return x != nil &&
		NonEmptyBytes(x.GetX()) &&
		NonEmptyBytes(x.GetY()) &&
		ec.IsOnCurve(
			new(big.Int).SetBytes(x.GetX()), new(big.Int).SetBytes(x.GetY()))
}
