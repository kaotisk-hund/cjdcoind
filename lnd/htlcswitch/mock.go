package htlcswitch

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaotisk-hund/cjdcoind/btcec"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/btcutil/util"
	sphinx "github.com/kaotisk-hund/cjdcoind/lightning-onion"
	"github.com/kaotisk-hund/cjdcoind/lnd/chainntnfs"
	"github.com/kaotisk-hund/cjdcoind/lnd/channeldb"
	"github.com/kaotisk-hund/cjdcoind/lnd/clock"
	"github.com/kaotisk-hund/cjdcoind/lnd/contractcourt"
	"github.com/kaotisk-hund/cjdcoind/lnd/htlcswitch/hop"
	"github.com/kaotisk-hund/cjdcoind/lnd/invoices"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnpeer"
	"github.com/kaotisk-hund/cjdcoind/lnd/lntest/mock"
	"github.com/kaotisk-hund/cjdcoind/lnd/lntypes"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwallet/chainfee"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwire"
	"github.com/kaotisk-hund/cjdcoind/lnd/ticker"
	"github.com/kaotisk-hund/cjdcoind/wire"
)

type mockPreimageCache struct {
	sync.Mutex
	preimageMap map[lntypes.Hash]lntypes.Preimage
}

func newMockPreimageCache() *mockPreimageCache {
	return &mockPreimageCache{
		preimageMap: make(map[lntypes.Hash]lntypes.Preimage),
	}
}

func (m *mockPreimageCache) LookupPreimage(
	hash lntypes.Hash) (lntypes.Preimage, bool) {

	m.Lock()
	defer m.Unlock()

	p, ok := m.preimageMap[hash]
	return p, ok
}

func (m *mockPreimageCache) AddPreimages(preimages ...lntypes.Preimage) er.R {
	m.Lock()
	defer m.Unlock()

	for _, preimage := range preimages {
		m.preimageMap[preimage.Hash()] = preimage
	}

	return nil
}

func (m *mockPreimageCache) SubscribeUpdates() *contractcourt.WitnessSubscription {
	return nil
}

type mockFeeEstimator struct {
	byteFeeIn chan chainfee.SatPerKWeight

	quit chan struct{}
}

func (m *mockFeeEstimator) EstimateFeePerKW(
	numBlocks uint32) (chainfee.SatPerKWeight, er.R) {

	select {
	case feeRate := <-m.byteFeeIn:
		return feeRate, nil
	case <-m.quit:
		return 0, er.Errorf("exiting")
	}
}

func (m *mockFeeEstimator) RelayFeePerKW() chainfee.SatPerKWeight {
	return 1e3
}

func (m *mockFeeEstimator) Start() er.R {
	return nil
}
func (m *mockFeeEstimator) Stop() er.R {
	close(m.quit)
	return nil
}

var _ chainfee.Estimator = (*mockFeeEstimator)(nil)

type mockForwardingLog struct {
	sync.Mutex

	events map[time.Time]channeldb.ForwardingEvent
}

func (m *mockForwardingLog) AddForwardingEvents(events []channeldb.ForwardingEvent) er.R {
	m.Lock()
	defer m.Unlock()

	for _, event := range events {
		m.events[event.Timestamp] = event
	}

	return nil
}

type mockServer struct {
	started  int32 // To be used atomically.
	shutdown int32 // To be used atomically.
	wg       sync.WaitGroup
	quit     chan struct{}

	t testing.TB

	name     string
	messages chan lnwire.Message

	id         [33]byte
	htlcSwitch *Switch

	registry         *mockInvoiceRegistry
	pCache           *mockPreimageCache
	interceptorFuncs []messageInterceptor
}

var _ lnpeer.Peer = (*mockServer)(nil)

func initDB() (*channeldb.DB, er.R) {
	tempPath, errr := ioutil.TempDir("", "switchdb")
	if errr != nil {
		return nil, er.E(errr)
	}

	db, err := channeldb.Open(tempPath)
	if err != nil {
		return nil, err
	}

	return db, err
}

func initSwitchWithDB(startingHeight uint32, db *channeldb.DB) (*Switch, er.R) {
	var err er.R

	if db == nil {
		db, err = initDB()
		if err != nil {
			return nil, err
		}
	}

	cfg := Config{
		DB:             db,
		SwitchPackager: channeldb.NewSwitchPackager(),
		FwdingLog: &mockForwardingLog{
			events: make(map[time.Time]channeldb.ForwardingEvent),
		},
		FetchLastChannelUpdate: func(lnwire.ShortChannelID) (*lnwire.ChannelUpdate, er.R) {
			return nil, nil
		},
		Notifier: &mock.ChainNotifier{
			SpendChan: make(chan *chainntnfs.SpendDetail),
			EpochChan: make(chan *chainntnfs.BlockEpoch),
			ConfChan:  make(chan *chainntnfs.TxConfirmation),
		},
		FwdEventTicker: ticker.NewForce(DefaultFwdEventInterval),
		LogEventTicker: ticker.NewForce(DefaultLogInterval),
		AckEventTicker: ticker.NewForce(DefaultAckInterval),
		HtlcNotifier:   &mockHTLCNotifier{},
		Clock:          clock.NewDefaultClock(),
		HTLCExpiry:     time.Hour,
	}

	return New(cfg, startingHeight)
}

func newMockServer(t testing.TB, name string, startingHeight uint32,
	db *channeldb.DB, defaultDelta uint32) (*mockServer, er.R) {

	var id [33]byte
	h := sha256.Sum256([]byte(name))
	copy(id[:], h[:])

	pCache := newMockPreimageCache()

	htlcSwitch, err := initSwitchWithDB(startingHeight, db)
	if err != nil {
		return nil, err
	}

	registry := newMockRegistry(defaultDelta)

	return &mockServer{
		t:                t,
		id:               id,
		name:             name,
		messages:         make(chan lnwire.Message, 3000),
		quit:             make(chan struct{}),
		registry:         registry,
		htlcSwitch:       htlcSwitch,
		pCache:           pCache,
		interceptorFuncs: make([]messageInterceptor, 0),
	}, nil
}

func (s *mockServer) Start() er.R {
	if !atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		return er.New("mock server already started")
	}

	if err := s.htlcSwitch.Start(); err != nil {
		return err
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		defer func() {
			s.htlcSwitch.Stop()
		}()

		for {
			select {
			case msg := <-s.messages:
				var shouldSkip bool

				for _, interceptor := range s.interceptorFuncs {
					skip, err := interceptor(msg)
					if err != nil {
						s.t.Fatalf("%v: error in the "+
							"interceptor: %v", s.name, err)
						return
					}
					shouldSkip = shouldSkip || skip
				}

				if shouldSkip {
					continue
				}

				if err := s.readHandler(msg); err != nil {
					s.t.Fatal(err)
					return
				}
			case <-s.quit:
				return
			}
		}
	}()

	return nil
}

func (s *mockServer) QuitSignal() <-chan struct{} {
	return s.quit
}

// mockHopIterator represents the test version of hop iterator which instead
// of encrypting the path in onion blob just stores the path as a list of hops.
type mockHopIterator struct {
	hops []*hop.Payload
}

func newMockHopIterator(hops ...*hop.Payload) hop.Iterator {
	return &mockHopIterator{hops: hops}
}

func (r *mockHopIterator) HopPayload() (*hop.Payload, er.R) {
	h := r.hops[0]
	r.hops = r.hops[1:]
	return h, nil
}

func (r *mockHopIterator) ExtraOnionBlob() []byte {
	return nil
}

func (r *mockHopIterator) ExtractErrorEncrypter(
	extracter hop.ErrorEncrypterExtracter) (hop.ErrorEncrypter,
	lnwire.FailCode) {

	return extracter(nil)
}

func (r *mockHopIterator) EncodeNextHop(w io.Writer) er.R {
	var hopLength [4]byte
	binary.BigEndian.PutUint32(hopLength[:], uint32(len(r.hops)))

	if _, err := util.Write(w, hopLength[:]); err != nil {
		return err
	}

	for _, hop := range r.hops {
		fwdInfo := hop.ForwardingInfo()
		if err := encodeFwdInfo(w, &fwdInfo); err != nil {
			return err
		}
	}

	return nil
}

func encodeFwdInfo(w io.Writer, f *hop.ForwardingInfo) er.R {
	if _, err := util.Write(w, []byte{byte(f.Network)}); err != nil {
		return err
	}

	if err := util.WriteBin(w, binary.BigEndian, f.NextHop); err != nil {
		return err
	}

	if err := util.WriteBin(w, binary.BigEndian, f.AmountToForward); err != nil {
		return err
	}

	if err := util.WriteBin(w, binary.BigEndian, f.OutgoingCTLV); err != nil {
		return err
	}

	return nil
}

var _ hop.Iterator = (*mockHopIterator)(nil)

// mockObfuscator mock implementation of the failure obfuscator which only
// encodes the failure and do not makes any onion obfuscation.
type mockObfuscator struct {
	ogPacket *sphinx.OnionPacket
	failure  lnwire.FailureMessage
}

// NewMockObfuscator initializes a dummy mockObfuscator used for testing.
func NewMockObfuscator() hop.ErrorEncrypter {
	return &mockObfuscator{}
}

func (o *mockObfuscator) OnionPacket() *sphinx.OnionPacket {
	return o.ogPacket
}

func (o *mockObfuscator) Type() hop.EncrypterType {
	return hop.EncrypterTypeMock
}

func (o *mockObfuscator) Encode(w io.Writer) er.R {
	return nil
}

func (o *mockObfuscator) Decode(r io.Reader) er.R {
	return nil
}

func (o *mockObfuscator) Reextract(
	extracter hop.ErrorEncrypterExtracter) er.R {

	return nil
}

func (o *mockObfuscator) EncryptFirstHop(failure lnwire.FailureMessage) (
	lnwire.OpaqueReason, er.R) {

	o.failure = failure

	var b bytes.Buffer
	if err := lnwire.EncodeFailure(&b, failure, 0); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (o *mockObfuscator) IntermediateEncrypt(reason lnwire.OpaqueReason) lnwire.OpaqueReason {
	return reason
}

func (o *mockObfuscator) EncryptMalformedError(reason lnwire.OpaqueReason) lnwire.OpaqueReason {
	return reason
}

// mockDeobfuscator mock implementation of the failure deobfuscator which
// only decodes the failure do not makes any onion obfuscation.
type mockDeobfuscator struct{}

func newMockDeobfuscator() ErrorDecrypter {
	return &mockDeobfuscator{}
}

func (o *mockDeobfuscator) DecryptError(reason lnwire.OpaqueReason) (*ForwardingError, er.R) {

	r := bytes.NewReader(reason)
	failure, err := lnwire.DecodeFailure(r, 0)
	if err != nil {
		return nil, err
	}

	return NewForwardingError(failure, 1), nil
}

var _ ErrorDecrypter = (*mockDeobfuscator)(nil)

// mockIteratorDecoder test version of hop iterator decoder which decodes the
// encoded array of hops.
type mockIteratorDecoder struct {
	mu sync.RWMutex

	responses map[[32]byte][]hop.DecodeHopIteratorResponse

	decodeFail bool
}

func newMockIteratorDecoder() *mockIteratorDecoder {
	return &mockIteratorDecoder{
		responses: make(map[[32]byte][]hop.DecodeHopIteratorResponse),
	}
}

func (p *mockIteratorDecoder) DecodeHopIterator(r io.Reader, rHash []byte,
	cltv uint32) (hop.Iterator, lnwire.FailCode) {

	var b [4]byte
	_, err := r.Read(b[:])
	if err != nil {
		return nil, lnwire.CodeTemporaryChannelFailure
	}
	hopLength := binary.BigEndian.Uint32(b[:])

	hops := make([]*hop.Payload, hopLength)
	for i := uint32(0); i < hopLength; i++ {
		var f hop.ForwardingInfo
		if err := decodeFwdInfo(r, &f); err != nil {
			return nil, lnwire.CodeTemporaryChannelFailure
		}

		var nextHopBytes [8]byte
		binary.BigEndian.PutUint64(nextHopBytes[:], f.NextHop.ToUint64())

		hops[i] = hop.NewLegacyPayload(&sphinx.HopData{
			Realm:         [1]byte{}, // hop.BitcoinNetwork
			NextAddress:   nextHopBytes,
			ForwardAmount: uint64(f.AmountToForward),
			OutgoingCltv:  f.OutgoingCTLV,
		})
	}

	return newMockHopIterator(hops...), lnwire.CodeNone
}

func (p *mockIteratorDecoder) DecodeHopIterators(id []byte,
	reqs []hop.DecodeHopIteratorRequest) (
	[]hop.DecodeHopIteratorResponse, er.R) {

	idHash := sha256.Sum256(id)

	p.mu.RLock()
	if resps, ok := p.responses[idHash]; ok {
		p.mu.RUnlock()
		return resps, nil
	}
	p.mu.RUnlock()

	batchSize := len(reqs)

	resps := make([]hop.DecodeHopIteratorResponse, 0, batchSize)
	for _, req := range reqs {
		iterator, failcode := p.DecodeHopIterator(
			req.OnionReader, req.RHash, req.IncomingCltv,
		)

		if p.decodeFail {
			failcode = lnwire.CodeTemporaryChannelFailure
		}

		resp := hop.DecodeHopIteratorResponse{
			HopIterator: iterator,
			FailCode:    failcode,
		}
		resps = append(resps, resp)
	}

	p.mu.Lock()
	p.responses[idHash] = resps
	p.mu.Unlock()

	return resps, nil
}

func decodeFwdInfo(r io.Reader, f *hop.ForwardingInfo) er.R {
	var net [1]byte
	if _, err := r.Read(net[:]); err != nil {
		return er.E(err)
	}
	f.Network = hop.Network(net[0])

	if err := util.ReadBin(r, binary.BigEndian, &f.NextHop); err != nil {
		return err
	}

	if err := util.ReadBin(r, binary.BigEndian, &f.AmountToForward); err != nil {
		return err
	}

	if err := util.ReadBin(r, binary.BigEndian, &f.OutgoingCTLV); err != nil {
		return err
	}

	return nil
}

// messageInterceptor is function that handles the incoming peer messages and
// may decide should the peer skip the message or not.
type messageInterceptor func(m lnwire.Message) (bool, er.R)

// Record is used to set the function which will be triggered when new
// lnwire message was received.
func (s *mockServer) intersect(f messageInterceptor) {
	s.interceptorFuncs = append(s.interceptorFuncs, f)
}

func (s *mockServer) SendMessage(sync bool, msgs ...lnwire.Message) er.R {

	for _, msg := range msgs {
		select {
		case s.messages <- msg:
		case <-s.quit:
			return er.New("server is stopped")
		}
	}

	return nil
}

func (s *mockServer) SendMessageLazy(sync bool, msgs ...lnwire.Message) er.R {
	panic("not implemented")
}

func (s *mockServer) readHandler(message lnwire.Message) er.R {
	var targetChan lnwire.ChannelID

	switch msg := message.(type) {
	case *lnwire.UpdateAddHTLC:
		targetChan = msg.ChanID
	case *lnwire.UpdateFulfillHTLC:
		targetChan = msg.ChanID
	case *lnwire.UpdateFailHTLC:
		targetChan = msg.ChanID
	case *lnwire.UpdateFailMalformedHTLC:
		targetChan = msg.ChanID
	case *lnwire.RevokeAndAck:
		targetChan = msg.ChanID
	case *lnwire.CommitSig:
		targetChan = msg.ChanID
	case *lnwire.FundingLocked:
		// Ignore
		return nil
	case *lnwire.ChannelReestablish:
		targetChan = msg.ChanID
	case *lnwire.UpdateFee:
		targetChan = msg.ChanID
	default:
		return er.Errorf("unknown message type: %T", msg)
	}

	// Dispatch the commitment update message to the proper channel link
	// dedicated to this channel. If the link is not found, we will discard
	// the message.
	link, err := s.htlcSwitch.GetLink(targetChan)
	if err != nil {
		return nil
	}

	// Create goroutine for this, in order to be able to properly stop
	// the server when handler stacked (server unavailable)
	link.HandleChannelUpdate(message)

	return nil
}

func (s *mockServer) PubKey() [33]byte {
	return s.id
}

func (s *mockServer) IdentityKey() *btcec.PublicKey {
	pubkey, _ := btcec.ParsePubKey(s.id[:], btcec.S256())
	return pubkey
}

func (s *mockServer) Address() net.Addr {
	return nil
}

func (s *mockServer) AddNewChannel(channel *channeldb.OpenChannel,
	cancel <-chan struct{}) er.R {

	return nil
}

func (s *mockServer) WipeChannel(*wire.OutPoint) {}

func (s *mockServer) LocalFeatures() *lnwire.FeatureVector {
	return nil
}

func (s *mockServer) RemoteFeatures() *lnwire.FeatureVector {
	return nil
}

func (s *mockServer) Stop() er.R {
	if !atomic.CompareAndSwapInt32(&s.shutdown, 0, 1) {
		return nil
	}

	close(s.quit)
	s.wg.Wait()

	return nil
}

func (s *mockServer) String() string {
	return s.name
}

type mockChannelLink struct {
	htlcSwitch *Switch

	shortChanID lnwire.ShortChannelID

	chanID lnwire.ChannelID

	peer lnpeer.Peer

	mailBox MailBox

	packets chan *htlcPacket

	eligible bool

	htlcID uint64

	checkHtlcTransitResult *LinkError

	checkHtlcForwardResult *LinkError
}

// completeCircuit is a helper method for adding the finalized payment circuit
// to the switch's circuit map. In testing, this should be executed after
// receiving an htlc from the downstream packets channel.
func (f *mockChannelLink) completeCircuit(cjdcoin *htlcPacket) er.R {
	switch htlc := cjdcoin.htlc.(type) {
	case *lnwire.UpdateAddHTLC:
		cjdcoin.outgoingChanID = f.shortChanID
		cjdcoin.outgoingHTLCID = f.htlcID
		htlc.ID = f.htlcID

		keystone := Keystone{cjdcoin.inKey(), cjdcoin.outKey()}
		if err := f.htlcSwitch.openCircuits(keystone); err != nil {
			return err
		}

		f.htlcID++

	case *lnwire.UpdateFulfillHTLC, *lnwire.UpdateFailHTLC:
		err := f.htlcSwitch.teardownCircuit(cjdcoin)
		if err != nil {
			return err
		}
	}

	f.mailBox.AckPacket(cjdcoin.inKey())

	return nil
}

func (f *mockChannelLink) deleteCircuit(cjdcoin *htlcPacket) er.R {
	return f.htlcSwitch.deleteCircuits(cjdcoin.inKey())
}

func newMockChannelLink(htlcSwitch *Switch, chanID lnwire.ChannelID,
	shortChanID lnwire.ShortChannelID, peer lnpeer.Peer, eligible bool,
) *mockChannelLink {

	return &mockChannelLink{
		htlcSwitch:  htlcSwitch,
		chanID:      chanID,
		shortChanID: shortChanID,
		peer:        peer,
		eligible:    eligible,
	}
}

func (f *mockChannelLink) HandleSwitchPacket(cjdcoin *htlcPacket) er.R {
	f.mailBox.AddPacket(cjdcoin)
	return nil
}

func (f *mockChannelLink) HandleLocalAddPacket(cjdcoin *htlcPacket) er.R {
	_ = f.mailBox.AddPacket(cjdcoin)
	return nil
}

func (f *mockChannelLink) HandleChannelUpdate(lnwire.Message) {
}

func (f *mockChannelLink) UpdateForwardingPolicy(_ ForwardingPolicy) {
}
func (f *mockChannelLink) CheckHtlcForward([32]byte, lnwire.MilliSatoshi,
	lnwire.MilliSatoshi, uint32, uint32, uint32) *LinkError {

	return f.checkHtlcForwardResult
}

func (f *mockChannelLink) CheckHtlcTransit(payHash [32]byte,
	amt lnwire.MilliSatoshi, timeout uint32,
	heightNow uint32) *LinkError {

	return f.checkHtlcTransitResult
}

func (f *mockChannelLink) Stats() (uint64, lnwire.MilliSatoshi, lnwire.MilliSatoshi) {
	return 0, 0, 0
}

func (f *mockChannelLink) AttachMailBox(mailBox MailBox) {
	f.mailBox = mailBox
	f.packets = mailBox.PacketOutBox()
}

func (f *mockChannelLink) Start() er.R {
	f.mailBox.ResetMessages()
	f.mailBox.ResetPackets()
	return nil
}

func (f *mockChannelLink) ChanID() lnwire.ChannelID                     { return f.chanID }
func (f *mockChannelLink) ShortChanID() lnwire.ShortChannelID           { return f.shortChanID }
func (f *mockChannelLink) Bandwidth() lnwire.MilliSatoshi               { return 99999999 }
func (f *mockChannelLink) Peer() lnpeer.Peer                            { return f.peer }
func (f *mockChannelLink) ChannelPoint() *wire.OutPoint                 { return &wire.OutPoint{} }
func (f *mockChannelLink) Stop()                                        {}
func (f *mockChannelLink) EligibleToForward() bool                      { return f.eligible }
func (f *mockChannelLink) setLiveShortChanID(sid lnwire.ShortChannelID) { f.shortChanID = sid }
func (f *mockChannelLink) UpdateShortChanID() (lnwire.ShortChannelID, er.R) {
	f.eligible = true
	return f.shortChanID, nil
}

var _ ChannelLink = (*mockChannelLink)(nil)

func newDB() (*channeldb.DB, func(), er.R) {
	// First, create a temporary directory to be used for the duration of
	// this test.
	tempDirName, errr := ioutil.TempDir("", "channeldb")
	if errr != nil {
		return nil, nil, er.E(errr)
	}

	// Next, create channeldb for the first time.
	cdb, err := channeldb.Open(tempDirName)
	if err != nil {
		os.RemoveAll(tempDirName)
		return nil, nil, err
	}

	cleanUp := func() {
		cdb.Close()
		os.RemoveAll(tempDirName)
	}

	return cdb, cleanUp, nil
}

const testInvoiceCltvExpiry = 6

type mockInvoiceRegistry struct {
	settleChan chan lntypes.Hash

	registry *invoices.InvoiceRegistry

	cleanup func()
}

func newMockRegistry(minDelta uint32) *mockInvoiceRegistry {
	cdb, cleanup, err := newDB()
	if err != nil {
		panic(err)
	}

	registry := invoices.NewRegistry(
		cdb,
		invoices.NewInvoiceExpiryWatcher(clock.NewDefaultClock()),
		&invoices.RegistryConfig{
			FinalCltvRejectDelta: 5,
		},
	)
	registry.Start()

	return &mockInvoiceRegistry{
		registry: registry,
		cleanup:  cleanup,
	}
}

func (i *mockInvoiceRegistry) LookupInvoice(rHash lntypes.Hash) (
	channeldb.Invoice, er.R) {

	return i.registry.LookupInvoice(rHash)
}

func (i *mockInvoiceRegistry) SettleHodlInvoice(preimage lntypes.Preimage) er.R {
	return i.registry.SettleHodlInvoice(preimage)
}

func (i *mockInvoiceRegistry) NotifyExitHopHtlc(rhash lntypes.Hash,
	amt lnwire.MilliSatoshi, expiry uint32, currentHeight int32,
	circuitKey channeldb.CircuitKey, hodlChan chan<- interface{},
	payload invoices.Payload) (invoices.HtlcResolution, er.R) {

	event, err := i.registry.NotifyExitHopHtlc(
		rhash, amt, expiry, currentHeight, circuitKey, hodlChan,
		payload,
	)
	if err != nil {
		return nil, err
	}
	if i.settleChan != nil {
		i.settleChan <- rhash
	}

	return event, nil
}

func (i *mockInvoiceRegistry) CancelInvoice(payHash lntypes.Hash) er.R {
	return i.registry.CancelInvoice(payHash)
}

func (i *mockInvoiceRegistry) AddInvoice(invoice channeldb.Invoice,
	paymentHash lntypes.Hash) er.R {

	_, err := i.registry.AddInvoice(&invoice, paymentHash)
	return err
}

func (i *mockInvoiceRegistry) HodlUnsubscribeAll(subscriber chan<- interface{}) {
	i.registry.HodlUnsubscribeAll(subscriber)
}

var _ InvoiceDatabase = (*mockInvoiceRegistry)(nil)

type mockCircuitMap struct {
	lookup chan *PaymentCircuit
}

var _ CircuitMap = (*mockCircuitMap)(nil)

func (m *mockCircuitMap) OpenCircuits(...Keystone) er.R {
	return nil
}

func (m *mockCircuitMap) TrimOpenCircuits(chanID lnwire.ShortChannelID,
	start uint64) er.R {
	return nil
}

func (m *mockCircuitMap) DeleteCircuits(inKeys ...CircuitKey) er.R {
	return nil
}

func (m *mockCircuitMap) CommitCircuits(
	circuit ...*PaymentCircuit) (*CircuitFwdActions, er.R) {

	return nil, nil
}

func (m *mockCircuitMap) CloseCircuit(outKey CircuitKey) (*PaymentCircuit,
	er.R) {
	return nil, nil
}

func (m *mockCircuitMap) FailCircuit(inKey CircuitKey) (*PaymentCircuit,
	er.R) {
	return nil, nil
}

func (m *mockCircuitMap) LookupCircuit(inKey CircuitKey) *PaymentCircuit {
	return <-m.lookup
}

func (m *mockCircuitMap) LookupOpenCircuit(outKey CircuitKey) *PaymentCircuit {
	return nil
}

func (m *mockCircuitMap) LookupByPaymentHash(hash [32]byte) []*PaymentCircuit {
	return nil
}

func (m *mockCircuitMap) NumPending() int {
	return 0
}

func (m *mockCircuitMap) NumOpen() int {
	return 0
}

type mockOnionErrorDecryptor struct {
	sourceIdx int
	message   []byte
	err       er.R
}

func (m *mockOnionErrorDecryptor) DecryptError(encryptedData []byte) (
	*sphinx.DecryptedError, er.R) {

	return &sphinx.DecryptedError{
		SenderIdx: m.sourceIdx,
		Message:   m.message,
	}, m.err
}

var _ htlcNotifier = (*mockHTLCNotifier)(nil)

type mockHTLCNotifier struct{}

func (h *mockHTLCNotifier) NotifyForwardingEvent(key HtlcKey, info HtlcInfo,
	eventType HtlcEventType) {
}

func (h *mockHTLCNotifier) NotifyLinkFailEvent(key HtlcKey, info HtlcInfo,
	eventType HtlcEventType, linkErr *LinkError, incoming bool) {
}

func (h *mockHTLCNotifier) NotifyForwardingFailEvent(key HtlcKey,
	eventType HtlcEventType) {
}

func (h *mockHTLCNotifier) NotifySettleEvent(key HtlcKey, eventType HtlcEventType) {
}
