package ed25519

import (
	"crypto/elliptic"
	mathbig "math/big"
	"math/bits"

	"filippo.io/edwards25519"
	"filippo.io/edwards25519/field"
	edwards255192 "github.com/agl/ed25519/edwards25519"
	"github.com/binance-chain/tss-lib/common"
	big "github.com/binance-chain/tss-lib/common/int"
)

// Reverse reverses a byte string.
func Reverse(s *[32]byte) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func CopyBytes(aB []byte) *[32]byte {
	if aB == nil {
		return nil
	}
	s := new([32]byte)

	// If we have a short byte string, expand
	// it so that it's long enough.
	aBLen := len(aB)
	if aBLen < 32 {
		diff := 32 - aBLen
		for i := 0; i < diff; i++ {
			aB = append([]byte{0x00}, aB...)
		}
	}

	for i := 0; i < 32; i++ {
		s[i] = aB[i]
	}

	return s
}

func BigIntToEncodedBytes(a *big.Int) *[32]byte {
	s := new([32]byte)
	if a == nil {
		return s
	}

	// Caveat: a can be longer than 32 bytes.
	s = CopyBytes(a.Bytes())

	// Reverse the byte string --> little endian after
	// encoding.
	Reverse(s)

	return s
}

// BigIntToFieldElement converts a big endian integer into its corresponding
// 40 byte field representation.
func BigIntToFieldElement(a *big.Int) *field.Element {
	return fromBig(new(field.Element), a)
}

// fromBig sets v = n, and returns v. The bit length of n must not exceed 256.
func fromBig(v *field.Element, n *big.Int) *field.Element {
	if n.BitLen() > 32*8 {
		panic("edwards25519: invalid field element input size")
	}

	buf := make([]byte, 0, 32)
	for _, word := range n.Bits() {
		for i := 0; i < bits.UintSize; i += 8 {
			if len(buf) >= cap(buf) {
				break
			}
			buf = append(buf, byte(word))
			word >>= 8
		}
	}

	v.SetBytes(buf[:32])
	return v
}

func Toxy(P *edwards25519.Point) (xt, yt *big.Int) {
	var zInv, x, y field.Element
	X, Y, Z, _ := P.ExtendedCoordinates()
	zInv.Invert(Z)       // zInv = 1 / Z
	x.Multiply(X, &zInv) // x = X / Z
	y.Multiply(Y, &zInv) // y = Y / Z
	xt = big.Wrap(elementToBigInt(&x))
	yt = big.Wrap(elementToBigInt(&y))
	return
}

func Fromxy(xt, yt *big.Int) (P *edwards25519.Point) {
	X := BigIntToFieldElement(xt)
	Y := BigIntToFieldElement(yt)
	Z := BigIntToFieldElement(big.NewInt(1))
	T := BigIntToFieldElement(big.NewInt(1))
	T.Multiply(X, Y)
	P = edwards25519.NewIdentityPoint()
	P.SetExtendedCoordinates(X, Y, Z, T)
	return
}

func toLittleEndian(a *big.Int) []byte {
	b := a.Bytes()
	for i := 0; i < len(b)/2; i++ {
		b[i], b[len(b)-i-1] = b[len(b)-i-1], b[i]
	}
	return b
}

// elementToBigInt returns v as a big.Int.
func elementToBigInt(v *field.Element) *mathbig.Int {
	buf := v.Bytes()

	words := make([]mathbig.Word, 32*8/bits.UintSize)
	for n := range words {
		for i := 0; i < bits.UintSize; i += 8 {
			if len(buf) == 0 {
				break
			}
			words[n] |= mathbig.Word(buf[0]) << mathbig.Word(i)
			buf = buf[1:]
		}
	}

	return new(mathbig.Int).SetBits(words)
}

func ECPointToExtendedElement(ec elliptic.Curve, x *big.Int, y *big.Int) edwards255192.ExtendedGroupElement {
	encodedXBytes := BigIntToEncodedBytes(x)
	encodedYBytes := BigIntToEncodedBytes(y)

	z := common.GetRandomPositiveInt(big.Wrap(ec.Params().N))
	encodedZBytes := BigIntToEncodedBytes(z)

	var fx, fy, fxy edwards255192.FieldElement
	edwards255192.FeFromBytes(&fx, encodedXBytes)
	edwards255192.FeFromBytes(&fy, encodedYBytes)

	var X, Y, Z, T edwards255192.FieldElement
	edwards255192.FeFromBytes(&Z, encodedZBytes)

	edwards255192.FeMul(&X, &fx, &Z)
	edwards255192.FeMul(&Y, &fy, &Z)
	edwards255192.FeMul(&fxy, &fx, &fy)
	edwards255192.FeMul(&T, &fxy, &Z)

	return edwards255192.ExtendedGroupElement{
		X: X,
		Y: Y,
		Z: Z,
		T: T,
	}
}

func ECPointToEncodedBytes(x *big.Int, y *big.Int) *[32]byte {
	s := BigIntToEncodedBytes(y)
	xB := BigIntToEncodedBytes(x)
	xFE := new(edwards255192.FieldElement)
	edwards255192.FeFromBytes(xFE, xB)
	isNegative := edwards255192.FeIsNegative(xFE) == 1

	if isNegative {
		s[31] |= 1 << 7
	} else {
		s[31] &^= 1 << 7
	}
	return s
}

func AddExtendedElements(p, q edwards255192.ExtendedGroupElement) edwards255192.ExtendedGroupElement {
	var r edwards255192.CompletedGroupElement
	var qCached edwards255192.CachedGroupElement
	q.ToCached(&qCached)
	edwards255192.GeAdd(&r, &p, &qCached)
	var result edwards255192.ExtendedGroupElement
	r.ToExtended(&result)
	return result
}
