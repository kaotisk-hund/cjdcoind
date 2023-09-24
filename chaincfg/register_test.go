package chaincfg_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	. "github.com/kaotisk-hund/cjdcoind/chaincfg"
)

// Define some of the required parameters for a user-registered
// network.  This is necessary to test the registration of and
// lookup of encoding magics from the network.
var mockNetParams = Params{
	Name:             "mocknet",
	Net:              1<<32 - 1,
	PubKeyHashAddrID: 0x9f,
	ScriptHashAddrID: 0xf9,
	Bech32HRPSegwit:  "tc",
	HDPrivateKeyID:   [4]byte{0x01, 0x02, 0x03, 0x04},
	HDPublicKeyID:    [4]byte{0x05, 0x06, 0x07, 0x08},
}

func TestRegister(t *testing.T) {
	type registerTest struct {
		name   string
		params *Params
		err    er.R
	}
	type magicTest struct {
		magic byte
		valid bool
	}
	type prefixTest struct {
		prefix string
		valid  bool
	}
	type hdTest struct {
		priv []byte
		want []byte
		err  er.R
	}

	tests := []struct {
		name           string
		register       []registerTest
		p2pkhMagics    []magicTest
		p2shMagics     []magicTest
		segwitPrefixes []prefixTest
		hdMagics       []hdTest
	}{
		{
			name: "default networks",
			register: []registerTest{
				{
					name:   "duplicate mainnet",
					params: &MainNetParams,
					err:    ErrDuplicateNet.Default(),
				},
				{
					name:   "duplicate regtest",
					params: &RegressionNetParams,
					err:    ErrDuplicateNet.Default(),
				},
				{
					name:   "duplicate testnet3",
					params: &TestNet3Params,
					err:    ErrDuplicateNet.Default(),
				},
				{
					name:   "duplicate simnet",
					params: &SimNetParams,
					err:    ErrDuplicateNet.Default(),
				},
			},
			p2pkhMagics: []magicTest{
				{
					magic: MainNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: TestNet3Params.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: RegressionNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: SimNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: mockNetParams.PubKeyHashAddrID,
					valid: false,
				},
				{
					magic: 0xFF,
					valid: false,
				},
			},
			p2shMagics: []magicTest{
				{
					magic: MainNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: TestNet3Params.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: RegressionNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: SimNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: mockNetParams.ScriptHashAddrID,
					valid: false,
				},
				{
					magic: 0xFF,
					valid: false,
				},
			},
			segwitPrefixes: []prefixTest{
				{
					prefix: MainNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: TestNet3Params.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: RegressionNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: SimNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: strings.ToUpper(MainNetParams.Bech32HRPSegwit + "1"),
					valid:  true,
				},
				{
					prefix: mockNetParams.Bech32HRPSegwit + "1",
					valid:  false,
				},
				{
					prefix: "abc1",
					valid:  false,
				},
				{
					prefix: "1",
					valid:  false,
				},
				{
					prefix: MainNetParams.Bech32HRPSegwit,
					valid:  false,
				},
			},
			hdMagics: []hdTest{
				{
					priv: MainNetParams.HDPrivateKeyID[:],
					want: MainNetParams.HDPublicKeyID[:],
					err:  nil,
				},
				{
					priv: TestNet3Params.HDPrivateKeyID[:],
					want: TestNet3Params.HDPublicKeyID[:],
					err:  nil,
				},
				{
					priv: RegressionNetParams.HDPrivateKeyID[:],
					want: RegressionNetParams.HDPublicKeyID[:],
					err:  nil,
				},
				{
					priv: SimNetParams.HDPrivateKeyID[:],
					want: SimNetParams.HDPublicKeyID[:],
					err:  nil,
				},
				{
					priv: mockNetParams.HDPrivateKeyID[:],
					err:  ErrUnknownHDKeyID.Default(),
				},
				{
					priv: []byte{0xff, 0xff, 0xff, 0xff},
					err:  ErrUnknownHDKeyID.Default(),
				},
				{
					priv: []byte{0xff},
					err:  ErrUnknownHDKeyID.Default(),
				},
			},
		},
		{
			name: "register mocknet",
			register: []registerTest{
				{
					name:   "mocknet",
					params: &mockNetParams,
					err:    nil,
				},
			},
			p2pkhMagics: []magicTest{
				{
					magic: MainNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: TestNet3Params.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: RegressionNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: SimNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: mockNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: 0xFF,
					valid: false,
				},
			},
			p2shMagics: []magicTest{
				{
					magic: MainNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: TestNet3Params.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: RegressionNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: SimNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: mockNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: 0xFF,
					valid: false,
				},
			},
			segwitPrefixes: []prefixTest{
				{
					prefix: MainNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: TestNet3Params.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: RegressionNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: SimNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: strings.ToUpper(MainNetParams.Bech32HRPSegwit + "1"),
					valid:  true,
				},
				{
					prefix: mockNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: "abc1",
					valid:  false,
				},
				{
					prefix: "1",
					valid:  false,
				},
				{
					prefix: MainNetParams.Bech32HRPSegwit,
					valid:  false,
				},
			},
			hdMagics: []hdTest{
				{
					priv: mockNetParams.HDPrivateKeyID[:],
					want: mockNetParams.HDPublicKeyID[:],
					err:  nil,
				},
			},
		},
		{
			name: "more duplicates",
			register: []registerTest{
				{
					name:   "duplicate mainnet",
					params: &MainNetParams,
					err:    ErrDuplicateNet.Default(),
				},
				{
					name:   "duplicate regtest",
					params: &RegressionNetParams,
					err:    ErrDuplicateNet.Default(),
				},
				{
					name:   "duplicate testnet3",
					params: &TestNet3Params,
					err:    ErrDuplicateNet.Default(),
				},
				{
					name:   "duplicate simnet",
					params: &SimNetParams,
					err:    ErrDuplicateNet.Default(),
				},
				{
					name:   "duplicate mocknet",
					params: &mockNetParams,
					err:    ErrDuplicateNet.Default(),
				},
			},
			p2pkhMagics: []magicTest{
				{
					magic: MainNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: TestNet3Params.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: RegressionNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: SimNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: mockNetParams.PubKeyHashAddrID,
					valid: true,
				},
				{
					magic: 0xFF,
					valid: false,
				},
			},
			p2shMagics: []magicTest{
				{
					magic: MainNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: TestNet3Params.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: RegressionNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: SimNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: mockNetParams.ScriptHashAddrID,
					valid: true,
				},
				{
					magic: 0xFF,
					valid: false,
				},
			},
			segwitPrefixes: []prefixTest{
				{
					prefix: MainNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: TestNet3Params.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: RegressionNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: SimNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: strings.ToUpper(MainNetParams.Bech32HRPSegwit + "1"),
					valid:  true,
				},
				{
					prefix: mockNetParams.Bech32HRPSegwit + "1",
					valid:  true,
				},
				{
					prefix: "abc1",
					valid:  false,
				},
				{
					prefix: "1",
					valid:  false,
				},
				{
					prefix: MainNetParams.Bech32HRPSegwit,
					valid:  false,
				},
			},
			hdMagics: []hdTest{
				{
					priv: MainNetParams.HDPrivateKeyID[:],
					want: MainNetParams.HDPublicKeyID[:],
					err:  nil,
				},
				{
					priv: TestNet3Params.HDPrivateKeyID[:],
					want: TestNet3Params.HDPublicKeyID[:],
					err:  nil,
				},
				{
					priv: RegressionNetParams.HDPrivateKeyID[:],
					want: RegressionNetParams.HDPublicKeyID[:],
					err:  nil,
				},
				{
					priv: SimNetParams.HDPrivateKeyID[:],
					want: SimNetParams.HDPublicKeyID[:],
					err:  nil,
				},
				{
					priv: mockNetParams.HDPrivateKeyID[:],
					want: mockNetParams.HDPublicKeyID[:],
					err:  nil,
				},
				{
					priv: []byte{0xff, 0xff, 0xff, 0xff},
					err:  ErrUnknownHDKeyID.Default(),
				},
				{
					priv: []byte{0xff},
					err:  ErrUnknownHDKeyID.Default(),
				},
			},
		},
	}

	for _, test := range tests {
		for _, regTest := range test.register {
			err := Register(regTest.params)
			if !er.Equals(err, regTest.err) {
				t.Errorf("%s:%s: Registered network with unexpected error: got %v expected %v",
					test.name, regTest.name, err, regTest.err)
			}
		}
		for i, magTest := range test.p2pkhMagics {
			valid := IsPubKeyHashAddrID(magTest.magic)
			if valid != magTest.valid {
				t.Errorf("%s: P2PKH magic %d valid mismatch: got %v expected %v",
					test.name, i, valid, magTest.valid)
			}
		}
		for i, magTest := range test.p2shMagics {
			valid := IsScriptHashAddrID(magTest.magic)
			if valid != magTest.valid {
				t.Errorf("%s: P2SH magic %d valid mismatch: got %v expected %v",
					test.name, i, valid, magTest.valid)
			}
		}
		for i, prxTest := range test.segwitPrefixes {
			valid := IsBech32SegwitPrefix(prxTest.prefix)
			if valid != prxTest.valid {
				t.Errorf("%s: segwit prefix %s (%d) valid mismatch: got %v expected %v",
					test.name, prxTest.prefix, i, valid, prxTest.valid)
			}
		}
		for i, magTest := range test.hdMagics {
			pubKey, err := HDPrivateKeyToPublicKeyID(magTest.priv)
			if !er.Equals(err, magTest.err) {
				t.Errorf("%s: HD magic %d mismatched error: got %v expected %v ",
					test.name, i, err, magTest.err)
				continue
			}
			if magTest.err == nil && !bytes.Equal(pubKey, magTest.want) {
				t.Errorf("%s: HD magic %d private and public mismatch: got %v expected %v ",
					test.name, i, pubKey, magTest.want)
			}
		}
	}
}
