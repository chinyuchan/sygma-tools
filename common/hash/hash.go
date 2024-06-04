// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package hash

import (
	"crypto"
	_ "crypto/sha512"
	"encoding/binary"

	"github.com/binance-chain/tss-lib/common"

	big "github.com/binance-chain/tss-lib/common/int"
)

const (
	hashInputDelimiter = byte('$')
)

func SHA256(in ...[]byte) []byte {
	var data []byte
	state := crypto.SHA256.New()
	inLen := len(in)
	if inLen == 0 {
		return nil
	}
	bzSize := 0
	// prevent hash collisions with this prefix containing the block count
	inLenBz := make([]byte, 8) // 64-bits
	// converting between int and uint64 doesn't change the sign bit, but it may be interpreted as a larger value.
	// this prefix is never read/interpreted, so that doesn't matter.
	binary.LittleEndian.PutUint64(inLenBz, uint64(inLen))
	for _, bz := range in {
		bzSize += len(bz)
	}
	dataCap := len(inLenBz) + bzSize + inLen + (inLen * 8)
	data = make([]byte, 0, dataCap)
	data = append(data, inLenBz...)
	for _, bz := range in {
		data = append(data, bz...)
		data = append(data, hashInputDelimiter) // safety delimiter
		dataLen := make([]byte, 8)              // 64-bits
		binary.LittleEndian.PutUint64(dataLen, uint64(len(bz)))
		data = append(data, dataLen...) // Security audit: length of each byte buffer should be added after
		// each security delimiters in order to enforce proper domain separation
	}
	// n < len(data) or an error will never happen.
	// see: https://golang.org/pkg/hash/#Hash and https://github.com/golang/go/wiki/Hashing#the-hashhash-interface
	if _, err := state.Write(data); err != nil {
		common.Logger.Errorf("SHA256 Write() failed: %v", err)
		return nil
	}
	return state.Sum(nil)
}

func SHA256i(in ...*big.Int) *big.Int {
	var data []byte
	state := crypto.SHA256.New()
	inLen := len(in)
	if inLen == 0 {
		return nil
	}
	bzSize := 0
	// prevent hash collisions with this prefix containing the block count
	inLenBz := make([]byte, 8) // 64-bits
	// converting between int and uint64 doesn't change the sign bit, but it may be interpreted as a larger value.
	// this prefix is never read/interpreted, so that doesn't matter.
	binary.LittleEndian.PutUint64(inLenBz, uint64(inLen))
	ptrs := make([][]byte, inLen)
	for i, n := range in {
		ptrs[i] = append(n.Bytes(), byte(n.Sign()))
		bzSize += len(ptrs[i])
	}
	dataCap := len(inLenBz) + bzSize + inLen + (inLen * 8)
	data = make([]byte, 0, dataCap)
	data = append(data, inLenBz...)
	for i := range in {
		data = append(data, ptrs[i]...)
		data = append(data, hashInputDelimiter) // safety delimiter
		dataLen := make([]byte, 8)              // 64-bits
		binary.LittleEndian.PutUint64(dataLen, uint64(len(ptrs[i])))
		data = append(data, dataLen...) // Security audit: length of each byte buffer should be added after
		// each security delimiters in order to enforce proper domain separation
	}
	// n < len(data) or an error will never happen.
	// see: https://golang.org/pkg/hash/#Hash and https://github.com/golang/go/wiki/Hashing#the-hashhash-interface
	if _, err := state.Write(data); err != nil {
		common.Logger.Errorf("SHA256i Write() failed: %v", err)
		return nil
	}
	return new(big.Int).SetBytes(state.Sum(nil))
}

func SHA256iOne(in *big.Int) *big.Int {
	var data []byte
	state := crypto.SHA256.New()
	if in == nil {
		return nil
	}
	data = append(in.Bytes(), byte(in.Sign()))
	// n < len(data) or an error will never happen.
	// see: https://golang.org/pkg/hash/#Hash and https://github.com/golang/go/wiki/Hashing#the-hashhash-interface
	if _, err := state.Write(data); err != nil {
		common.Logger.Errorf("SHA256iOne Write() failed: %v", err)
		return nil
	}
	return new(big.Int).SetBytes(state.Sum(nil))
}
