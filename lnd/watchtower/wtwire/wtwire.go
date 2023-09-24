package wtwire

import (
	"encoding/binary"
	"io"

	"github.com/kaotisk-hund/cjdcoind/btcec"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/btcutil/util"
	"github.com/kaotisk-hund/cjdcoind/chaincfg/chainhash"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwallet/chainfee"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnwire"
	"github.com/kaotisk-hund/cjdcoind/lnd/watchtower/blob"
	"github.com/kaotisk-hund/cjdcoind/wire"
)

// WriteElement is a one-stop shop to write the big endian representation of
// any element which is to be serialized for the wire protocol. The passed
// io.Writer should be backed by an appropriately sized byte slice, or be able
// to dynamically expand to accommodate additional data.
func WriteElement(w io.Writer, element interface{}) er.R {
	switch e := element.(type) {
	case uint8:
		var b [1]byte
		b[0] = e
		if _, err := util.Write(w, b[:]); err != nil {
			return err
		}

	case uint16:
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], e)
		if _, err := util.Write(w, b[:]); err != nil {
			return err
		}

	case blob.Type:
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(e))
		if _, err := util.Write(w, b[:]); err != nil {
			return err
		}

	case uint32:
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], e)
		if _, err := util.Write(w, b[:]); err != nil {
			return err
		}

	case uint64:
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], e)
		if _, err := util.Write(w, b[:]); err != nil {
			return err
		}

	case [16]byte:
		if _, err := util.Write(w, e[:]); err != nil {
			return err
		}

	case [32]byte:
		if _, err := util.Write(w, e[:]); err != nil {
			return err
		}

	case [33]byte:
		if _, err := util.Write(w, e[:]); err != nil {
			return err
		}

	case []byte:
		if err := wire.WriteVarBytes(w, 0, e); err != nil {
			return err
		}

	case chainfee.SatPerKWeight:
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], uint64(e))
		if _, err := util.Write(w, b[:]); err != nil {
			return err
		}

	case ErrorCode:
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(e))
		if _, err := util.Write(w, b[:]); err != nil {
			return err
		}

	case chainhash.Hash:
		if _, err := util.Write(w, e[:]); err != nil {
			return err
		}

	case *lnwire.RawFeatureVector:
		if e == nil {
			return er.Errorf("cannot write nil feature vector")
		}

		if err := e.Encode(w); err != nil {
			return err
		}

	case *btcec.PublicKey:
		if e == nil {
			return er.Errorf("cannot write nil pubkey")
		}

		var b [33]byte
		serializedPubkey := e.SerializeCompressed()
		copy(b[:], serializedPubkey)
		if _, err := util.Write(w, b[:]); err != nil {
			return err
		}

	default:
		return er.Errorf("Unknown type in WriteElement: %T", e)
	}

	return nil
}

// WriteElements is writes each element in the elements slice to the passed
// io.Writer using WriteElement.
func WriteElements(w io.Writer, elements ...interface{}) er.R {
	for _, element := range elements {
		err := WriteElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadElement is a one-stop utility function to deserialize any datastructure
// encoded using the serialization format of lnwire.
func ReadElement(r io.Reader, element interface{}) er.R {
	switch e := element.(type) {
	case *uint8:
		var b [1]uint8
		if _, err := r.Read(b[:]); err != nil {
			return er.E(err)
		}
		*e = b[0]

	case *uint16:
		var b [2]byte
		if _, err := util.ReadFull(r, b[:]); err != nil {
			return err
		}
		*e = binary.BigEndian.Uint16(b[:])

	case *blob.Type:
		var b [2]byte
		if _, err := util.ReadFull(r, b[:]); err != nil {
			return err
		}
		*e = blob.Type(binary.BigEndian.Uint16(b[:]))

	case *uint32:
		var b [4]byte
		if _, err := util.ReadFull(r, b[:]); err != nil {
			return err
		}
		*e = binary.BigEndian.Uint32(b[:])

	case *uint64:
		var b [8]byte
		if _, err := util.ReadFull(r, b[:]); err != nil {
			return err
		}
		*e = binary.BigEndian.Uint64(b[:])

	case *[16]byte:
		if _, err := util.ReadFull(r, e[:]); err != nil {
			return err
		}

	case *[32]byte:
		if _, err := util.ReadFull(r, e[:]); err != nil {
			return err
		}

	case *[33]byte:
		if _, err := util.ReadFull(r, e[:]); err != nil {
			return err
		}

	case *[]byte:
		bytes, err := wire.ReadVarBytes(r, 0, 66000, "[]byte")
		if err != nil {
			return err
		}
		*e = bytes

	case *chainfee.SatPerKWeight:
		var b [8]byte
		if _, err := util.ReadFull(r, b[:]); err != nil {
			return err
		}
		*e = chainfee.SatPerKWeight(binary.BigEndian.Uint64(b[:]))

	case *ErrorCode:
		var b [2]byte
		if _, err := util.ReadFull(r, b[:]); err != nil {
			return err
		}
		*e = ErrorCode(binary.BigEndian.Uint16(b[:]))

	case *chainhash.Hash:
		if _, err := util.ReadFull(r, e[:]); err != nil {
			return err
		}

	case **lnwire.RawFeatureVector:
		f := lnwire.NewRawFeatureVector()
		err := f.Decode(r)
		if err != nil {
			return err
		}

		*e = f

	case **btcec.PublicKey:
		var b [btcec.PubKeyBytesLenCompressed]byte
		if _, err := util.ReadFull(r, b[:]); err != nil {
			return err
		}

		pubKey, err := btcec.ParsePubKey(b[:], btcec.S256())
		if err != nil {
			return err
		}
		*e = pubKey

	default:
		return er.Errorf("Unknown type in ReadElement: %T", e)
	}

	return nil
}

// ReadElements deserializes a variable number of elements into the passed
// io.Reader, with each element being deserialized according to the ReadElement
// function.
func ReadElements(r io.Reader, elements ...interface{}) er.R {
	for _, element := range elements {
		err := ReadElement(r, element)
		if err != nil {
			return err
		}
	}
	return nil
}
