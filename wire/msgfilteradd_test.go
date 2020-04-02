// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"io"
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/wire/protocol"
)

// TestFilterAddLatest tests the MsgFilterAdd API against the latest protocol
// version.
func TestFilterAddLatest(t *testing.T) {
	enc := BaseEncoding
	pver := protocol.ProtocolVersion

	data := []byte{0x01, 0x02}
	msg := NewMsgFilterAdd(data)

	// Ensure the command is expected value.
	wantCmd := "filteradd"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgFilterAdd: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	wantPayload := uint32(523)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	// Test encode with latest protocol version.
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, pver, enc)
	if err != nil {
		t.Errorf("encode of MsgFilterAdd failed %v err <%v>", msg, err)
	}

	// Test decode with latest protocol version.
	var readmsg MsgFilterAdd
	err = readmsg.BtcDecode(&buf, pver, enc)
	if err != nil {
		t.Errorf("decode of MsgFilterAdd failed [%v] err <%v>", buf, err)
	}
}

// TestFilterAddCrossProtocol tests the MsgFilterAdd API when encoding with the
// latest protocol version and decoding with BIP0031Version.
func TestFilterAddCrossProtocol(t *testing.T) {
	data := []byte{0x01, 0x02}
	msg := NewMsgFilterAdd(data)
	if !bytes.Equal(msg.Data, data) {
		t.Errorf("should get same data back out")
	}

	// Encode with latest protocol version.
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, protocol.ProtocolVersion, LatestEncoding)
	if err != nil {
		t.Errorf("encode of MsgFilterAdd failed %v err <%v>", msg, err)
	}

	// Decode with old protocol version.
	var readmsg MsgFilterAdd
	err = readmsg.BtcDecode(&buf, protocol.BIP0031Version, LatestEncoding)
	if err == nil {
		t.Errorf("decode of MsgFilterAdd succeeded when it shouldn't "+
			"have %v", msg)
	}

	// Since one of the protocol versions doesn't support the filteradd
	// message, make sure the data didn't get encoded and decoded back out.
	if bytes.Equal(msg.Data, readmsg.Data) {
		t.Error("should not get same data for cross protocol")
	}

}

// TestFilterAddMaxDataSize tests the MsgFilterAdd API maximum data size.
func TestFilterAddMaxDataSize(t *testing.T) {
	data := bytes.Repeat([]byte{0xff}, 521)
	msg := NewMsgFilterAdd(data)

	// Encode with latest protocol version.
	var buf bytes.Buffer
	err := msg.BtcEncode(&buf, protocol.ProtocolVersion, LatestEncoding)
	if err == nil {
		t.Errorf("encode of MsgFilterAdd succeeded when it shouldn't "+
			"have %v", msg)
	}

	// Decode with latest protocol version.
	readbuf := bytes.NewReader(data)
	err = msg.BtcDecode(readbuf, protocol.ProtocolVersion, LatestEncoding)
	if err == nil {
		t.Errorf("decode of MsgFilterAdd succeeded when it shouldn't "+
			"have %v", msg)
	}
}

// TestFilterAddWireErrors performs negative tests against wire encode and decode
// of MsgFilterAdd to confirm error paths work correctly.
func TestFilterAddWireErrors(t *testing.T) {
	pver := protocol.ProtocolVersion
	pverNoFilterAdd := protocol.BIP0037Version - 1
	wireErr := MessageError.Default()

	baseData := []byte{0x01, 0x02, 0x03, 0x04}
	baseFilterAdd := NewMsgFilterAdd(baseData)
	baseFilterAddEncoded := append([]byte{0x04}, baseData...)

	tests := []struct {
		in       *MsgFilterAdd   // Value to encode
		buf      []byte          // Wire encoding
		pver     uint32          // Protocol version for wire encoding
		enc      MessageEncoding // Message encoding format
		max      int             // Max size of fixed buffer to induce errors
		writeErr er.R            // Expected write error
		readErr  er.R            // Expected read error
	}{
		// Latest protocol version with intentional read/write errors.
		// Force error in data size.
		{
			baseFilterAdd, baseFilterAddEncoded, pver, BaseEncoding, 0,
			er.E(io.ErrShortWrite), er.E(io.EOF),
		},
		// Force error in data.
		{
			baseFilterAdd, baseFilterAddEncoded, pver, BaseEncoding, 1,
			er.E(io.ErrShortWrite), er.E(io.EOF),
		},
		// Force error due to unsupported protocol version.
		{
			baseFilterAdd, baseFilterAddEncoded, pverNoFilterAdd, BaseEncoding, 5,
			wireErr, wireErr,
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

		// Decode from wire format.
		var msg MsgFilterAdd
		r := newFixedReader(test.max, test.buf)
		err = msg.BtcDecode(r, test.pver, test.enc)
		if !er.FuzzyEquals(err, test.readErr) {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}
