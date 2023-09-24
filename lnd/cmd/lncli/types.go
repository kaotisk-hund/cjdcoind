package main

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/chaincfg/chainhash"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnrpc"
)

// OutPoint displays an outpoint string in the form "<txid>:<output-index>".
type OutPoint string

// NewOutPointFromProto formats the lnrpc.OutPoint into an OutPoint for display.
func NewOutPointFromProto(op *lnrpc.OutPoint) OutPoint {
	var hash chainhash.Hash
	copy(hash[:], op.TxidBytes)
	return OutPoint(fmt.Sprintf("%v:%d", hash, op.OutputIndex))
}

// NewProtoOutPoint parses an OutPoint into its corresponding lnrpc.OutPoint
// type.
func NewProtoOutPoint(op string) (*lnrpc.OutPoint, er.R) {
	parts := strings.Split(op, ":")
	if len(parts) != 2 {
		return nil, er.New("outpoint should be of the form txid:index")
	}
	txid := parts[0]
	if hex.DecodedLen(len(txid)) != chainhash.HashSize {
		return nil, er.Errorf("invalid hex-encoded txid %v", txid)
	}
	outputIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, er.Errorf("invalid output index: %v", err)
	}
	return &lnrpc.OutPoint{
		TxidStr:     txid,
		OutputIndex: uint32(outputIndex),
	}, nil
}

// Utxo displays information about an unspent output, including its address,
// amount, pkscript, and confirmations.
type Utxo struct {
	Type          lnrpc.AddressType `json:"address_type"`
	Address       string            `json:"address"`
	AmountSat     int64             `json:"amount_sat"`
	PkScript      string            `json:"pk_script"`
	OutPoint      OutPoint          `json:"outpoint"`
	Confirmations int64             `json:"confirmations"`
}

// NewUtxoFromProto creates a display Utxo from the Utxo proto. This filters out
// the raw txid bytes from the provided outpoint, which will otherwise be
// printed in base64.
func NewUtxoFromProto(utxo *lnrpc.Utxo) *Utxo {
	return &Utxo{
		Type:          utxo.AddressType,
		Address:       utxo.Address,
		AmountSat:     utxo.AmountSat,
		PkScript:      utxo.PkScript,
		OutPoint:      NewOutPointFromProto(utxo.Outpoint),
		Confirmations: utxo.Confirmations,
	}
}
