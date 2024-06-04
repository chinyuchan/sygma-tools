// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package common

import (
	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/ipfs/go-log"
)

var Logger = log.Logger("tss-lib")

func FormatBigInt(a *big.Int) string {
	if a == nil {
		return "<nil>"
	}
	var aux = new(big.Int).SetInt64(0xFFFFFFFF)
	return func(i *big.Int) string {
		return new(big.Int).And(i, aux).Text(16)
	}(a)
}
