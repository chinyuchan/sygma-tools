package hash_test

import (
	"testing"

	. "github.com/binance-chain/tss-lib/common"
	. "github.com/binance-chain/tss-lib/common/hash"
	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/stretchr/testify/assert"
)

func TestSHA256(t *testing.T) {
	input := [][]byte{[]byte("abc"), []byte("def"), []byte("ghi")}
	input2 := [][]byte{[]byte("abc"), []byte("def"), []byte("gh")}
	type args struct {
		in [][]byte
	}
	tests := []struct {
		name     string
		args     args
		want     []byte
		wantDiff bool
		wantLen  int
	}{{
		name:    "same inputs produce the same hash",
		args:    args{input},
		want:    SHA256(input...),
		wantLen: 256 / 8,
	}, {
		name:     "different inputs produce a differing hash",
		args:     args{input2},
		want:     SHA256(input...),
		wantDiff: true,
		wantLen:  256 / 8,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SHA256(tt.args.in...)
			if tt.wantDiff {
				if !assert.NotEqualf(t, tt.want, got, "SHA256(%v)", tt.args.in) {
					t.Errorf("SHA256() = %v, do not want %v", got, tt.want)
				}
			} else {
				if !assert.Equalf(t, tt.want, got, "SHA256(%v)", tt.args.in) {
					t.Errorf("SHA256() = %v, want %v", got, tt.want)
				}
			}
			if tt.wantLen != len(got) {
				t.Errorf("SHA256() = bitlen %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestSHA256i(t *testing.T) {
	input := ByteSlicesToBigInts([][]byte{[]byte("abc"), []byte("def"), []byte("ghi")})
	input2 := ByteSlicesToBigInts([][]byte{[]byte("abc"), []byte("def"), []byte("gh")})
	input3 := new(big.Int).SetBytes([]byte("abc"))
	t.Logf("%d", input3.Int64())
	t.Logf("%d", new(big.Int).Neg(input3).Int64())
	type args struct {
		in []*big.Int
	}
	tests := []struct {
		name     string
		args     args
		want     *big.Int
		wantDiff bool
	}{{
		name: "same inputs produce the same hash",
		args: args{input},
		want: SHA256i(input...),
	}, {
		name:     "different inputs produce a differing hash",
		args:     args{input2},
		want:     SHA256i(input...),
		wantDiff: true,
	}, {
		name:     "different inputs produce a differing hash: Hash(-a) != Hash(a)",
		args:     args{[]*big.Int{new(big.Int).Neg(input3)}},
		want:     SHA256i(input3),
		wantDiff: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SHA256i(tt.args.in...)
			if tt.wantDiff {
				if !assert.NotEqualf(t, tt.want, got, "SHA256i(%v)", tt.args.in) {
					t.Errorf("SHA256i() = %v, do not want %v", got, tt.want)
				}
			} else {
				if !assert.Equalf(t, tt.want, got, "SHA256i(%v)", tt.args.in) {
					t.Errorf("SHA256i() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSHA256iOne(t *testing.T) {
	input := new(big.Int).SetBytes([]byte("abc"))
	input2 := new(big.Int).SetBytes([]byte("ab"))
	input3 := new(big.Int).SetBytes([]byte("cd"))
	type args struct {
		in *big.Int
	}
	tests := []struct {
		name     string
		args     args
		want     *big.Int
		wantDiff bool
	}{{
		name: "same inputs produce the same hash",
		args: args{input},
		want: SHA256iOne(input),
	}, {
		name:     "different inputs produce a differing hash",
		args:     args{input2},
		want:     SHA256iOne(input),
		wantDiff: true,
	}, {
		name:     "different inputs produce a differing hash: Hash(-a) != Hash(a)",
		args:     args{new(big.Int).Neg(input3)},
		want:     SHA256i(input3),
		wantDiff: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SHA256iOne(tt.args.in)
			if tt.wantDiff {
				if !assert.NotEqualf(t, tt.want, got, "SHA256iOne(%v)", tt.args.in) {
					t.Errorf("SHA256iOne() = %v, do not want %v", got, tt.want)
				}
			} else {
				if !assert.Equalf(t, tt.want, got, "SHA256iOne(%v)", tt.args.in) {
					t.Errorf("SHA256iOne() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
