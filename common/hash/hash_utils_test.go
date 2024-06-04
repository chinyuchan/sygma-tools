package hash_test

import (
	"crypto/sha512"
	"reflect"
	"testing"

	"github.com/binance-chain/tss-lib/common"
	. "github.com/binance-chain/tss-lib/common/hash"
	big "github.com/binance-chain/tss-lib/common/int"
)

func TestRejectionSample(t *testing.T) {
	curveQ := common.GetRandomPrimeInt(256)
	smallQ := common.MustGetRandomInt(64)
	hash := SHA256iOne(big.NewInt(123))
	temp := sha512.Sum512(big.NewInt(123).Bytes())
	hashX2 := new(big.Int).SetBytes(temp[:])
	type args struct {
		q     *big.Int
		eHash *big.Int
	}
	tests := []struct {
		name          string
		args          args
		want          *big.Int
		wantMaxBitLen int
		notEqual      bool
	}{{
		name:          "with 256-bit curve order",
		args:          args{curveQ, hash},
		want:          RejectionSample(curveQ, hash),
		wantMaxBitLen: 256,
	}, {
		name:          "with 256-bit curve order, large hash",
		args:          args{curveQ, hashX2},
		want:          RejectionSample(curveQ, hashX2),
		wantMaxBitLen: 256,
	}, {
		name:          "with 64-bit q",
		args:          args{smallQ, hash},
		want:          RejectionSample(smallQ, hash),
		wantMaxBitLen: 64,
	}, {
		name:          "inequality with 256-bit curve order and different input",
		args:          args{smallQ, hash},
		want:          RejectionSample(common.MustGetRandomInt(256), hash),
		wantMaxBitLen: 256,
		notEqual:      true,
	}, {
		name:          "inequality with 64-bit curve order and different input",
		args:          args{smallQ, hash},
		want:          RejectionSample(common.MustGetRandomInt(64), hash),
		wantMaxBitLen: 64,
		notEqual:      true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RejectionSample(tt.args.q, tt.args.eHash)
			if !tt.notEqual && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RejectionSample() = %v, want %v", got, tt.want)
			}
			if tt.wantMaxBitLen < got.BitLen() { // leading zeros not counted
				t.Errorf("RejectionSample() = bitlen %d, want %d", got.BitLen(), tt.wantMaxBitLen)
			}
		})
	}
}
