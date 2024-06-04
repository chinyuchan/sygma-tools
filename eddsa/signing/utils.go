// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package signing

import (
	"fmt"

	big "github.com/binance-chain/tss-lib/common/int"

	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/ed25519"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func encodedBytesToBigInt(s *[32]byte) *big.Int {
	// Use a copy so we don't screw up our original
	// memory.
	sCopy := new([32]byte)
	for i := 0; i < 32; i++ {
		sCopy[i] = s[i]
	}
	ed25519.Reverse(sCopy)

	bi := new(big.Int).SetBytes(sCopy[:])

	return bi
}

func OddY(a *crypto.ECPoint) bool {
	return a.Y().Bit(0) > 0
}

func SchnorrVerify(p *btcec.PublicKey, m []byte, r *big.Int, s *big.Int) error {
	var R btcec.FieldVal
	R.SetByteSlice(r.Bytes())
	var S btcec.ModNScalar
	S.SetByteSlice(s.Bytes())
	return schnorrVerify(m, p, R, S)
}

// /////////////////////////

// signatureError creates an Error given a set of arguments.
func signatureError(kind schnorr.ErrorKind, desc string) schnorr.Error {
	return schnorr.Error{Err: kind, Description: desc}
}

// from https://github.com/Roasbeef/btcd/blob/5a59e7c0ddfb46d1bd7a99b87dbb8f7657a14382/btcec/schnorr/signature.go
// for whatever reason using this code directly yields some issues
func schnorrVerify(hash []byte, pubKey *btcec.PublicKey, r btcec.FieldVal, s btcec.ModNScalar) error {
	// The algorithm for producing a BIP-340 signature is described in
	// README.md and is reproduced here for reference:
	//
	// 1. Fail if m is not 32 bytes
	// 2. P = lift_x(int(pk)).
	// 3. r = int(sig[0:32]); fail is r >= p.
	// 4. s = int(sig[32:64]); fail if s >= n.
	// 5. e = int(tagged_hash("BIP0340/challenge", bytes(r) || bytes(P) || M)) mod n.
	// 6. R = s*G - e*P
	// 7. Fail if is_infinite(R)
	// 8. Fail if not hash_even_y(R)
	// 9. Fail is x(R) != r.
	// 10. Return success iff not failure occured before reachign this
	// point.

	// Step 1.
	//
	// Fail if m is not 32 bytes
	if len(hash) != 32 {
		str := fmt.Sprintf("wrong size for message (got %v, want %v)", len(hash), 32)
		return signatureError("ErrInvalidHashLen", str)
	}

	// Step 2.
	//
	// P = lift_x(int(pk))
	//
	// Fail if P is not a point on the curve
	if !pubKey.IsOnCurve() {
		str := "pubkey point is not on curve"
		return signatureError("ErrPubKeyNotOnCurve", str)
	}

	// Step 3.
	//
	// Fail if r >= p
	//
	// Note this is already handled by the fact r is a field element.

	// Step 4.
	//
	// Fail if s >= n
	//
	// Note this is already handled by the fact s is a mod n scalar.

	// Step 5.
	//
	// e = int(tagged_hash("BIP0340/challenge", bytes(r) || bytes(P) || M)) mod n.
	var rBytes [32]byte
	r.PutBytesUnchecked(rBytes[:])
	pBytes := schnorr.SerializePubKey(pubKey)

	commitment := chainhash.TaggedHash(
		[]byte("BIP0340/challenge"), rBytes[:], pBytes, hash,
	)

	var e btcec.ModNScalar
	if overflow := e.SetBytes((*[32]byte)(commitment)); overflow != 0 {
		str := "hash of (r || P || m) too big"
		return signatureError("ErrSchnorrHashValue", str)
	}

	// Negate e here so we can use AddNonConst below to subtract the s*G
	// point from e*P.
	e.Negate()

	// Step 6.
	//
	// R = s*G - e*P
	var P, R, sG, eP btcec.JacobianPoint
	pubKey.AsJacobian(&P)
	btcec.ScalarBaseMultNonConst(&s, &sG)
	btcec.ScalarMultNonConst(&e, &P, &eP)
	btcec.AddNonConst(&sG, &eP, &R)

	// Step 7.
	//
	// Fail if R is the point at infinity
	if (R.X.IsZero() && R.Y.IsZero()) || R.Z.IsZero() {
		str := "calculated R point is the point at infinity"
		return signatureError("ErrSigRNotOnCurve", str)
	}

	// Step 8.
	//
	// Fail if R.y is odd
	//
	// Note that R must be in affine coordinates for this check.
	R.ToAffine()
	if R.Y.IsOdd() {
		str := "calculated R y-value is odd"
		return signatureError("ErrSigRYIsOdd", str)
	}

	// Step 9.
	//
	// Verified if R.x == r
	//
	// Note that R must be in affine coordinates for this check.
	if !r.Equals(&R.X) {
		str := "calculated R point was not given R"
		return signatureError("ErrUnequalRValues", str)
	}

	// Step 10.
	//
	// Return success iff not failure occured before reachign this
	return nil
}
