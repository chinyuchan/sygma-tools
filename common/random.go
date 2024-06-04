// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package common

import (
	"crypto/rand"
	"fmt"
	gbig "math/big"

	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/pkg/errors"
)

const (
	mustGetRandomIntMaxBits = 5000
)

var (
	zero = big.NewInt(0)
	one  = big.NewInt(1)
	two  = big.NewInt(2)
)

// MustGetRandomInt panics if it is unable to gather entropy from `rand.Reader` or when `bits` is <= 0
func MustGetRandomInt(minBits int, upper ...*big.Int) *big.Int {
	if minBits <= 0 || mustGetRandomIntMaxBits < minBits {
		panic(fmt.Errorf("MustGetRandomInt: minBits should be positive, non-zero and less than %d", mustGetRandomIntMaxBits))
	}
	// Max random value e.g. 2^256 - 1
	max := new(big.Int)
	if len(upper) == 0 {
		max = max.Exp(two, big.NewInt(uint64(minBits)), nil).Sub(max, one)
	} else {
		max = upper[0]
	}
	maxBI := max

	// Generate cryptographically strong pseudo-random int between 0 - max
	var n *gbig.Int
	var err error
	for do := true; do; do = n.BitLen() < minBits {
		if n, err = rand.Int(rand.Reader, maxBI); err != nil {
			panic(errors.Wrap(err, "rand.Int failure in MustGetRandomInt!"))
		}
	}
	return big.Wrap(n)
}

func GetRandomPositiveInt(upper *big.Int) *big.Int {
	if upper == nil || zero.Cmp(upper) != -1 {
		return nil
	}
	var try *big.Int
	for {
		try = MustGetRandomInt(upper.BitLen()-1, upper)
		if try.Cmp(upper) < 0 && try.Cmp(zero) > 0 {
			break
		}
	}
	return try
}

func GetBigRandomPositiveInt(upper *big.Int, minBits int) *big.Int {
	if upper == nil || zero.Cmp(upper) != -1 || minBits < 8 || upper.BitLen() < minBits {
		return nil
	}
	var try *big.Int
	maxRetries := 100
	ok := false
	for i := 0; i < maxRetries; i++ {
		try = MustGetRandomInt(minBits, upper)
		if try.Cmp(upper) < 0 && try.Cmp(zero) >= 0 {
			ok = true
			break
		}
	}
	if !ok {
		return nil
	}
	return try
}

func GetRandomPrimeInt(minBits int) *big.Int {
	if minBits <= 0 {
		return nil
	}
	try, err := rand.Prime(rand.Reader, minBits)
	if err != nil ||
		try.Cmp(zero) == 0 {
		// fallback to older method
		for {
			try = MustGetRandomInt(minBits)
			if probablyPrime(big.Wrap(try)) {
				break
			}
		}
	}
	return big.Wrap(try)
}

// Generate a random element in the group of all the elements in Z/nZ that
// has a multiplicative inverse.
func GetRandomPositiveRelativelyPrimeInt(n *big.Int) *big.Int {
	if n == nil || zero.Cmp(n) != -1 {
		return nil
	}
	var try *big.Int
	for {
		try = MustGetRandomInt(n.BitLen())
		if IsNumberInMultiplicativeGroup(n, try) {
			break
		}
	}
	return try
}

func IsNumberInMultiplicativeGroup(n, v *big.Int) bool {
	if n == nil || v == nil || zero.Cmp(n) != -1 {
		return false
	}
	gcd := big.NewInt(0)
	return v.Cmp(n) < 0 && v.Cmp(one) >= 0 &&
		gcd.GCD(nil, nil, v, n).Cmp(one) == 0
}

//	Return a random generator of RQn with high probability.
//	THIS METHOD ONLY WORKS IF N IS THE PRODUCT OF TWO SAFE PRIMES!
//
// https://github.com/didiercrunch/paillier/blob/d03e8850a8e4c53d04e8016a2ce8762af3278b71/utils.go#L39
func GetRandomGeneratorOfTheQuadraticResidue(n *big.Int) *big.Int {
	f := GetRandomPositiveRelativelyPrimeInt(n)
	fSq := new(big.Int).Mul(f, f)
	return fSq.Mod(fSq, n)
}

func GetRandomQuadraticNonResidue(n *big.Int) *big.Int {
	for {
		w := GetRandomPositiveInt(n)
		if gbig.Jacobi(w, n) == -1 {
			return w
		}
	}
}
