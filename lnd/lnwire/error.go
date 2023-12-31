package lnwire

import (
	"fmt"
	"io"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
)

// FundingError represents a set of errors that can be encountered and sent
// during the funding workflow.
var FundingError = er.NewErrorType("lnwire.FundingError")

var (
	// ErrMaxPendingChannels is returned by remote peer when the number of
	// active pending channels exceeds their maximum policy limit.
	ErrMaxPendingChannels = FundingError.CodeWithDetail("ErrMaxPendingChannels",
		"Number of pending channels exceed maximum")

	// ErrSynchronizingChain is returned by a remote peer that receives a
	// channel update or a funding request while their still syncing to the
	// latest state of the blockchain.
	ErrSynchronizingChain = FundingError.CodeWithDetail("ErrSynchronizingChain",
		"Synchronizing blockchain")

	// ErrChanTooLarge is returned by a remote peer that receives a
	// FundingOpen request for a channel that is above their current
	// soft-limit.
	ErrChanTooLarge = FundingError.CodeWithDetail("ErrChanTooLarge",
		"channel too large")
)

// ErrorData is a set of bytes associated with a particular sent error. A
// receiving node SHOULD only print out data verbatim if the string is composed
// solely of printable ASCII characters. For reference, the printable character
// set includes byte values 32 through 127 inclusive.
type ErrorData []byte

// Error represents a generic error bound to an exact channel. The message
// format is purposefully general in order to allow expression of a wide array
// of possible errors. Each Error message is directed at a particular open
// channel referenced by ChannelPoint.
type Error struct {
	// ChanID references the active channel in which the error occurred
	// within. If the ChanID is all zeros, then this error applies to the
	// entire established connection.
	ChanID ChannelID

	// Data is the attached error data that describes the exact failure
	// which caused the error message to be sent.
	Data ErrorData
}

// NewError creates a new Error message.
func NewError() *Error {
	return &Error{}
}

// A compile time check to ensure Error implements the lnwire.Message
// interface.
var _ Message = (*Error)(nil)

// Error returns the string representation to Error.
//
// NOTE: Satisfies the error interface.
func (c *Error) Error() string {
	errMsg := "non-ascii data"
	if isASCII(c.Data) {
		errMsg = string(c.Data)
	}

	return fmt.Sprintf("chan_id=%v, err=%v", c.ChanID, errMsg)
}

// Decode deserializes a serialized Error message stored in the passed
// io.Reader observing the specified protocol version.
//
// This is part of the lnwire.Message interface.
func (c *Error) Decode(r io.Reader, pver uint32) er.R {
	return ReadElements(r,
		&c.ChanID,
		&c.Data,
	)
}

// Encode serializes the target Error into the passed io.Writer observing the
// protocol version specified.
//
// This is part of the lnwire.Message interface.
func (c *Error) Encode(w io.Writer, pver uint32) er.R {
	return WriteElements(w,
		c.ChanID,
		c.Data,
	)
}

// MsgType returns the integer uniquely identifying an Error message on the
// wire.
//
// This is part of the lnwire.Message interface.
func (c *Error) MsgType() MessageType {
	return MsgError
}

// MaxPayloadLength returns the maximum allowed payload size for an Error
// complete message observing the specified protocol version.
//
// This is part of the lnwire.Message interface.
func (c *Error) MaxPayloadLength(uint32) uint32 {
	// 32 + 2 + 65501
	return 65535
}

// isASCII is a helper method that checks whether all bytes in `data` would be
// printable ASCII characters if interpreted as a string.
func isASCII(data []byte) bool {
	for _, c := range data {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}
