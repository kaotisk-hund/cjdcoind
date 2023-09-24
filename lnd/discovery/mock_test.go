package discovery

import (
	"net"
	"sync"

	"github.com/kaotisk-hund/cjdcoind/btcec"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/channeldb"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnpeer"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwire"
	"github.com/kaotisk-hund/cjdcoind/wire"
)

// mockPeer implements the lnpeer.Peer interface and is used to test the
// gossiper's interaction with peers.
type mockPeer struct {
	pk       *btcec.PublicKey
	sentMsgs chan lnwire.Message
	quit     chan struct{}
}

var _ lnpeer.Peer = (*mockPeer)(nil)

func (p *mockPeer) SendMessage(_ bool, msgs ...lnwire.Message) er.R {
	if p.sentMsgs == nil && p.quit == nil {
		return nil
	}

	for _, msg := range msgs {
		select {
		case p.sentMsgs <- msg:
		case <-p.quit:
			return er.New("peer disconnected")
		}
	}

	return nil
}

func (p *mockPeer) SendMessageLazy(sync bool, msgs ...lnwire.Message) er.R {
	return p.SendMessage(sync, msgs...)
}

func (p *mockPeer) AddNewChannel(_ *channeldb.OpenChannel, _ <-chan struct{}) er.R {
	return nil
}
func (p *mockPeer) WipeChannel(_ *wire.OutPoint)  {}
func (p *mockPeer) IdentityKey() *btcec.PublicKey { return p.pk }
func (p *mockPeer) PubKey() [33]byte {
	var pubkey [33]byte
	copy(pubkey[:], p.pk.SerializeCompressed())
	return pubkey
}
func (p *mockPeer) Address() net.Addr { return nil }
func (p *mockPeer) QuitSignal() <-chan struct{} {
	return p.quit
}
func (p *mockPeer) LocalFeatures() *lnwire.FeatureVector {
	return nil
}
func (p *mockPeer) RemoteFeatures() *lnwire.FeatureVector {
	return nil
}

// mockMessageStore is an in-memory implementation of the MessageStore interface
// used for the gossiper's unit tests.
type mockMessageStore struct {
	sync.Mutex
	messages map[[33]byte]map[lnwire.Message]struct{}
}

func newMockMessageStore() *mockMessageStore {
	return &mockMessageStore{
		messages: make(map[[33]byte]map[lnwire.Message]struct{}),
	}
}

var _ GossipMessageStore = (*mockMessageStore)(nil)

func (s *mockMessageStore) AddMessage(msg lnwire.Message, pubKey [33]byte) er.R {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.messages[pubKey]; !ok {
		s.messages[pubKey] = make(map[lnwire.Message]struct{})
	}

	s.messages[pubKey][msg] = struct{}{}

	return nil
}

func (s *mockMessageStore) DeleteMessage(msg lnwire.Message, pubKey [33]byte) er.R {
	s.Lock()
	defer s.Unlock()

	peerMsgs, ok := s.messages[pubKey]
	if !ok {
		return nil
	}

	delete(peerMsgs, msg)
	return nil
}

func (s *mockMessageStore) Messages() (map[[33]byte][]lnwire.Message, er.R) {
	s.Lock()
	defer s.Unlock()

	msgs := make(map[[33]byte][]lnwire.Message, len(s.messages))
	for peer, peerMsgs := range s.messages {
		for msg := range peerMsgs {
			msgs[peer] = append(msgs[peer], msg)
		}
	}
	return msgs, nil
}

func (s *mockMessageStore) Peers() (map[[33]byte]struct{}, er.R) {
	s.Lock()
	defer s.Unlock()

	peers := make(map[[33]byte]struct{}, len(s.messages))
	for peer := range s.messages {
		peers[peer] = struct{}{}
	}
	return peers, nil
}

func (s *mockMessageStore) MessagesForPeer(pubKey [33]byte) ([]lnwire.Message, er.R) {
	s.Lock()
	defer s.Unlock()

	peerMsgs, ok := s.messages[pubKey]
	if !ok {
		return nil, nil
	}

	msgs := make([]lnwire.Message, 0, len(peerMsgs))
	for msg := range peerMsgs {
		msgs = append(msgs, msg)
	}

	return msgs, nil
}
