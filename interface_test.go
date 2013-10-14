// Copyright (c) 2013 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcdb_test

import (
	"github.com/conformal/btcdb"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
	"github.com/davecgh/go-spew/spew"
	"reflect"
	"testing"
)

// testContext is used to store context information about a running test which
// is passed into helper functions.
type testContext struct {
	t           *testing.T
	dbType      string
	db          btcdb.Db
	blockHeight int64
	blockHash   *btcwire.ShaHash
	block       *btcutil.Block
}

// testInsertBlock ensures InsertBlock conforms to the interface contract.
func testInsertBlock(tc *testContext) bool {
	// The block must insert without any errors.
	newHeight, err := tc.db.InsertBlock(tc.block)
	if err != nil {
		tc.t.Errorf("InsertBlock (%s): failed to insert block %v "+
			"err %v", tc.dbType, tc.blockHeight, err)
		return false
	}

	// The returned height must be the expected value.
	if newHeight != tc.blockHeight {
		tc.t.Errorf("InsertBlock (%s): height mismatch got: %v, "+
			"want: %v", tc.dbType, newHeight, tc.blockHeight)
		return false
	}

	return true
}

// testExistsSha ensures ExistsSha conforms to the interface contract.
func testExistsSha(tc *testContext) bool {
	// The block must exist in the database.
	if exists := tc.db.ExistsSha(tc.blockHash); !exists {
		tc.t.Errorf("ExistsSha (%s): block %v does not exist",
			tc.dbType, tc.blockHash)
		return false
	}

	return true
}

// testFetchBlockBySha ensures FetchBlockBySha conforms to the interface
// contract.
func testFetchBlockBySha(tc *testContext) bool {
	// The block must be fetchable by its hash without any errors.
	blockFromDb, err := tc.db.FetchBlockBySha(tc.blockHash)
	if err != nil {
		tc.t.Errorf("FetchBlockBySha (%s): %v", tc.dbType, err)
		return false
	}

	// The block fetched from the database must give back the same MsgBlock
	// and raw bytes that were stored.
	if !reflect.DeepEqual(tc.block.MsgBlock(), blockFromDb.MsgBlock()) {
		tc.t.Errorf("FetchBlockBySha (%s): block from database "+
			"does not match stored block\ngot: %v\n"+
			"want: %v", tc.dbType,
			spew.Sdump(blockFromDb.MsgBlock()),
			spew.Sdump(tc.block.MsgBlock()))
		return false
	}
	blockBytes, err := tc.block.Bytes()
	if err != nil {
		tc.t.Errorf("block.Bytes: %v", err)
		return false
	}
	blockFromDbBytes, err := blockFromDb.Bytes()
	if err != nil {
		tc.t.Errorf("blockFromDb.Bytes: %v", err)
		return false
	}
	if !reflect.DeepEqual(blockBytes, blockFromDbBytes) {
		tc.t.Errorf("FetchBlockBySha (%s): block bytes from "+
			"database do not match stored block bytes\n"+
			"got: %v\nwant: %v", tc.dbType,
			spew.Sdump(blockFromDbBytes), spew.Sdump(blockBytes))
		return false
	}

	return true
}

// testFetchBlockShaByHeight ensures FetchBlockShaByHeight conforms to the
// interface contract.
func testFetchBlockShaByHeight(tc *testContext) bool {
	// The hash returned for the block by its height must be the expected
	// value.
	hashFromDb, err := tc.db.FetchBlockShaByHeight(tc.blockHeight)
	if err != nil {
		tc.t.Errorf("FetchBlockShaByHeight (%s): %v", tc.dbType, err)
		return false
	}
	if !hashFromDb.IsEqual(tc.blockHash) {
		tc.t.Errorf("FetchBlockShaByHeight (%s): returned hash "+
			"does  not match expected value - got: %v, "+
			"want: %v", tc.dbType, hashFromDb, tc.blockHash)
		return false
	}

	// Invalid heights must error and return a nil hash.
	tests := []int64{-1, tc.blockHeight + 1, tc.blockHeight + 2}
	for i, wantHeight := range tests {
		hashFromDb, err := tc.db.FetchBlockShaByHeight(wantHeight)
		if err == nil {
			tc.t.Errorf("FetchBlockShaByHeight #%d (%s): did not "+
				"return error on invalid index: %d - got: %v, "+
				"want: non-nil", i, tc.dbType, wantHeight, err)
			return false
		}
		if hashFromDb != nil {
			tc.t.Errorf("FetchBlockShaByHeight #%d (%s): returned "+
				"hash is not nil on invalid index: %d - got: "+
				"%v, want: nil", i, tc.dbType, wantHeight, err)
			return false
		}
	}

	return true
}

// testExistsTxSha ensures ExistsTxSha conforms to the interface contract.
func testExistsTxSha(tc *testContext) bool {
	txHashes, err := tc.block.TxShas()
	if err != nil {
		tc.t.Errorf("block.TxShas: %v", err)
		return false
	}

	for i := range txHashes {
		// The transaction must exist in the database.
		txHash := txHashes[i]
		if exists := tc.db.ExistsTxSha(txHash); !exists {
			tc.t.Errorf("ExistsTxSha (%s): tx %v does not exist",
				tc.dbType, txHash)
			return false
		}
	}

	return true
}

// testFetchTxBySha ensures FetchTxBySha conforms to the interface contract.
func testFetchTxBySha(tc *testContext) bool {
	txHashes, err := tc.block.TxShas()
	if err != nil {
		tc.t.Errorf("block.TxShas: %v", err)
		return false
	}

	for i, tx := range tc.block.MsgBlock().Transactions {
		txHash := txHashes[i]
		txReplyList, err := tc.db.FetchTxBySha(txHash)
		if err != nil {
			tc.t.Errorf("FetchTxBySha (%s): %v", tc.dbType, err)
			return false
		}
		if len(txReplyList) == 0 {
			tc.t.Errorf("FetchTxBySha (%s): tx %v did not "+
				"return reply data", tc.dbType, txHash)
			return false
		}
		txFromDb := txReplyList[len(txReplyList)-1].Tx
		if !reflect.DeepEqual(tx, txFromDb) {
			tc.t.Errorf("FetchTxBySha (%s): tx %v from "+
				"database does not match stored tx\n"+
				"got: %v\nwant: %v", tc.dbType, txHash,
				spew.Sdump(txFromDb), spew.Sdump(tx))
			return false
		}
	}

	return true
}

// testFetchTxByShaList ensures FetchTxByShaList conforms to the interface
// contract.
func testFetchTxByShaList(tc *testContext) bool {
	txHashes, err := tc.block.TxShas()
	if err != nil {
		tc.t.Errorf("block.TxShas: %v", err)
		return false
	}

	txReplyList := tc.db.FetchTxByShaList(txHashes)
	if len(txReplyList) != len(txHashes) {
		tc.t.Errorf("FetchTxByShaList (%s): tx reply list for "+
			"block #%d (%v) does not match expected length "+
			"- got: %v, want: %v", tc.dbType, tc.blockHeight,
			tc.blockHash, len(txReplyList), len(txHashes))
		return false
	}
	for i, tx := range tc.block.MsgBlock().Transactions {
		txHash := txHashes[i]
		txD := txReplyList[i]

		// The transaction hash in the reply must be the expected value.
		if !txD.Sha.IsEqual(txHash) {
			tc.t.Errorf("FetchTxByShaList (%s): tx #%d hash "+
				"does not match expected - got %v, "+
				"want %v", tc.dbType, i, txD.Sha, txHashes[i])
			return false
		}

		// The reply must not indicate any errors.
		if txD.Err != nil {
			tc.t.Errorf("FetchTxByShaList (%s): tx #%d (%v) "+
				"returned unexpected error - got %v, "+
				"want nil", tc.dbType, i, txD.Sha, txD.Err)
			return false
		}

		// The transaction in the reply fetched from the database must
		// be the same MsgTx that was stored.
		if !reflect.DeepEqual(tx, txD.Tx) {
			tc.t.Errorf("FetchTxByShaList (%s): tx #%d (%v) from "+
				"database does not match stored tx\n"+
				"got: %v\nwant: %v", tc.dbType, i, txHash,
				spew.Sdump(txD.Tx), spew.Sdump(tx))
			return false
		}

		// The block hash in the reply from the database must be the
		// expected value.
		if txD.BlkSha == nil {
			tc.t.Errorf("FetchTxByShaList (%s): tx #%d (%v) "+
				"returned nil block hash", tc.dbType, i, txD.Sha)
			return false
		}
		if !txD.BlkSha.IsEqual(tc.blockHash) {
			tc.t.Errorf("FetchTxByShaList (%s): tx #%d (%v) "+
				"returned unexpected block hash - got %v, "+
				"want %v", tc.dbType, i, txD.Sha, txD.BlkSha,
				tc.blockHash)
			return false
		}

		// The block height in the reply from the database must be the
		// expected value.
		if txD.Height != tc.blockHeight {
			tc.t.Errorf("FetchTxByShaList (%s): tx #%d (%v) "+
				"returned unexpected block height - got %v, "+
				"want %v", tc.dbType, i, txD.Sha, txD.Height,
				tc.blockHeight)
			return false
		}

		// The spend data in the reply from the database must not
		// indicate any of the transactions that were just inserted are
		// spent.
		if txD.TxSpent == nil {
			tc.t.Errorf("FetchTxByShaList (%s): tx #%d (%v) "+
				"returned nil spend data", tc.dbType, i, txD.Sha)
			return false
		}
		noSpends := make([]bool, len(tx.TxOut))
		if !reflect.DeepEqual(txD.TxSpent, noSpends) {
			tc.t.Errorf("FetchTxByShaList (%s): tx #%d (%v) "+
				"returned unexpected spend data - got %v, "+
				"want %v", tc.dbType, i, txD.Sha, txD.TxSpent,
				noSpends)
			return false
		}
	}

	return true
}

// testFetchUnSpentTxByShaList ensures FetchUnSpentTxByShaList conforms to the
// interface contract.
func testFetchUnSpentTxByShaList(tc *testContext) bool {
	txHashes, err := tc.block.TxShas()
	if err != nil {
		tc.t.Errorf("block.TxShas: %v", err)
		return false
	}

	txReplyList := tc.db.FetchUnSpentTxByShaList(txHashes)
	if len(txReplyList) != len(txHashes) {
		tc.t.Errorf("FetchUnSpentTxByShaList (%s): tx reply list for "+
			"block #%d (%v) does not match expected length "+
			"- got: %v, want: %v", tc.dbType, tc.blockHeight,
			tc.blockHash, len(txReplyList), len(txHashes))
		return false
	}
	for i, tx := range tc.block.MsgBlock().Transactions {
		txHash := txHashes[i]
		txD := txReplyList[i]

		// The transaction hash in the reply must be the expected value.
		if !txD.Sha.IsEqual(txHash) {
			tc.t.Errorf("FetchUnSpentTxByShaList (%s): tx #%d hash "+
				"does not match expected - got %v, "+
				"want %v", tc.dbType, i, txD.Sha, txHashes[i])
			return false
		}

		// The reply must not indicate any errors.
		if txD.Err != nil {
			tc.t.Errorf("FetchUnSpentTxByShaList (%s): tx #%d (%v) "+
				"returned unexpected error - got %v, "+
				"want nil", tc.dbType, i, txD.Sha, txD.Err)
			return false
		}

		// The transaction in the reply fetched from the database must
		// be the same MsgTx that was stored.
		if !reflect.DeepEqual(tx, txD.Tx) {
			tc.t.Errorf("FetchUnSpentTxByShaList (%s): tx #%d (%v) "+
				"from database does not match stored tx\n"+
				"got: %v\nwant: %v", tc.dbType, i, txHash,
				spew.Sdump(txD.Tx), spew.Sdump(tx))
			return false
		}

		// The block hash in the reply from the database must be the
		// expected value.
		if txD.BlkSha == nil {
			tc.t.Errorf("FetchUnSpentTxByShaList (%s): tx #%d (%v) "+
				"returned nil block hash", tc.dbType, i, txD.Sha)
			return false
		}
		if !txD.BlkSha.IsEqual(tc.blockHash) {
			tc.t.Errorf("FetchUnSpentTxByShaList (%s): tx #%d (%v) "+
				"returned unexpected block hash - got %v, "+
				"want %v", tc.dbType, i, txD.Sha, txD.BlkSha,
				tc.blockHash)
			return false
		}

		// The block height in the reply from the database must be the
		// expected value.
		if txD.Height != tc.blockHeight {
			tc.t.Errorf("FetchUnSpentTxByShaList (%s): tx #%d (%v) "+
				"returned unexpected block height - got %v, "+
				"want %v", tc.dbType, i, txD.Sha, txD.Height,
				tc.blockHeight)
			return false
		}

		// The spend data in the reply from the database must not
		// indicate any of the transactions that were just inserted are
		// spent.
		if txD.TxSpent == nil {
			tc.t.Errorf("FetchUnSpentTxByShaList (%s): tx #%d (%v) "+
				"returned nil spend data", tc.dbType, i, txD.Sha)
			return false
		}
		noSpends := make([]bool, len(tx.TxOut))
		if !reflect.DeepEqual(txD.TxSpent, noSpends) {
			tc.t.Errorf("FetchUnSpentTxByShaList (%s): tx #%d (%v) "+
				"returned unexpected spend data - got %v, "+
				"want %v", tc.dbType, i, txD.Sha, txD.TxSpent,
				noSpends)
			return false
		}
	}

	return true
}

// testInterface tests performs tests for the various interfaces of btcdb which
// require state in the database for the given database type.
func testInterface(t *testing.T, dbType string) {
	db, teardown, err := setupDB(dbType, "interface")
	if err != nil {
		t.Errorf("Failed to create test database (%s) %v", dbType, err)
		return
	}
	defer teardown()

	// Load up a bunch of test blocks.
	blocks, err := loadBlocks(t)
	if err != nil {
		t.Errorf("Unable to load blocks from test data %v: %v",
			blockDataFile, err)
		return
	}

	// Create a test context to pass around.
	context := testContext{t: t, dbType: dbType, db: db}

	t.Logf("Loaded %d blocks for testing %s", len(blocks), dbType)
	for height := int64(1); height < int64(len(blocks)); height++ {
		// Get the appropriate block and hash and update the test
		// context accordingly.
		block := blocks[height]
		blockHash, err := block.Sha()
		if err != nil {
			t.Errorf("block.Sha: %v", err)
			return
		}
		context.blockHeight = height
		context.blockHash = blockHash
		context.block = block

		// The block must insert without any errors and return the
		// expected height.
		if !testInsertBlock(&context) {
			return
		}

		// The block must now exist in the database.
		if !testExistsSha(&context) {
			return
		}

		// Loading the block back from the database must give back
		// the same MsgBlock and raw bytes that were stored.
		if !testFetchBlockBySha(&context) {
			return
		}

		// The hash returned for the block by its height must be the
		// expected value.
		if !testFetchBlockShaByHeight(&context) {
			return
		}

		// All of the transactions in the block must now exist in the
		// database.
		if !testExistsTxSha(&context) {
			return
		}

		// Loading all of the transactions in the block back from the
		// database must give back the same MsgTx that was stored.
		if !testFetchTxBySha(&context) {
			return
		}

		// All of the transactions in the block must be fetchable via
		// FetchTxByShaList and all of the list replies must have the
		// expected values.
		if !testFetchTxByShaList(&context) {
			return
		}

		// All of the transactions in the block must be fetchable via
		// FetchUnSpentTxByShaList and all of the list replies must have
		// the expected values.
		if !testFetchUnSpentTxByShaList(&context) {
			return
		}
	}

	// TODO(davec): Need to figure out how to handle the special checks
	// required for the duplicate transactions allowed by blocks 91842 and
	// 91880 on the main network due to the old miner + Satoshi client bug.

	// TODO(davec): Add tests for the following functions:
	/*
	   Close()
	   DropAfterBlockBySha(*btcwire.ShaHash) (err error)
	   - ExistsSha(sha *btcwire.ShaHash) (exists bool)
	   - FetchBlockBySha(sha *btcwire.ShaHash) (blk *btcutil.Block, err error)
	   - FetchBlockShaByHeight(height int64) (sha *btcwire.ShaHash, err error)
	   FetchHeightRange(startHeight, endHeight int64) (rshalist []btcwire.ShaHash, err error)
	   - ExistsTxSha(sha *btcwire.ShaHash) (exists bool)
	   - FetchTxBySha(txsha *btcwire.ShaHash) ([]*TxListReply, error)
	   - FetchTxByShaList(txShaList []*btcwire.ShaHash) []*TxListReply
	   - FetchUnSpentTxByShaList(txShaList []*btcwire.ShaHash) []*TxListReply
	   - InsertBlock(block *btcutil.Block) (height int64, err error)
	   InvalidateBlockCache()
	   InvalidateCache()
	   InvalidateTxCache()
	   NewIterateBlocks() (pbi BlockIterator, err error)
	   NewestSha() (sha *btcwire.ShaHash, height int64, err error)
	   RollbackClose()
	   SetDBInsertMode(InsertMode)
	   Sync()
	*/
}
