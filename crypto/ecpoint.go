// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	big "github.com/binance-chain/tss-lib/common/int"

	"filippo.io/edwards25519"
	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto/ed25519"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/decred/dcrd/dcrec/edwards/v2"
)

// ECPoint convenience helper
type ECPoint struct {
	curve  elliptic.Curve
	coords [2]*big.Int
}

// Creates a new ECPoint and checks that the given coordinates are on the elliptic curve.
func NewECPoint(curve elliptic.Curve, X, Y *big.Int) (*ECPoint, error) {
	if !isOnCurve(curve, X, Y) {
		return nil, fmt.Errorf("NewECPoint: the given point is not on the elliptic curve")
	}
	return &ECPoint{curve, [2]*big.Int{X, Y}}, nil
}

// Creates a new ECPoint without checking that the coordinates are on the elliptic curve.
// Only use this function when you are completely sure that the point is already on the curve.
func NewECPointNoCurveCheck(curve elliptic.Curve, X, Y *big.Int) *ECPoint {
	return &ECPoint{curve, [2]*big.Int{X, Y}}
}

func (P *ECPoint) X() *big.Int {
	return new(big.Int).Set(P.coords[0])
}

func (P *ECPoint) Y() *big.Int {
	return new(big.Int).Set(P.coords[1])
}

func (P *ECPoint) Add(p1 *ECPoint) (*ECPoint, error) {
	x, y := P.curve.Add(P.X(), P.Y(), p1.X(), p1.Y())
	return NewECPoint(P.curve, big.Wrap(x), big.Wrap(y))
}

func (P *ECPoint) ScalarMult(k *big.Int) *ECPoint {
	return ScalarMult(P.curve, P, k)
}

// ScalarMultConstantTime uses the Montgomery Ladder Point Multiplication to compute R0 = k * P
// Implementation based on https://asecuritysite.com/golang/go_bitcoin
func (P *ECPoint) ScalarMultConstantTime(k *big.Int) *ECPoint {
	curve := P.curve
	k = big.NewInt(0).SetBytes(moduloReduce(k.Bytes(), curve.Params()))
	zero := big.NewInt(0)
	R0 := NewECPointNoCurveCheck(curve, zero, zero)
	R1 := NewECPointNoCurveCheck(curve, P.X(), P.Y())
	for i := P.curve.Params().N.BitLen() - 1; i >= 0; i-- {
		if k.Bit(i) == 0 {
			x, y := curve.Add(R0.X(), R0.Y(), R1.X(), R1.Y())
			R1 = NewECPointNoCurveCheck(curve, big.Wrap(x), big.Wrap(y))

			x, y = curve.Add(R0.X(), R0.Y(), R0.X(), R0.Y())
			R0 = NewECPointNoCurveCheck(curve, big.Wrap(x), big.Wrap(y))
		} else {
			x, y := curve.Add(R0.X(), R0.Y(), R1.X(), R1.Y())
			R0 = NewECPointNoCurveCheck(curve, big.Wrap(x), big.Wrap(y))

			x, y = curve.Add(R1.X(), R1.Y(), R1.X(), R1.Y())
			R1 = NewECPointNoCurveCheck(curve, big.Wrap(x), big.Wrap(y))
		}
	}
	return R0
}
func moduloReduce(k []byte, curveParams *elliptic.CurveParams) []byte {
	// Since the order of G is curve.N, we can use a much smaller number by
	// doing modulo curve.N
	if len(k) > (curveParams.BitSize / 8) {
		tmpK := new(big.Int).SetBytes(k)
		tmpK.Mod(tmpK, big.Wrap(curveParams.N))
		return tmpK.Bytes()
	}

	return k
}

func (P *ECPoint) ToBtcecPubKey() *btcec.PublicKey {
	var x, y btcec.FieldVal
	x.SetByteSlice(P.X().Bytes())
	y.SetByteSlice(P.Y().Bytes())
	return btcec.NewPublicKey(&x, &y)
}

func (P *ECPoint) ToEdwardsPubKey() *edwards.PublicKey {
	ecdsaPK := ecdsa.PublicKey{
		Curve: P.curve,
		X:     P.X(),
		Y:     P.Y(),
	}
	pk := edwards.PublicKey(ecdsaPK)
	return &pk
}

func (P *ECPoint) IsOnCurve() bool {
	return isOnCurve(P.curve, P.coords[0], P.coords[1])
}

func (P *ECPoint) Curve() elliptic.Curve {
	return P.curve
}

func (P *ECPoint) Equals(p2 *ECPoint) bool {
	if P == nil || p2 == nil {
		return false
	}
	return P.X().Cmp(p2.X()) == 0 && P.Y().Cmp(p2.Y()) == 0
}

func (P *ECPoint) SetCurve(curve elliptic.Curve) *ECPoint {
	P.curve = curve
	return P
}

func (P *ECPoint) ValidateBasic() bool {
	return P != nil && P.coords[0] != nil && P.coords[1] != nil && P.IsOnCurve()
}

/* func (p *ECPoint) EightInvEight() *ECPoint {
	return p.ScalarMult(eight).ScalarMult(eightInv)
}
*/

func ScalarBaseMult(curve elliptic.Curve, k *big.Int) *ECPoint {

	G := NewECPointNoCurveCheck(curve, big.Wrap(curve.Params().Gx), big.Wrap(curve.Params().Gy))
	return ScalarMult(curve, G, k)
}

func ScalarMult(curve elliptic.Curve, P *ECPoint, k *big.Int) *ECPoint {
	// use the curve's function only if it is a constant-time implementation
	if specificCurve, match := matchesSpecificCurve(curve.Params(), elliptic.P224(), elliptic.P521()); match {
		x, y := specificCurve.ScalarBaseMult(k.Bytes())
		p, err := NewECPoint(curve, big.Wrap(x), big.Wrap(y)) // it must be on the curve, no need to check.
		if err != nil {
			common.Logger.Errorf("error %v", err)
		}
		return p
	}
	if _, isEdwards := curve.(*edwards.TwistedEdwardsCurve); isEdwards {
		Pʹ := ed25519.Fromxy(P.X(), P.Y())
		if curve.Params().N.Cmp(k) == -1 {
			common.Logger.Warn("warn")
		}
		kBytes := ed25519.BigIntToEncodedBytes(k)
		scalar := edwards25519.NewScalar()
		_, err := scalar.SetCanonicalBytes(kBytes[:])
		if err != nil {
			common.Logger.Errorf("error %v", err)
		}
		x, y := ed25519.Toxy(Pʹ.ScalarMult(scalar, Pʹ))
		Q, err := NewECPoint(curve, x, y)
		if err != nil {
			common.Logger.Errorf("error %v", err)
		}
		return Q
	}
	return P.ScalarMultConstantTime(k)
}

func matchesSpecificCurve(params *elliptic.CurveParams, available ...elliptic.Curve) (elliptic.Curve, bool) {
	for _, c := range available {
		if params == c.Params() {
			return c, true
		}
	}
	return nil, false
}

func isOnCurve(c elliptic.Curve, x, y *big.Int) bool {
	if x == nil || y == nil {
		return false
	}
	return c.IsOnCurve(x, y)
}

// ----- //

func FlattenECPoints(in []*ECPoint) ([]*big.Int, error) {
	if in == nil {
		return nil, errors.New("FlattenECPoints encountered a nil in slice")
	}
	flat := make([]*big.Int, 0, len(in)*2)
	for _, point := range in {
		if point == nil || point.coords[0] == nil || point.coords[1] == nil {
			return nil, errors.New("FlattenECPoints found nil point/coordinate")
		}
		flat = append(flat, point.coords[0])
		flat = append(flat, point.coords[1])
	}
	return flat, nil
}

func UnFlattenECPoints(curve elliptic.Curve, in []*big.Int, noCurveCheck ...bool) ([]*ECPoint, error) {
	if in == nil || len(in)%2 != 0 {
		return nil, errors.New("UnFlattenECPoints expected an in len divisible by 2")
	}
	var err error
	unFlat := make([]*ECPoint, len(in)/2)
	for i, j := 0, 0; i < len(in); i, j = i+2, j+1 {
		if len(noCurveCheck) == 0 || !noCurveCheck[0] {
			unFlat[j], err = NewECPoint(curve, in[i], in[i+1])
			if err != nil {
				return nil, err
			}
		} else {
			unFlat[j] = NewECPointNoCurveCheck(curve, in[i], in[i+1])
		}
	}
	for _, point := range unFlat {
		if point.coords[0] == nil || point.coords[1] == nil {
			return nil, errors.New("UnFlattenECPoints found nil coordinate after unpack")
		}
	}
	return unFlat, nil
}

// ----- //
// Gob helpers for if you choose to encode messages with Gob.

func (P *ECPoint) GobEncode() ([]byte, error) {
	buf := &bytes.Buffer{}
	x, err := P.coords[0].GobEncode()
	if err != nil {
		return nil, err
	}
	y, err := P.coords[1].GobEncode()
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, uint32(len(x)))
	if err != nil {
		return nil, err
	}
	buf.Write(x)
	err = binary.Write(buf, binary.LittleEndian, uint32(len(y)))
	if err != nil {
		return nil, err
	}
	buf.Write(y)

	return buf.Bytes(), nil
}

func (P *ECPoint) GobDecode(buf []byte) error {
	reader := bytes.NewReader(buf)
	var length uint32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return err
	}
	x := make([]byte, length)
	n, err := reader.Read(x)
	if n != int(length) || err != nil {
		return fmt.Errorf("gob decode failed: %v", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return err
	}
	y := make([]byte, length)
	n, err = reader.Read(y)
	if n != int(length) || err != nil {
		return fmt.Errorf("gob decode failed: %v", err)
	}

	X := new(big.Int)
	if err := X.GobDecode(x); err != nil {
		return err
	}
	Y := new(big.Int)
	if err := Y.GobDecode(y); err != nil {
		return err
	}
	P.curve = tss.EC()
	P.coords = [2]*big.Int{X, Y}
	if !P.IsOnCurve() {
		return errors.New("ECPoint.UnmarshalJSON: the point is not on the elliptic curve")
	}
	return nil
}

// ----- //
func (P *ECPoint) Bytes() [2][]byte {
	return [...][]byte{
		P.X().Bytes(),
		P.Y().Bytes(),
	}
}

func NewECPointFromBytes(ec elliptic.Curve, bzs [][]byte) (*ECPoint, error) {
	if !common.NonEmptyMultiBytes(bzs, 2) {
		return nil, fmt.Errorf("NewECPointFromBytes expects 2 points in bzs, got %d", len(bzs))
	}
	point, err := NewECPoint(ec,
		new(big.Int).SetBytes(bzs[0]),
		new(big.Int).SetBytes(bzs[1]))
	if err != nil {
		return nil, err
	}
	return point, nil
}

// crypto.ECPoint is not inherently json marshal-able
func (P *ECPoint) MarshalJSON() ([]byte, error) {
	ecName, ok := tss.GetCurveName(P.curve)
	if !ok {
		return nil, fmt.Errorf("cannot find %T name in curve registry, please call tss.RegisterCurve(name, curve) to register it first", P.curve)
	}

	return json.Marshal(&struct {
		Curve  string
		Coords [2]*big.Int
	}{
		Curve:  string(ecName),
		Coords: P.coords,
	})
}

func (P *ECPoint) UnmarshalJSON(payload []byte) error {
	aux := &struct {
		Curve  string
		Coords [2]*big.Int
	}{}
	if err := json.Unmarshal(payload, &aux); err != nil {
		return err
	}
	P.coords = [2]*big.Int{aux.Coords[0], aux.Coords[1]}

	if len(aux.Curve) > 0 {
		ec, ok := tss.GetCurveByName(tss.CurveName(aux.Curve))
		if !ok {
			return fmt.Errorf("cannot find curve named with %s in curve registry, please call tss.RegisterCurve(name, curve) to register it first", aux.Curve)
		}
		P.curve = ec
	} else {
		// forward compatible, use global ec as default value
		P.curve = tss.EC()
	}

	if !P.IsOnCurve() {
		return fmt.Errorf("ECPoint.UnmarshalJSON: the point is not on the elliptic curve (%T) ", P.curve)
	}

	return nil
}
