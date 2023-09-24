package input

import (
	"encoding/binary"
	"io"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/btcutil/util"
	"github.com/kaotisk-hund/cjdcoind/wire"
)

// writeTxOut serializes a wire.TxOut struct into the passed io.Writer stream.
func writeTxOut(w io.Writer, txo *wire.TxOut) er.R {
	var scratch [8]byte

	binary.BigEndian.PutUint64(scratch[:], uint64(txo.Value))
	if _, err := util.Write(w, scratch[:]); err != nil {
		return err
	}

	if err := wire.WriteVarBytes(w, 0, txo.PkScript); err != nil {
		return err
	}

	return nil
}

// readTxOut deserializes a wire.TxOut struct from the passed io.Reader stream.
func readTxOut(r io.Reader, txo *wire.TxOut) er.R {
	var scratch [8]byte

	if _, err := util.ReadFull(r, scratch[:]); err != nil {
		return err
	}
	value := int64(binary.BigEndian.Uint64(scratch[:]))

	pkScript, err := wire.ReadVarBytes(r, 0, 80, "pkScript")
	if err != nil {
		return err
	}

	*txo = wire.TxOut{
		Value:    value,
		PkScript: pkScript,
	}

	return nil
}
