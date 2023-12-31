package netann_test

import (
	"testing"
	"time"

	"github.com/kaotisk-hund/cjdcoind/btcec"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/input"
	"github.com/kaotisk-hund/cjdcoind/lnd/keychain"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwallet"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwire"
	"github.com/kaotisk-hund/cjdcoind/lnd/netann"
	"github.com/kaotisk-hund/cjdcoind/lnd/routing"
)

type mockSigner struct {
	err *er.ErrorCode
}

func (m *mockSigner) SignMessage(pk *btcec.PublicKey,
	msg []byte) (input.Signature, er.R) {

	if m.err != nil {
		return nil, m.err.Default()
	}

	return nil, nil
}

var _ lnwallet.MessageSigner = (*mockSigner)(nil)

var (
	privKey, _    = btcec.NewPrivateKey(btcec.S256())
	privKeySigner = &keychain.PrivKeyDigestSigner{PrivKey: privKey}

	pubKey = privKey.PubKey()

	errFailedToSign = er.GenericErrorType.CodeWithDetail("errFailedToSign",
		"unable to sign message")
)

type updateDisableTest struct {
	name         string
	startEnabled bool
	disable      bool
	startTime    time.Time
	signer       lnwallet.MessageSigner
	expErr       *er.ErrorCode
}

var updateDisableTests = []updateDisableTest{
	{
		name:         "working signer enabled to disabled",
		startEnabled: true,
		disable:      true,
		startTime:    time.Now(),
		signer:       netann.NewNodeSigner(privKeySigner),
	},
	{
		name:         "working signer enabled to enabled",
		startEnabled: true,
		disable:      false,
		startTime:    time.Now(),
		signer:       netann.NewNodeSigner(privKeySigner),
	},
	{
		name:         "working signer disabled to enabled",
		startEnabled: false,
		disable:      false,
		startTime:    time.Now(),
		signer:       netann.NewNodeSigner(privKeySigner),
	},
	{
		name:         "working signer disabled to disabled",
		startEnabled: false,
		disable:      true,
		startTime:    time.Now(),
		signer:       netann.NewNodeSigner(privKeySigner),
	},
	{
		name:         "working signer future monotonicity",
		startEnabled: true,
		disable:      true,
		startTime:    time.Now().Add(time.Hour), // must increment
		signer:       netann.NewNodeSigner(privKeySigner),
	},
	{
		name:      "failing signer",
		startTime: time.Now(),
		signer:    &mockSigner{err: errFailedToSign},
		expErr:    errFailedToSign,
	},
	{
		name:      "invalid sig from signer",
		startTime: time.Now(),
		signer:    &mockSigner{}, // returns a nil signature
		expErr:    lnwire.ErrEmptySignature,
	},
}

// TestUpdateDisableFlag checks the behavior of UpdateDisableFlag, asserting
// that the proper channel flags are set, the timestamp always increases
// monotonically, and that the correct errors are returned in the event that the
// signer is unable to produce a signature.
func TestUpdateDisableFlag(t *testing.T) {
	t.Parallel()

	for _, tc := range updateDisableTests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Create the initial update, the only fields we are
			// concerned with in this test are the timestamp and the
			// channel flags.
			ogUpdate := &lnwire.ChannelUpdate{
				Timestamp: uint32(tc.startTime.Unix()),
			}
			if !tc.startEnabled {
				ogUpdate.ChannelFlags |= lnwire.ChanUpdateDisabled
			}

			// Create new update to sign using the same fields as
			// the original. UpdateDisableFlag will mutate the
			// passed channel update, so we keep the old one to test
			// against.
			newUpdate := &lnwire.ChannelUpdate{
				Timestamp:    ogUpdate.Timestamp,
				ChannelFlags: ogUpdate.ChannelFlags,
			}

			// Attempt to update and sign the new update, specifying
			// disabled or enabled as prescribed in the test case.
			err := netann.SignChannelUpdate(
				tc.signer, pubKey, newUpdate,
				netann.ChanUpdSetDisable(tc.disable),
				netann.ChanUpdSetTimestamp,
			)

			var fail bool
			switch {

			// Both nil, pass.
			case tc.expErr == nil && err == nil:

			// Both non-nil, compare error strings since some
			// methods don't return concrete error types.
			case tc.expErr != nil && err != nil:
				if !er.Cis(tc.expErr, err) {
					fail = true
				}

			// Otherwise, one is nil and one is non-nil.
			default:
				fail = true
			}

			if fail {
				t.Fatalf("expected error: %v, got %v",
					tc.expErr, err)
			}

			// Exit early if the test expected a failure.
			if tc.expErr != nil {
				return
			}

			// Verify that the timestamp has increased from the
			// original update.
			if newUpdate.Timestamp <= ogUpdate.Timestamp {
				t.Fatalf("update timestamp should be "+
					"monotonically increasing, "+
					"original: %d, new %d",
					ogUpdate.Timestamp, newUpdate.Timestamp)
			}

			// Verify that the disabled flag is properly set.
			disabled := newUpdate.ChannelFlags&
				lnwire.ChanUpdateDisabled != 0
			if disabled != tc.disable {
				t.Fatalf("expected disable:%v, found:%v",
					tc.disable, disabled)
			}

			// Finally, validate the signature using the router's
			// verification logic.
			err = routing.ValidateChannelUpdateAnn(
				pubKey, 0, newUpdate,
			)
			if err != nil {
				t.Fatalf("channel update failed to "+
					"validate: %v", err)
			}
		})
	}
}
