package wtwire_test

import (
	"testing"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/chaincfg"
	"github.com/kaotisk-hund/cjdcoind/chaincfg/chainhash"
	"github.com/kaotisk-hund/cjdcoind/lnd/feature"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwire"
	"github.com/kaotisk-hund/cjdcoind/lnd/watchtower/wtwire"
)

var (
	testnetChainHash = *chaincfg.TestNet3Params.GenesisHash
	mainnetChainHash = *chaincfg.MainNetParams.GenesisHash
)

type checkRemoteInitTest struct {
	name      string
	lFeatures *lnwire.RawFeatureVector
	lHash     chainhash.Hash
	rFeatures *lnwire.RawFeatureVector
	rHash     chainhash.Hash
	expErr    *er.ErrorCode
}

var checkRemoteInitTests = []checkRemoteInitTest{
	{
		name:      "same chain, local-optional remote-required",
		lFeatures: lnwire.NewRawFeatureVector(wtwire.AltruistSessionsOptional),
		lHash:     testnetChainHash,
		rFeatures: lnwire.NewRawFeatureVector(wtwire.AltruistSessionsRequired),
		rHash:     testnetChainHash,
	},
	{
		name:      "same chain, local-required remote-optional",
		lFeatures: lnwire.NewRawFeatureVector(wtwire.AltruistSessionsRequired),
		lHash:     testnetChainHash,
		rFeatures: lnwire.NewRawFeatureVector(wtwire.AltruistSessionsOptional),
		rHash:     testnetChainHash,
	},
	{
		name:      "different chain, local-optional remote-required",
		lFeatures: lnwire.NewRawFeatureVector(wtwire.AltruistSessionsOptional),
		lHash:     testnetChainHash,
		rFeatures: lnwire.NewRawFeatureVector(wtwire.AltruistSessionsRequired),
		rHash:     mainnetChainHash,
		expErr:    wtwire.ErrUnknownChainHash,
	},
	{
		name:      "different chain, local-required remote-optional",
		lFeatures: lnwire.NewRawFeatureVector(wtwire.AltruistSessionsOptional),
		lHash:     testnetChainHash,
		rFeatures: lnwire.NewRawFeatureVector(wtwire.AltruistSessionsRequired),
		rHash:     mainnetChainHash,
		expErr:    wtwire.ErrUnknownChainHash,
	},
	{
		name:      "same chain, remote-unknown-required",
		lFeatures: lnwire.NewRawFeatureVector(wtwire.AltruistSessionsOptional),
		lHash:     testnetChainHash,
		rFeatures: lnwire.NewRawFeatureVector(lnwire.GossipQueriesRequired),
		rHash:     testnetChainHash,
		expErr:    feature.ErrUnknownRequired,
	},
}

// TestCheckRemoteInit asserts the behavior of CheckRemoteInit when called with
// the remote party's Init message and the default wtwire.Features. We assert
// the validity of advertised features from the perspective of both client and
// server, as well as failure cases such as differing chain hashes or unknown
// required features.
func TestCheckRemoteInit(t *testing.T) {
	for _, test := range checkRemoteInitTests {
		t.Run(test.name, func(t *testing.T) {
			testCheckRemoteInit(t, test)
		})
	}
}

func testCheckRemoteInit(t *testing.T, test checkRemoteInitTest) {
	localInit := wtwire.NewInitMessage(test.lFeatures, test.lHash)
	remoteInit := wtwire.NewInitMessage(test.rFeatures, test.rHash)

	err := localInit.CheckRemoteInit(remoteInit, wtwire.FeatureNames)
	if !er.Cis(test.expErr, err) {
		t.Fatalf("error mismatch, want: %v, got: %v", test.expErr, err)
	}
}
