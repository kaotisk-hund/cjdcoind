package netann

import (
	"github.com/kaotisk-hund/cjdcoind/btcec"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/input"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwallet"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwire"
)

// SignAnnouncement signs any type of gossip message that is announced on the
// network.
func SignAnnouncement(signer lnwallet.MessageSigner, pubKey *btcec.PublicKey,
	msg lnwire.Message) (input.Signature, er.R) {

	var (
		data []byte
		err  er.R
	)

	switch m := msg.(type) {
	case *lnwire.ChannelAnnouncement:
		data, err = m.DataToSign()
	case *lnwire.ChannelUpdate:
		data, err = m.DataToSign()
	case *lnwire.NodeAnnouncement:
		data, err = m.DataToSign()
	default:
		return nil, er.Errorf("can't sign %T message", m)
	}
	if err != nil {
		return nil, er.Errorf("unable to get data to sign: %v", err)
	}

	return signer.SignMessage(pubKey, data)
}
