// Optionally constant time big.Int (best effort)

package int

import (
	"math/big"
)

type (
	Int = big.Int
)

func NewInt(x uint64) *Int {
	return new(big.Int).SetUint64(x)
}

func Wrap(i2 *big.Int) *Int {
	return i2
}
