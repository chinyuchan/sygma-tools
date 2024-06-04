// SPDX-License-Identifier: MIT
//
// Copyright (C) 2021 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

package hash

import (
	"encoding/binary"
	"errors"
)

var (
	errInputNegative  = errors.New("negative input")
	errInputLarge32   = errors.New("integer too large : > 2^32")
	errInputLarge     = errors.New("input is too high for length")
	errLengthNegative = errors.New("length is negative or 0")
	errLengthTooBig   = errors.New("requested length is > 4")
)

// i2osp 32 bit Integer to Octet Stream Primitive on maximum 4 bytes.
func i2osp(value, length int) []byte {
	if length <= 0 {
		panic(errLengthNegative)
	}

	if length > 4 {
		panic(errLengthTooBig)
	}

	out := make([]byte, 4)

	switch v := value; {
	case v < 0:
		panic(errInputNegative)
	case v >= 1<<(8*length):
		panic(errInputLarge)
	case length == 1:
		binary.BigEndian.PutUint16(out, uint16(v))

		return out[1:2]
	case length == 2:
		binary.BigEndian.PutUint16(out, uint16(v))
	case length == 3 || length == 4:
		binary.BigEndian.PutUint32(out, uint32(v))
	default:
		panic(errInputLarge32)
	}

	return out[:length]
}
