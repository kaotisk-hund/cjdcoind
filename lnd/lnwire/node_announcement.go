package lnwire

import (
	"bytes"
	"image/color"
	"io"
	"io/ioutil"
	"net"
	"unicode/utf8"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
)

var Err = er.NewErrorType("lnd.lnwire")

// ErrUnknownAddrType is an error returned if we encounter an unknown address type
// when parsing addresses.
var ErrUnknownAddrType = Err.CodeWithDetail("ErrUnknownAddrType",
	"unknown address type")

var ErrInvalidNodeAlias = Err.CodeWithDetail("ErrInvalidNodeAlias",
	"node alias has non-utf8 characters")

// NodeAlias a hex encoded UTF-8 string that may be displayed as an alternative
// to the node's ID. Notice that aliases are not unique and may be freely
// chosen by the node operators.
type NodeAlias [32]byte

// NewNodeAlias creates a new instance of a NodeAlias. Verification is
// performed on the passed string to ensure it meets the alias requirements.
func NewNodeAlias(s string) (NodeAlias, er.R) {
	var n NodeAlias

	if len(s) > 32 {
		return n, er.Errorf("alias too large: max is %v, got %v", 32,
			len(s))
	}

	if !utf8.ValidString(s) {
		return n, ErrInvalidNodeAlias.Default()
	}

	copy(n[:], []byte(s))
	return n, nil
}

// String returns a utf8 string representation of the alias bytes.
func (n NodeAlias) String() string {
	// Trim trailing zero-bytes for presentation
	return string(bytes.Trim(n[:], "\x00"))
}

// NodeAnnouncement message is used to announce the presence of a Lightning
// node and also to signal that the node is accepting incoming connections.
// Each NodeAnnouncement authenticating the advertised information within the
// announcement via a signature using the advertised node pubkey.
type NodeAnnouncement struct {
	// Signature is used to prove the ownership of node id.
	Signature Sig

	// Features is the list of protocol features this node supports.
	Features *RawFeatureVector

	// Timestamp allows ordering in the case of multiple announcements.
	Timestamp uint32

	// NodeID is a public key which is used as node identification.
	NodeID [33]byte

	// RGBColor is used to customize their node's appearance in maps and
	// graphs
	RGBColor color.RGBA

	// Alias is used to customize their node's appearance in maps and
	// graphs
	Alias NodeAlias

	// Address includes two specification fields: 'ipv6' and 'port' on
	// which the node is accepting incoming connections.
	Addresses []net.Addr

	// ExtraOpaqueData is the set of data that was appended to this
	// message, some of which we may not actually know how to iterate or
	// parse. By holding onto this data, we ensure that we're able to
	// properly validate the set of signatures that cover these new fields,
	// and ensure we're able to make upgrades to the network in a forwards
	// compatible manner.
	ExtraOpaqueData []byte
}

// A compile time check to ensure NodeAnnouncement implements the
// lnwire.Message interface.
var _ Message = (*NodeAnnouncement)(nil)

// Decode deserializes a serialized NodeAnnouncement stored in the passed
// io.Reader observing the specified protocol version.
//
// This is part of the lnwire.Message interface.
func (a *NodeAnnouncement) Decode(r io.Reader, pver uint32) er.R {
	err := ReadElements(r,
		&a.Signature,
		&a.Features,
		&a.Timestamp,
		&a.NodeID,
		&a.RGBColor,
		&a.Alias,
		&a.Addresses,
	)
	if err != nil {
		return err
	}

	// Now that we've read out all the fields that we explicitly know of,
	// we'll collect the remainder into the ExtraOpaqueData field. If there
	// aren't any bytes, then we'll snip off the slice to avoid carrying
	// around excess capacity.
	var errr error
	a.ExtraOpaqueData, errr = ioutil.ReadAll(r)
	if errr != nil {
		return er.E(errr)
	}
	if len(a.ExtraOpaqueData) == 0 {
		a.ExtraOpaqueData = nil
	}

	return nil
}

// Encode serializes the target NodeAnnouncement into the passed io.Writer
// observing the protocol version specified.
//
func (a *NodeAnnouncement) Encode(w io.Writer, pver uint32) er.R {
	return WriteElements(w,
		a.Signature,
		a.Features,
		a.Timestamp,
		a.NodeID,
		a.RGBColor,
		a.Alias,
		a.Addresses,
		a.ExtraOpaqueData,
	)
}

// MsgType returns the integer uniquely identifying this message type on the
// wire.
//
// This is part of the lnwire.Message interface.
func (a *NodeAnnouncement) MsgType() MessageType {
	return MsgNodeAnnouncement
}

// MaxPayloadLength returns the maximum allowed payload size for this message
// observing the specified protocol version.
//
// This is part of the lnwire.Message interface.
func (a *NodeAnnouncement) MaxPayloadLength(pver uint32) uint32 {
	return 65533
}

// DataToSign returns the part of the message that should be signed.
func (a *NodeAnnouncement) DataToSign() ([]byte, er.R) {

	// We should not include the signatures itself.
	var w bytes.Buffer
	err := WriteElements(&w,
		a.Features,
		a.Timestamp,
		a.NodeID,
		a.RGBColor,
		a.Alias[:],
		a.Addresses,
		a.ExtraOpaqueData,
	)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}
