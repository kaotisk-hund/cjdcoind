// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/wire/protocol"

	"github.com/davecgh/go-spew/spew"
)

// TestFilterCLearLatest tests the MsgFilterClear API against the latest
// protocol version.
func TestFilterClearLatest(t *testing.T) {
	pver := protocol.ProtocolVersion

	msg := NewMsgFilterClear()

	// Ensure the command is expected value.
	wantCmd := "filterclear"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgFilterClear: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	wantPayload := uint32(0)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}
}

// TestFilterClearCrossProtocol tests the MsgFilterClear API when encoding with
// the latest protocol version and decoding with BIP0031Version.
func TestFilterClearCrossProtocol(t *testing.T) {
	msg := NewMsgFilterClear()

	// Encode with latest protocol version.
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, protocol.ProtocolVersion, LatestEncoding)
	if err != nil {
		t.Errorf("encode of MsgFilterClear failed %v err <%v>", msg, err)
	}

	// Decode with old protocol version.
	var readmsg MsgFilterClear
	err = readmsg.BtcDecode(&buf, protocol.BIP0031Version, LatestEncoding)
	if err == nil {
		t.Errorf("decode of MsgFilterClear succeeded when it "+
			"shouldn't have %v", msg)
	}
}

// TestFilterClearWire tests the MsgFilterClear wire encode and decode for
// various protocol versions.
func TestFilterClearWire(t *testing.T) {
	msgFilterClear := NewMsgFilterClear()
	msgFilterClearEncoded := []byte{}

	tests := []struct {
		in   *MsgFilterClear // Message to encode
		out  *MsgFilterClear // Expected decoded message
		buf  []byte          // Wire encoding
		pver uint32          // Protocol version for wire encoding
		enc  MessageEncoding // Message encoding format
	}{
		// Latest protocol version.
		{
			msgFilterClear,
			msgFilterClear,
			msgFilterClearEncoded,
			protocol.ProtocolVersion,
			BaseEncoding,
		},

		// Protocol version BIP0037Version + 1.
		{
			msgFilterClear,
			msgFilterClear,
			msgFilterClearEncoded,
			protocol.BIP0037Version + 1,
			BaseEncoding,
		},

		// Protocol version BIP0037Version.
		{
			msgFilterClear,
			msgFilterClear,
			msgFilterClearEncoded,
			protocol.BIP0037Version,
			BaseEncoding,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to wire format.
		var buf bytes.Buffer
		err := test.in.BtcEncode(&buf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcEncode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("BtcEncode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode the message from wire format.
		var msg MsgFilterClear
		rbuf := bytes.NewReader(test.buf)
		err = msg.BtcDecode(rbuf, test.pver, test.enc)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}

// TestFilterClearWireErrors performs negative tests against wire encode and
// decode of MsgFilterClear to confirm error paths work correctly.
func TestFilterClearWireErrors(t *testing.T) {
	pverNoFilterClear := protocol.BIP0037Version - 1
	wireErr := MessageError.Default()

	baseFilterClear := NewMsgFilterClear()
	baseFilterClearEncoded := []byte{}

	tests := []struct {
		in       *MsgFilterClear // Value to encode
		buf      []byte          // Wire encoding
		pver     uint32          // Protocol version for wire encoding
		enc      MessageEncoding // Message encoding format
		max      int             // Max size of fixed buffer to induce errors
		writeErr er.R            // Expected write error
		readErr  er.R            // Expected read error
	}{
		// Force error due to unsupported protocol version.
		{
			baseFilterClear, baseFilterClearEncoded,
			pverNoFilterClear, BaseEncoding, 4, wireErr, wireErr,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		w := newFixedWriter(test.max)
		err := test.in.BtcEncode(w, test.pver, test.enc)
		if !er.FuzzyEquals(err, test.writeErr) {
			t.Errorf("BtcEncode #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		// For errors which are not of type MessageError, check them for
		// equality.
		if !MessageError.Is(err) {
			if err != test.writeErr {
				t.Errorf("BtcEncode #%d wrong error got: %v, "+
					"want: %v", i, err, test.writeErr)
				continue
			}
		}

		// Decode from wire format.
		var msg MsgFilterClear
		r := newFixedReader(test.max, test.buf)
		err = msg.BtcDecode(r, test.pver, test.enc)
		if !er.FuzzyEquals(err, test.readErr) {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}

		// For errors which are not of type MessageError, check them for
		// equality.
		if !MessageError.Is(err) {
			if err != test.readErr {
				t.Errorf("BtcDecode #%d wrong error got: %v, "+
					"want: %v", i, err, test.readErr)
				continue
			}
		}

	}
}
