// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package crypto

import (
	"fmt"

	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/binance-chain/tss-lib/common"
)

func GenerateNTildei(safePrimes [2]*big.Int) (NTildei, h1i, h2i *big.Int, err error) {
	if safePrimes[0] == nil || safePrimes[1] == nil {
		return nil, nil, nil, fmt.Errorf("GenerateNTildei: needs two primes, got %v", safePrimes)
	}
	if !safePrimes[0].ProbablyPrime(30) || !safePrimes[1].ProbablyPrime(30) {
		return nil, nil, nil, fmt.Errorf("GenerateNTildei: expected two primes")
	}
	NTildei = new(big.Int).Mul(safePrimes[0], safePrimes[1])
	h1 := common.GetRandomGeneratorOfTheQuadraticResidue(NTildei)
	h2 := common.GetRandomGeneratorOfTheQuadraticResidue(NTildei)
	return NTildei, h1, h2, nil
}

func FormatECPoint(p *ECPoint) string {
	if p == nil {
		return "<nil>"
	}
	x := common.FormatBigInt(p.X())
	y := common.FormatBigInt(p.Y())
	return "(" + x + "," + y + ")"
}