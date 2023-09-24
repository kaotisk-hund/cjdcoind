// Copyright (c) 2014-2017 The btcsuite developers
// Copyright (c) 2015-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcclient

import (
	"bytes"
	"encoding/hex"

	jsoniter "github.com/json-iterator/go"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"

	"github.com/kaotisk-hund/cjdcoind/btcjson"
	"github.com/kaotisk-hund/cjdcoind/chaincfg/chainhash"
	"github.com/kaotisk-hund/cjdcoind/wire"
)

// FutureGetBlockResult is a future promise to deliver the result of a
// GetBlockAsync RPC invocation (or an applicable error).
type FutureGetBlockResult chan *response

// Receive waits for the response promised by the future and returns the raw
// block requested from the server given its hash.
func (r FutureGetBlockResult) Receive() (*wire.MsgBlock, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a string.
	var blockHex string
	err = er.E(jsoniter.Unmarshal(res, &blockHex))
	if err != nil {
		return nil, err
	}

	// Decode the serialized block hex to raw bytes.
	serializedBlock, errr := hex.DecodeString(blockHex)
	if errr != nil {
		return nil, er.E(errr)
	}

	// Deserialize the block and return it.
	var msgBlock wire.MsgBlock
	err = msgBlock.Deserialize(bytes.NewReader(serializedBlock))
	if err != nil {
		return nil, err
	}
	return &msgBlock, nil
}

// GetBlockAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetBlock for the blocking version and more details.
func (c *Client) GetBlockAsync(blockHash *chainhash.Hash) FutureGetBlockResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetBlockCmd(hash, btcjson.Bool(false), nil)
	return c.sendCmd(cmd)
}

// GetBlock returns a raw block from the server given its hash.
//
// See GetBlockVerbose to retrieve a data structure with information about the
// block instead.
func (c *Client) GetBlock(blockHash *chainhash.Hash) (*wire.MsgBlock, er.R) {
	return c.GetBlockAsync(blockHash).Receive()
}

// FutureGetBlockVerboseResult is a future promise to deliver the result of a
// GetBlockVerboseAsync RPC invocation (or an applicable error).
type FutureGetBlockVerboseResult chan *response

// Receive waits for the response promised by the future and returns the data
// structure from the server with information about the requested block.
func (r FutureGetBlockVerboseResult) Receive() (*btcjson.GetBlockVerboseResult, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal the raw result into a BlockResult.
	var blockResult btcjson.GetBlockVerboseResult
	err = er.E(jsoniter.Unmarshal(res, &blockResult))
	if err != nil {
		return nil, err
	}
	return &blockResult, nil
}

// GetBlockVerboseTxAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetBlockVerboseTx or the blocking version and more details.
func (c *Client) GetBlockVerboseTxAsync(blockHash *chainhash.Hash) FutureGetBlockVerboseResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetBlockCmd(hash, btcjson.Bool(true), btcjson.Bool(true))
	return c.sendCmd(cmd)
}

// GetBlockVerboseTx returns a data structure from the server with information
// about a block and its transactions given its hash.
//
// See GetBlockVerbose if only transaction hashes are preferred.
// See GetBlock to retrieve a raw block instead.
func (c *Client) GetBlockVerboseTx(blockHash *chainhash.Hash) (*btcjson.GetBlockVerboseResult, er.R) {
	return c.GetBlockVerboseTxAsync(blockHash).Receive()
}

// FutureGetBlockHashResult is a future promise to deliver the result of a
// GetBlockHashAsync RPC invocation (or an applicable error).
type FutureGetBlockHashResult chan *response

// Receive waits for the response promised by the future and returns the hash of
// the block in the best block chain at the given height.
func (r FutureGetBlockHashResult) Receive() (*chainhash.Hash, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal the result as a string-encoded sha.
	var txHashStr string
	err = er.E(jsoniter.Unmarshal(res, &txHashStr))
	if err != nil {
		return nil, err
	}
	return chainhash.NewHashFromStr(txHashStr)
}

// GetBlockHashAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetBlockHash for the blocking version and more details.
func (c *Client) GetBlockHashAsync(blockHeight int64) FutureGetBlockHashResult {
	cmd := btcjson.NewGetBlockHashCmd(blockHeight)
	return c.sendCmd(cmd)
}

// GetBlockHash returns the hash of the block in the best block chain at the
// given height.
func (c *Client) GetBlockHash(blockHeight int64) (*chainhash.Hash, er.R) {
	return c.GetBlockHashAsync(blockHeight).Receive()
}

// FutureGetBlockHeaderResult is a future promise to deliver the result of a
// GetBlockHeaderAsync RPC invocation (or an applicable error).
type FutureGetBlockHeaderResult chan *response

// Receive waits for the response promised by the future and returns the
// blockheader requested from the server given its hash.
func (r FutureGetBlockHeaderResult) Receive() (*wire.BlockHeader, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a string.
	var bhHex string
	err = er.E(jsoniter.Unmarshal(res, &bhHex))
	if err != nil {
		return nil, err
	}

	serializedBH, errr := hex.DecodeString(bhHex)
	if errr != nil {
		return nil, er.E(errr)
	}

	// Deserialize the blockheader and return it.
	var bh wire.BlockHeader
	err = bh.Deserialize(bytes.NewReader(serializedBH))
	if err != nil {
		return nil, err
	}

	return &bh, err
}

// GetBlockHeaderAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetBlockHeader for the blocking version and more details.
func (c *Client) GetBlockHeaderAsync(blockHash *chainhash.Hash) FutureGetBlockHeaderResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetBlockHeaderCmd(hash, btcjson.Bool(false))
	return c.sendCmd(cmd)
}

// GetBlockHeader returns the blockheader from the server given its hash.
//
// See GetBlockHeaderVerbose to retrieve a data structure with information about the
// block instead.
func (c *Client) GetBlockHeader(blockHash *chainhash.Hash) (*wire.BlockHeader, er.R) {
	return c.GetBlockHeaderAsync(blockHash).Receive()
}

// FutureGetRawMempoolResult is a future promise to deliver the result of a
// GetRawMempoolAsync RPC invocation (or an applicable error).
type FutureGetRawMempoolResult chan *response

// Receive waits for the response promised by the future and returns the hashes
// of all transactions in the memory pool.
func (r FutureGetRawMempoolResult) Receive() ([]*chainhash.Hash, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal the result as an array of strings.
	var txHashStrs []string
	err = er.E(jsoniter.Unmarshal(res, &txHashStrs))
	if err != nil {
		return nil, err
	}

	// Create a slice of ShaHash arrays from the string slice.
	txHashes := make([]*chainhash.Hash, 0, len(txHashStrs))
	for _, hashStr := range txHashStrs {
		txHash, err := chainhash.NewHashFromStr(hashStr)
		if err != nil {
			return nil, err
		}
		txHashes = append(txHashes, txHash)
	}

	return txHashes, nil
}

// GetRawMempoolAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetRawMempool for the blocking version and more details.
func (c *Client) GetRawMempoolAsync() FutureGetRawMempoolResult {
	cmd := btcjson.NewGetRawMempoolCmd(btcjson.Bool(false))
	return c.sendCmd(cmd)
}

// GetRawMempool returns the hashes of all transactions in the memory pool.
//
// See GetRawMempoolVerbose to retrieve data structures with information about
// the transactions instead.
func (c *Client) GetRawMempool() ([]*chainhash.Hash, er.R) {
	return c.GetRawMempoolAsync().Receive()
}

// FutureGetTxOutResult is a future promise to deliver the result of a
// GetTxOutAsync RPC invocation (or an applicable error).
type FutureGetTxOutResult chan *response

// Receive waits for the response promised by the future and returns a
// transaction given its hash.
func (r FutureGetTxOutResult) Receive() (*btcjson.GetTxOutResult, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// take care of the special case where the output has been spent already
	// it should return the string "null"
	if string(res) == "null" {
		return nil, nil
	}

	// Unmarshal result as an gettxout result object.
	var txOutInfo *btcjson.GetTxOutResult
	err = er.E(jsoniter.Unmarshal(res, &txOutInfo))
	if err != nil {
		return nil, err
	}

	return txOutInfo, nil
}

// GetTxOutAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetTxOut for the blocking version and more details.
func (c *Client) GetTxOutAsync(txHash *chainhash.Hash, index uint32, mempool bool) FutureGetTxOutResult {
	hash := ""
	if txHash != nil {
		hash = txHash.String()
	}

	cmd := btcjson.NewGetTxOutCmd(hash, index, &mempool)
	return c.sendCmd(cmd)
}

// FutureGetCFilterResult is a future promise to deliver the result of a
// GetCFilterAsync RPC invocation (or an applicable error).
type FutureGetCFilterResult chan *response

// Receive waits for the response promised by the future and returns the raw
// filter requested from the server given its block hash.
func (r FutureGetCFilterResult) Receive() (*wire.MsgCFilter, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a string.
	var filterHex string
	err = er.E(jsoniter.Unmarshal(res, &filterHex))
	if err != nil {
		return nil, err
	}

	// Decode the serialized cf hex to raw bytes.
	serializedFilter, errr := hex.DecodeString(filterHex)
	if errr != nil {
		return nil, er.E(errr)
	}

	// Assign the filter bytes to the correct field of the wire message.
	// We aren't going to set the block hash or extended flag, since we
	// don't actually get that back in the RPC response.
	var msgCFilter wire.MsgCFilter
	msgCFilter.Data = serializedFilter
	return &msgCFilter, nil
}

// GetCFilterAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetCFilter for the blocking version and more details.
func (c *Client) GetCFilterAsync(blockHash *chainhash.Hash,
	filterType wire.FilterType) FutureGetCFilterResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetCFilterCmd(hash, filterType)
	return c.sendCmd(cmd)
}

// GetCFilter returns a raw filter from the server given its block hash.
func (c *Client) GetCFilter(blockHash *chainhash.Hash,
	filterType wire.FilterType) (*wire.MsgCFilter, er.R) {
	return c.GetCFilterAsync(blockHash, filterType).Receive()
}

// FutureGetCFilterHeaderResult is a future promise to deliver the result of a
// GetCFilterHeaderAsync RPC invocation (or an applicable error).
type FutureGetCFilterHeaderResult chan *response

// Receive waits for the response promised by the future and returns the raw
// filter header requested from the server given its block hash.
func (r FutureGetCFilterHeaderResult) Receive() (*wire.MsgCFHeaders, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a string.
	var headerHex string
	err = er.E(jsoniter.Unmarshal(res, &headerHex))
	if err != nil {
		return nil, err
	}

	// Assign the decoded header into a hash
	headerHash, err := chainhash.NewHashFromStr(headerHex)
	if err != nil {
		return nil, err
	}

	// Assign the hash to a headers message and return it.
	msgCFHeaders := wire.MsgCFHeaders{PrevFilterHeader: *headerHash}
	return &msgCFHeaders, nil

}

// GetCFilterHeaderAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function
// on the returned instance.
//
// See GetCFilterHeader for the blocking version and more details.
func (c *Client) GetCFilterHeaderAsync(blockHash *chainhash.Hash,
	filterType wire.FilterType) FutureGetCFilterHeaderResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetCFilterHeaderCmd(hash, filterType)
	return c.sendCmd(cmd)
}

// GetCFilterHeader returns a raw filter header from the server given its block
// hash.
func (c *Client) GetCFilterHeader(blockHash *chainhash.Hash,
	filterType wire.FilterType) (*wire.MsgCFHeaders, er.R) {
	return c.GetCFilterHeaderAsync(blockHash, filterType).Receive()
}

// GetTxOut returns the transaction output info if it's unspent and
// nil, otherwise.
func (c *Client) GetTxOut(txHash *chainhash.Hash, index uint32, mempool bool) (*btcjson.GetTxOutResult, er.R) {
	return c.GetTxOutAsync(txHash, index, mempool).Receive()
}

// FutureGetBlockHeaderVerboseResult is a future promise to deliver the result of a
// GetBlockAsync RPC invocation (or an applicable error).
type FutureGetBlockHeaderVerboseResult chan *response

// Receive waits for the response promised by the future and returns the
// data structure of the blockheader requested from the server given its hash.
func (r FutureGetBlockHeaderVerboseResult) Receive() (*btcjson.GetBlockHeaderVerboseResult, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a string.
	var bh btcjson.GetBlockHeaderVerboseResult
	errr := jsoniter.Unmarshal(res, &bh)
	if errr != nil {
		return nil, er.E(errr)
	}

	return &bh, nil
}

// GetBlockHeaderVerboseAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetBlockHeader for the blocking version and more details.
func (c *Client) GetBlockHeaderVerboseAsync(blockHash *chainhash.Hash) FutureGetBlockHeaderVerboseResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}

	cmd := btcjson.NewGetBlockHeaderCmd(hash, btcjson.Bool(true))
	return c.sendCmd(cmd)
}

// GetBlockHeaderVerbose returns a data structure with information about the
// blockheader from the server given its hash.
//
// See GetBlockHeader to retrieve a blockheader instead.
func (c *Client) GetBlockHeaderVerbose(blockHash *chainhash.Hash) (*btcjson.GetBlockHeaderVerboseResult, er.R) {
	return c.GetBlockHeaderVerboseAsync(blockHash).Receive()
}

// FutureRescanBlocksResult is a future promise to deliver the result of a
// RescanBlocksAsync RPC invocation (or an applicable error).
//
// NOTE: This is a btcsuite extension ported from
// github.com/decred/dcrrpcclient.
type FutureRescanBlocksResult chan *response

// Receive waits for the response promised by the future and returns the
// discovered rescanblocks data.
//
// NOTE: This is a btcsuite extension ported from
// github.com/decred/dcrrpcclient.
func (r FutureRescanBlocksResult) Receive() ([]btcjson.RescannedBlock, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return nil, err
	}

	var rescanBlocksResult []btcjson.RescannedBlock
	err = er.E(jsoniter.Unmarshal(res, &rescanBlocksResult))
	if err != nil {
		return nil, err
	}

	return rescanBlocksResult, nil
}

// RescanBlocksAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See RescanBlocks for the blocking version and more details.
//
// NOTE: This is a btcsuite extension ported from
// github.com/decred/dcrrpcclient.
func (c *Client) RescanBlocksAsync(blockHashes []chainhash.Hash) FutureRescanBlocksResult {
	strBlockHashes := make([]string, len(blockHashes))
	for i := range blockHashes {
		strBlockHashes[i] = blockHashes[i].String()
	}

	cmd := btcjson.NewRescanBlocksCmd(strBlockHashes)
	return c.sendCmd(cmd)
}

// RescanBlocks rescans the blocks identified by blockHashes, in order, using
// the client's loaded transaction filter.  The blocks do not need to be on the
// main chain, but they do need to be adjacent to each other.
//
// NOTE: This is a btcsuite extension ported from
// github.com/decred/dcrrpcclient.
func (c *Client) RescanBlocks(blockHashes []chainhash.Hash) ([]btcjson.RescannedBlock, er.R) {
	return c.RescanBlocksAsync(blockHashes).Receive()
}

// GetBlockVerboseAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See GetBlockVerbose for the blocking version and more details.
func (c *Client) GetBlockVerboseAsync(blockHash *chainhash.Hash) FutureGetBlockVerboseResult {
	hash := ""
	if blockHash != nil {
		hash = blockHash.String()
	}
	// From the bitcoin-cli getblock documentation:
	// "If verbosity is 1, returns an Object with information about block ."
	cmd := btcjson.NewGetBlockCmd(hash, btcjson.Bool(true), btcjson.Bool(false))
	return c.sendCmd(cmd)
}

// GetBlockVerbose returns a data structure from the server with information
// about a block given its hash.
//
// See GetBlockVerboseTx to retrieve transaction data structures as well.
// See GetBlock to retrieve a raw block instead.
func (c *Client) GetBlockVerbose(blockHash *chainhash.Hash) (*btcjson.GetBlockVerboseResult, er.R) {
	return c.GetBlockVerboseAsync(blockHash).Receive()
}

// FutureEstimateFeeResult is a future promise to deliver the result of a
// EstimateFeeAsync RPC invocation (or an applicable error).
type FutureEstimateFeeResult chan *response

// Receive waits for the response promised by the future and returns the info
// provided by the server.
func (r FutureEstimateFeeResult) Receive() (float64, er.R) {
	res, err := receiveFuture(r)
	if err != nil {
		return -1, err
	}

	// Unmarshal result as a getinfo result object.
	var fee float64
	err = er.E(jsoniter.Unmarshal(res, &fee))
	if err != nil {
		return -1, err
	}

	return fee, nil
}

// EstimateFeeAsync returns an instance of a type that can be used to get the result
// of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See EstimateFee for the blocking version and more details.
func (c *Client) EstimateFeeAsync(numBlocks int64) FutureEstimateFeeResult {
	cmd := btcjson.NewEstimateFeeCmd(numBlocks)
	return c.sendCmd(cmd)
}

// EstimateFee provides an estimated fee  in bitcoins per kilobyte.
func (c *Client) EstimateFee(numBlocks int64) (float64, er.R) {
	return c.EstimateFeeAsync(numBlocks).Receive()
}
