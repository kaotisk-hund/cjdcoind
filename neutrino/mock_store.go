package neutrino

import (
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"

	"github.com/kaotisk-hund/cjdcoind/blockchain"
	"github.com/kaotisk-hund/cjdcoind/chaincfg/chainhash"
	"github.com/kaotisk-hund/cjdcoind/neutrino/headerfs"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/waddrmgr"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/walletdb"
	"github.com/kaotisk-hund/cjdcoind/wire"
)

// mockBlockHeaderStore is an implementation of the BlockHeaderStore backed by
// a simple map.
type mockBlockHeaderStore struct {
	headers map[chainhash.Hash]wire.BlockHeader
	heights map[uint32]wire.BlockHeader
}

// A compile-time check to ensure the mockBlockHeaderStore adheres to the
// BlockHeaderStore interface.
var _ headerfs.BlockHeaderStore = (*mockBlockHeaderStore)(nil)

// NewMockBlockHeaderStore returns a version of the BlockHeaderStore that's
// backed by an in-memory map. This instance is meant to be used by callers
// outside the package to unit test components that require a BlockHeaderStore
// interface.
func newMockBlockHeaderStore() *mockBlockHeaderStore {
	return &mockBlockHeaderStore{
		headers: make(map[chainhash.Hash]wire.BlockHeader),
		heights: make(map[uint32]wire.BlockHeader),
	}
}

func (m *mockBlockHeaderStore) ChainTip() (*wire.BlockHeader,
	uint32, er.R) {
	return nil, 0, nil
}
func (m *mockBlockHeaderStore) ChainTip1(tx walletdb.ReadTx) (*wire.BlockHeader,
	uint32, er.R) {
	return nil, 0, nil
}
func (m *mockBlockHeaderStore) LatestBlockLocator() (
	blockchain.BlockLocator, er.R) {
	return nil, nil
}

func (m *mockBlockHeaderStore) FetchHeaderByHeight(height uint32) (
	*wire.BlockHeader, er.R) {

	if header, ok := m.heights[height]; ok {
		return &header, nil
	}

	return nil, headerfs.ErrHeightNotFound.Default()
}

func (m *mockBlockHeaderStore) FetchHeaderByHeight1(tx walletdb.ReadTx, height uint32) (
	*wire.BlockHeader, er.R) {
	return m.FetchHeaderByHeight(height)
}

func (m *mockBlockHeaderStore) FetchHeaderAncestors(uint32,
	*chainhash.Hash) ([]wire.BlockHeader, uint32, er.R) {

	return nil, 0, nil
}
func (m *mockBlockHeaderStore) HeightFromHash(*chainhash.Hash) (uint32, er.R) {
	return 0, nil

}
func (m *mockBlockHeaderStore) RollbackLastBlock(tx walletdb.ReadWriteTx) (*waddrmgr.BlockStamp,
	er.R) {
	return nil, nil
}

func (m *mockBlockHeaderStore) FetchHeader(h *chainhash.Hash) (
	*wire.BlockHeader, uint32, er.R) {
	if header, ok := m.headers[*h]; ok {
		return &header, 0, nil
	}
	return nil, 0, er.Errorf("not found")
}
func (m *mockBlockHeaderStore) FetchHeader1(tx walletdb.ReadTx, h *chainhash.Hash) (
	*wire.BlockHeader, uint32, er.R) {
	if header, ok := m.headers[*h]; ok {
		return &header, 0, nil
	}
	return nil, 0, er.Errorf("not found")
}

func (m *mockBlockHeaderStore) WriteHeaders(tx walletdb.ReadWriteTx, headers ...headerfs.BlockHeader) er.R {
	for _, h := range headers {
		m.headers[h.BlockHash()] = *h.BlockHeader
	}

	return nil
}
