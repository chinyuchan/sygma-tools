// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package zkpdec

import (
	"testing"
	"time"

	big "github.com/binance-chain/tss-lib/common/int"
	"github.com/ipfs/go-log"

	int2 "github.com/binance-chain/tss-lib/common/int"
	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/tss"
)

// Using a modulus length of 2048 is recommended in the GG18 spec
const (
	testSafePrimeBits = 1024
)

func TestDec(test *testing.T) {
	ec := tss.EC()
	q := big.Wrap(ec.Params().N)

	primes := [2]*big.Int{common.GetRandomPrimeInt(testSafePrimeBits), common.GetRandomPrimeInt(testSafePrimeBits)}
	NCap, s, t, err := crypto.GenerateNTildei(primes)
	assert.NoError(test, err)

	sk, pk, err := paillier.GenerateKeyPair(testSafePrimeBits*2, time.Minute*15)
	assert.NoError(test, err)

	x := common.GetRandomPositiveInt(q)
	y := new(big.Int).Add(x, q)
	C, rho, err := sk.EncryptAndReturnRandomness(y)
	assert.NoError(test, err)

	proof, err := NewProof(ec, pk, C, x, NCap, s, t, y, rho)
	assert.NoError(test, err)

	ok := proof.Verify(ec, pk, C, x, NCap, s, t)
	assert.True(test, ok, "proof must verify")
}

func TestDecWithNonce(test *testing.T) {
	ec := tss.EC()
	q := big.Wrap(ec.Params().N)

	primes := [2]*big.Int{common.GetRandomPrimeInt(testSafePrimeBits), common.GetRandomPrimeInt(testSafePrimeBits)}
	NCap, s, t, err := crypto.GenerateNTildei(primes)
	assert.NoError(test, err)

	sk, pk, err := paillier.GenerateKeyPair(testSafePrimeBits*2, time.Minute*15)
	assert.NoError(test, err)

	x := common.GetRandomPositiveInt(q)
	y := new(big.Int).Add(x, q)
	C, rho, err := sk.EncryptAndReturnRandomness(y)
	assert.NoError(test, err)
	nonce := common.GetBigRandomPositiveInt(q, q.BitLen())

	proof, err := NewProofGivenNonce(ec, pk, C, x, NCap, s, t, y, rho, nonce)
	assert.NoError(test, err)

	ok := proof.VerifyWithNonce(ec, pk, C, x, NCap, s, t, nonce)
	assert.True(test, ok, "proof must verify")
}

func TestDecWithCompositions(test *testing.T) {
	if err := log.SetLogLevel("tss-lib", "debug"); err != nil {
		panic(err)
	}
	ec := tss.EC()
	q := big.Wrap(ec.Params().N)
	q3 := new(big.Int).Mul(q, new(big.Int).Mul(q, q))
	modQ3 := int2.ModInt(q3)
	modN := int2.ModInt(q)
	zero := big.NewInt(0)

	primes := [2]*big.Int{common.GetRandomPrimeInt(testSafePrimeBits), common.GetRandomPrimeInt(testSafePrimeBits)}
	NCap, s, t, err := crypto.GenerateNTildei(primes)
	assert.NoError(test, err)

	_, pk, err := paillier.GenerateKeyPair(testSafePrimeBits*2, time.Minute*10)
	assert.NoError(test, err)
	N2 := pk.NSquare()

	// Ki = enc(ki,𝜌i)
	𝛾i := common.GetRandomPositiveInt(q)
	ki := common.GetRandomPositiveInt(q)
	Ki, 𝜌i, err := pk.EncryptAndReturnRandomness(ki)

	/*
		common.Logger.Debugf("dec step 7 - pk.N: %v, 𝛾i:%v, ki: %v, Ki:%v, 𝜌i: %v",
			common.FormatBigInt(pk.N),
			common.FormatBigInt(𝛾i), common.FormatBigInt(ki),
			common.FormatBigInt(Ki), common.FormatBigInt(𝜌i))
	*/

	proof1, err := NewProof(ec, pk, Ki, modN.Add(zero, ki), NCap, s, t, ki, 𝜌i)
	assert.NoError(test, err)
	ok1 := proof1.Verify(ec, pk, Ki, modN.Add(zero, ki), NCap, s, t)
	assert.True(test, ok1, "proof must verify")

	// 𝛾K = (𝛾i ⊗ Ki)
	𝛾K, err := pk.HomoMult(𝛾i, Ki)
	𝜌ʹ := big.NewInt(1).Exp(𝜌i, 𝛾i, N2)
	yʹ := modQ3.Mul(𝛾i, ki)
	proof2, err := NewProof(ec, pk, 𝛾K, modN.Add(zero, yʹ), NCap, s, t, yʹ, 𝜌ʹ)
	assert.NoError(test, err)
	ok2 := proof2.Verify(ec, pk, 𝛾K, modN.Add(zero, yʹ), NCap, s, t)
	assert.True(test, ok2, "proof must verify")

	// Di = (𝛾i ⊗ Ki) ⊕ enc(-𝛽,si)
	x := common.GetRandomPositiveInt(q)
	𝛽ʹ := new(big.Int).Add(x, q)
	T, si, err := pk.EncryptAndReturnRandomness(𝛽ʹ)
	assert.NoError(test, err)
	Di, err := pk.HomoAdd(𝛾K, T)

	𝜌ʺ := N2.Mul(big.NewInt(1).Exp(𝜌i, 𝛾i, N2), si)
	yʺ := modQ3.Add(𝛽ʹ, modQ3.Mul(𝛾i, ki))
	proof3, err := NewProof(ec, pk, Di, modN.Add(zero, yʺ), NCap, s, t, yʺ, 𝜌ʺ)
	assert.NoError(test, err)
	/* common.Logger.Infof("dec step 11 - pk.N: %v, Di:%v, modN yʺ: %v, NCap:%v, s:%v, t:%v, yʺ: %v, 𝜌ʺ:%v, proof: %v",
	common.FormatBigInt(pk.N),
	common.FormatBigInt(Di), common.FormatBigInt(modN.Add(zero, yʺ)),
	common.FormatBigInt(NCap), common.FormatBigInt(s), common.FormatBigInt(t),
	common.FormatBigInt(yʺ), common.FormatBigInt(𝜌ʺ), FormatProofDec(proof3))
	*/

	ok3 := proof3.Verify(ec, pk, Di, modN.Add(zero, yʺ), NCap, s, t)
	assert.True(test, ok3, "proof must verify")

}

func TestDecWithCompositionsConstants(test *testing.T) {
	if err := log.SetLogLevel("tss-lib", "debug"); err != nil {
		panic(err)
	}
	ec := tss.EC()
	q := big.Wrap(ec.Params().N)
	q3 := new(big.Int).Mul(q, new(big.Int).Mul(q, q))
	modQ3 := int2.ModInt(q3)
	modN := int2.ModInt(q)
	zero := big.NewInt(0)

	NCap, _ := new(big.Int).SetString("bc088106a7ba492048b39aadb185606379450125aeca24ebc19695760278a96338b31aa31d958d4e927363500d361b35066bc1bba9e950679d499ed94da9e5adcef35c062ed918e7492a895d2843b4eb8f08f30ff5c33db3ce54bd6e3157999336adf8ae192483c1d690ded319393e9077505e567f7aac564e037d7cc3aba0b4e313e9be68dfaaedfc30f30325d8c8e970433f89b30bec257d789388c73a15f9d520e9e238dbde5d7a83e1fc63530307a700ef8a13d1e86e92ca61c9af219281af0412624404c8f8a5e18e990724e6088bf133d8dfdb69fcf4bb6ed327fd25ea97c92492e430889a059162e6f68296cc65f548457e7da2d91d8f2a6cb9d67535", 16)
	s, _ := new(big.Int).SetString("7510b53324881a84a169a2d3930240d81740d20767199520287c745190b261fea596de73dad5e39305c7a60df539eba4b4a3d62277d0dfd70a42e69ed588b86e2ed89789af01fb19e0cf9d5ac7ff285c1eea4e3bd5ac58961b234faad1e9b4960cc4e935783686c507bfe780736cf8815139c839e113bfaed329f954e521a2c95991629dfdd71b95868e0ef42cc541c22a83adf544dd2e5ce7ed287f1ec959baf6ac98ec6bc318da3c37369ceea0e9c7e7a3636477cce568bfc22bdbfef9876f2483dfbeea2863b7ad0d37032f1360f5e27d4be746634c47f122560fa20d23d2d941be2b33817ea2a809dcaa920ad83e7ea1eb2176584db0ed59c4ce89a6f46a", 16)
	t, _ := new(big.Int).SetString("b27362b478320b1e37f39fa194bc25c591fdafcbbf7b4083214655f993e442bf54ae3eda90f979a09a39cf88f82d8e8f661ad18581e923eaeabc5ce4119ef5e957075bed7032707205ca64c31874571d8e1e519e97baaa809a51eb09c226232f854409f49a889c2ada43862db4868d44ec3e2a399afb628ba3981c97464305a840581ad258fc7a7ab9858a890d99409d218700ece7a019bb8f4679121abbede0338fc7b298d0ec5de863016fc570bbffd547f025706e9a71d62f8f86e68ac0ba6388585aebd107f1ccc2d9c58a305bab9329204a66b70b72bd8abc5cc493a7bb24512ad712eead0565d3f7583e50221237bdcb6dd1625bc74ba37a9948207999", 16)

	alpha, _ := new(big.Int).SetString("9bcd038d3da3f274e62030f164d5807d29e98a173c4f0475eb2f227f9bfaad1a84b6ceeb8eb121d0a817456b7598731db2f30fd06b894b93db2f47b396b158b8e5b260df94a0eaf5b4f7c022fde58d12a4639cb1b0eabfd9e44b3e1ecef1589a", 16)
	mu, _ := new(big.Int).SetString("509d421a6bf4f83e96971fe9867c41032761545eba4c6b4154661b8652872890d32b671f2e1ca20d5c3a3c424d16baeb74ec2969b9f9e422024fe88cc74bed163905244c89f2bb0e99c66a43c37fe3326b869f072968bf192559582746642cfb585e01a40e94fc842f63eb50f7d93bbc91f76af26d39fe13838b1075ea7e10831f71047db9716eeacf307ad6c73dcaa2935640d385025bdc0d8bbb668d294c7b074d909f80394bbe6046e3b78663562764bf85722cbed329362c911e359b6f1bfed4ce2bb75bf0634d3d3630a8b13f2b30931c85a45b2523185e7fc9f8565e51d3bb1ada2c9e31f19d8969da971433a1d8da58e7d74fb366bb877259505b18cda43019e0898b3002d1b2c7732f5a407971558ed0542e702fdb5def616308c5e3", 16)
	v, _ := new(big.Int).SetString("3bac3a63c189361ff8605780c49bb4ae638d9b94331362e8d208b6c1f4f32b9bca06029c53b1f0afac1c3fbf6d4442c1e15d84829bd50ee6cf188d10881824ba0ab8b635b45acef7f126d976e79576cb27c775d7a0cce3afb4a779e1b1341e4216ff935014dbefd18c92a027eac991ba21ade2442c1b9eb8e86f4ea5840ceca749203eb30bf0ff3c62d40da2fee0a09bfdcddb244fb787c5b9fbf5d2a4e22251640cac8851b883d5f27a923121fa64e3f6720f35aca43ad8f9ca60570908be157b4e1f888f91774c7fee6eb3044708f8807e6e3832fb0454de856a7d9fe2a1f2753563c7fec94d6b19d6544b3f0a3f668d297d4ede1aa4b39204294c0e37d9cd015c094be5601a389b5ce5b46168774e9c64fb443d5b541ca4b4aa940841cc1b3a697c3e47e78231718cc7d3925c35104c3f799603f7a250465a02868762b9afb52b3b540d79440903dd146dee9632f2e00e281d57809316b7fd9846aea494ea", 16)
	r, _ := new(big.Int).SetString("95b4498d23dbd3005eaf313789387a5c15f4a867dbbc7d57eb7872b060de4ffd0c7f4abb7c6f41bdff7eba3ea56f17c95bef71c0f38f1099c33d9f444882a8b26ad8a1127cb11dc7a815d92f8f89f4ed806d003a4894dd4ff4b5baf36069bb31d6c27e9567c37faf19f42a08410457da24012d2f2d2745e9761b1ce32e1965aedaad6e51bffcd41521868b8bf81e68ba26bd2ea8b5f89b37bc5f947d641a8c9dfed548276900aca15c0547bb2276c0f7c920c3332cbb847bfc6e6bf907c320109dd07a9bffbdfc9aeac883856d299e98b3cb995195ff4ae29c8fd36ec89de0ddbec0057c8c2debc34b61eecf93e2bb31f37f11d6c68e3c450020f05240fcf2e", 16)

	pkN, _ := new(big.Int).SetString("c75cab8e0d3b2eb7dac8a787b59bbb0ba83983a310b88e41ead2634128a6b4a1065d5d2806ee1aad06fe5a7b4e4f46900dc4f58418160525e6d07488281e167f7b1310099e4e00a4e9b894e5e9124be507d81a5c6490b5515c5f0384f2e38e123b4475fd57d5f71de472823f1cea72d64faeaeb7124b699302b5ae30eb04df4f6bd6a6b0ad68f8d10c74c20fcfa291db96fbbeabb464e4f8a2f21c9d24f56f2888cadffcef6ec77e060426524e87921135144ac88b55a363f6c06a14a803219cf3667ba37dc0c0ef2f4216501cdec9a7837eb5d8e2bb6339b14b3ee207798b616d9b5f3b030936e7c134b100c6315335ee1ef4091eb430e6cf654010aeb500c9", 16)
	pk := &paillier.PublicKey{N: pkN}
	N2 := pk.NSquare()

	// Ki = enc(ki,𝜌i)
	𝛾i, _ := new(big.Int).SetString("c7070cf90b032eb3c844411c918f51653659ea85d007e42fe9a2d29b2ac5a9c", 16)
	ki, _ := new(big.Int).SetString("64237ca0996e32e32523c31ad653d61744fd41042bba2e6e3981a848f7ecd6f5", 16)
	𝜌i, _ := new(big.Int).SetString("4d6760cf3d6f434435735b908addf65184d02fb2059bfb690495866a6fc1aaeec211acc415264a2a94528ca4315c5abbe2cd4baa27117bcb55c11a205b3b8cfa3fbc8702e6aaa06490925a73a1ff328722a8dd1fa0df7fe459a227c43eaa40b377a33c9c5d1a41e58bbdb99b52aa78b54e14efaf02e5b0c3cd49211f4e1233fa9691e60af86df3050ec0a0b8dbe29a1b770a3fbf7753aa303d90901659e6d5b45911aedc0090f8723055039feeefa7e68bfe7c6f317f8d338f7ec8802db6ed3c002236000a81b269bc4a45391584572e446b81e3d28e3162ea76f9860d4821ade46b953a4908613092f08dfa1b7f9627e014286adb529a9f2e5cc6a265005ca8", 16)
	Ki, err := pk.EncryptWithGivenRandomness(ki, 𝜌i)

	/* common.Logger.Debugf("dec step 13 - pk.N: %v, 𝛾i:%v, ki: %v, Ki:%v, 𝜌i: %v",
	common.FormatBigInt(pk.N),
	common.FormatBigInt(𝛾i), common.FormatBigInt(ki),
	common.FormatBigInt(Ki), common.FormatBigInt(𝜌i))
	*/

	proof1, err := NewProofGivenAux(ec, pk, Ki, modN.Add(zero, ki), NCap, s, t, ki, 𝜌i, alpha, mu, v, r)
	assert.NoError(test, err)
	ok1 := proof1.Verify(ec, pk, Ki, modN.Add(zero, ki), NCap, s, t)
	assert.True(test, ok1, "proof must verify")

	// 𝛾K = (𝛾i ⊗ Ki)
	𝛾K, err := pk.HomoMult(𝛾i, Ki)
	𝜌ʹ := big.NewInt(1).Exp(𝜌i, 𝛾i, N2)
	yʹ := modQ3.Mul(𝛾i, ki)
	proof2, err := NewProofGivenAux(ec, pk, 𝛾K, modN.Add(zero, yʹ), NCap, s, t, yʹ, 𝜌ʹ, alpha, mu, v, r)
	assert.NoError(test, err)
	ok2 := proof2.Verify(ec, pk, 𝛾K, modN.Add(zero, yʹ), NCap, s, t)
	assert.True(test, ok2, "proof must verify")

	// Di = (𝛾i ⊗ Ki) ⊕ enc(-𝛽,si)
	x, _ := new(big.Int).SetString("64237ca0996e32e32523c31ad653d61744fd41042bba2e6e3981a848f7ecd6f5", 16)
	𝛽ʹ := new(big.Int).Add(x, q)
	si, _ := new(big.Int).SetString("b669aad2e61e4dacc0daf3f8aa640e565155cc87a6713a1cf30c363a7b20b0785e23a828f08454bc41f8e776375da8bfbb7d332293843ac8927e13892421a3f2f60a174007c3185378c4a8084a43fd9daf9d26cb8e65f4ef3bb747956f7e55918612d7d00a280df555f41f5694a5698a1c109616112589dd648f425fd84075040af45c71f1cb6331ae7f70fc12bed7d8dfeed2724b9f00a47f195c7f7508c89001ed2ce9658019802a69547066022da0cf3ae338de594cbab45024413a5f261f815a7e29f14a9467c690bd9588efcf62fcd7251bef3fdfa2d8a26c8a99d080fef010f8278e84856b6c4c4ef6655ea09969ab26a7f823eb858827309cef88c800", 16)
	T, err := pk.EncryptWithGivenRandomness(𝛽ʹ, si)
	assert.NoError(test, err)
	Di, err := pk.HomoAdd(𝛾K, T)

	𝜌ʺ := N2.Mul(big.NewInt(1).Exp(𝜌i, 𝛾i, N2), si)
	yʺ := modQ3.Add(𝛽ʹ, modQ3.Mul(𝛾i, ki))

	proof3, err := NewProofGivenAux(ec, pk, Di, modN.Add(zero, yʺ), NCap, s, t, yʺ, 𝜌ʺ, alpha, mu, v, r)
	assert.NoError(test, err)
	/*
		common.Logger.Debugf("dec step 17 - pk.N: %v, Di:%v, modN yʺ: %v, NCap:%v, s:%v, t:%v, yʺ: %v, 𝜌ʺ:%v, proof: %v",
			common.FormatBigInt(pk.N),
			common.FormatBigInt(Di), common.FormatBigInt(modN.Add(zero, yʺ)),
			common.FormatBigInt(NCap), common.FormatBigInt(s), common.FormatBigInt(t),
			common.FormatBigInt(yʺ), common.FormatBigInt(𝜌ʺ), FormatProofDec(proof3))
	*/

	ok3 := proof3.Verify(ec, pk, Di, modN.Add(zero, yʺ), NCap, s, t)
	assert.True(test, ok3, "proof must verify")

}

func TestSmallMod(test *testing.T) {
	if err := log.SetLogLevel("tss-lib", "debug"); err != nil {
		panic(err)
	}
	ec := tss.EC()
	q := big.Wrap(ec.Params().N)
	q3 := new(big.Int).Mul(q, new(big.Int).Mul(q, q))
	modQ3 := int2.ModInt(q3)
	x, _ := new(big.Int).SetString("64237ca0996e32e32523c31ad653d61744fd41042bba2e6e3981a848f7ecd6f5", 16)
	𝛾i, _ := new(big.Int).SetString("c7070cf90b032eb3c844411c918f51653659ea85d007e42fe9a2d29b2ac5a9c", 16)
	ki, _ := new(big.Int).SetString("64237ca0996e32e32523c31ad653d61744fd41042bba2e6e3981a848f7ecd6f5", 16)
	𝛽ʹ := new(big.Int).Add(x, q)
	// modQ3Mul𝛾iki := modQ3.Mul(𝛾i, ki)
	yʺ := modQ3.Add(𝛽ʹ, modQ3.Mul(𝛾i, ki))

	expectedyʺ, _ := new(big.Int).SetString("4dda57ec545360026cabe9eb31b6d7f74f103ea3d44081ab1d08be25747f17bd207522528b9379360160f810113713ff27bb3a4eaeb8e5b2302f949e6a43782", 16)

	/*
		common.Logger.Debugf("q3: %v, q3Mul𝛾iki: %v, yʺ: %v, expectedyʺ: %v",
			common.FormatBigInt(q3), common.FormatBigInt(modQ3Mul𝛾iki),
			common.FormatBigInt(yʺ), common.FormatBigInt(expectedyʺ))
	*/
	assert.True(test, yʺ.Cmp(expectedyʺ) == 0)
}
