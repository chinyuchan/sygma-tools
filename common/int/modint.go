// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package int

import (
	"math/big"
)

// modInt is a *big.Int that performs all of its arithmetic with modular reduction.
type modInt Int

func ModInt(mod *Int) *modInt {
	i := new(big.Int).SetBytes(mod.Bytes())
	return (*modInt)(i)
}

func (mi *modInt) Add(x, y *Int) *Int {
	i := new(Int)
	i.Add(x, y)
	return i.Mod(i, mi.int())
}

func (mi *modInt) Sub(x, y *Int) *Int {
	i := new(Int)
	i.Sub(x, y)
	return i.Mod(i, mi.int())
}

func (mi *modInt) Div(x, y *Int) *Int {
	i := new(Int)
	i.Div(x, y)
	return i.Mod(i, mi.int())
}

func (mi *modInt) Mul(x, y *Int) *Int {
	i := new(big.Int)
	i.Mul(x, y)
	return i.Mod(i, mi.int())
}

func (mi *modInt) Exp(x, y *Int) *Int {
	return new(Int).Exp(x, y, mi.int())
}

func (mi *modInt) Neg(x *Int) *Int {
	i := new(Int)
	i.Neg(x)
	return i.Mod(i, mi.int())
}

func (mi *modInt) Inverse(g *Int) *Int {
	return new(Int).ModInverse(g, mi.int())
}

func (mi *modInt) Sqrt(x *Int) *Int {
	return new(Int).ModSqrt(x, mi.int())
}

func (mi *modInt) Size() int {
	return mi.int().BitLen() / 8
}

func (mi *modInt) int() *Int {
	return new(Int).Set((*Int)(mi))
}
