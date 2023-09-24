// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package scriptbuilder

import (
	"encoding/binary"
	"fmt"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/txscript/opcode"
	"github.com/kaotisk-hund/cjdcoind/txscript/params"
	"github.com/kaotisk-hund/cjdcoind/txscript/scriptnum"
	"github.com/kaotisk-hund/cjdcoind/txscript/txscripterr"
)

const (
	// DefaultScriptAlloc is the default size used for the backing array
	// for a script being built by the ScriptBuilder.  The array will
	// dynamically grow as needed, but this figure is intended to provide
	// enough space for vast majority of scripts without needing to grow the
	// backing array multiple times.
	DefaultScriptAlloc = 500
)

// ErrScriptNotCanonical identifies a non-canonical script.  The caller can use
// a type assertion to detect this error type.
var ErrScriptNotCanonical = txscripterr.Err.Code("ErrScriptNotCanonical")

// ScriptBuilder provides a facility for building custom scripts.  It allows
// you to push opcodes, ints, and data while respecting canonical encoding.  In
// general it does not ensure the script will execute correctly, however any
// data pushes which would exceed the maximum allowed script engine limits and
// are therefore guaranteed not to execute will not be pushed and will result in
// the Script function returning an error.
//
// For example, the following would build a 2-of-3 multisig script for usage in
// a pay-to-script-hash (although in this situation MultiSigScript() would be a
// better choice to generate the script):
// 	builder := scriptbuilder.NewScriptBuilder()
// 	builder.AddOp(opcode.OP_2).AddData(pubKey1).AddData(pubKey2)
// 	builder.AddData(pubKey3).AddOp(opcode.OP_3)
// 	builder.AddOp(opcode.OP_CHECKMULTISIG)
// 	script, err := builder.Script()
// 	if err != nil {
// 		// Handle the error.
// 		return
// 	}
// 	fmt.Printf("Final multi-sig script: %x\n", script)
type ScriptBuilder struct {
	ScriptInt []byte
	ErrInt    er.R
}

// AddOp pushes the passed opcode to the end of the script.  The script will not
// be modified if pushing the opcode would cause the script to exceed the
// maximum allowed script engine size.
func (b *ScriptBuilder) AddOp(opcode byte) *ScriptBuilder {
	if b.ErrInt != nil {
		return b
	}

	// Pushes that would cause the script to exceed the largest allowed
	// script size would result in a non-canonical script.
	if len(b.ScriptInt)+1 > params.MaxScriptSize {
		str := fmt.Sprintf("adding an opcode would exceed the maximum "+
			"allowed canonical script length of %d", params.MaxScriptSize)
		b.ErrInt = ErrScriptNotCanonical.New(str, nil)
		return b
	}

	b.ScriptInt = append(b.ScriptInt, opcode)
	return b
}

// AddOps pushes the passed opcodes to the end of the script.  The script will
// not be modified if pushing the opcodes would cause the script to exceed the
// maximum allowed script engine size.
func (b *ScriptBuilder) AddOps(opcodes []byte) *ScriptBuilder {
	if b.ErrInt != nil {
		return b
	}

	// Pushes that would cause the script to exceed the largest allowed
	// script size would result in a non-canonical script.
	if len(b.ScriptInt)+len(opcodes) > params.MaxScriptSize {
		str := fmt.Sprintf("adding opcodes would exceed the maximum "+
			"allowed canonical script length of %d", params.MaxScriptSize)
		b.ErrInt = ErrScriptNotCanonical.New(str, nil)
		return b
	}

	b.ScriptInt = append(b.ScriptInt, opcodes...)
	return b
}

// CanonicalDataSize returns the number of bytes the canonical encoding of the
// data will take.
func CanonicalDataSize(data []byte) int {
	dataLen := len(data)

	// When the data consists of a single number that can be represented
	// by one of the "small integer" opcodes, that opcode will be instead
	// of a data push opcode followed by the number.
	if dataLen == 0 {
		return 1
	} else if dataLen == 1 && data[0] <= 16 {
		return 1
	} else if dataLen == 1 && data[0] == 0x81 {
		return 1
	}

	if dataLen < opcode.OP_PUSHDATA1 {
		return 1 + dataLen
	} else if dataLen <= 0xff {
		return 2 + dataLen
	} else if dataLen <= 0xffff {
		return 3 + dataLen
	}

	return 5 + dataLen
}

// addData is the internal function that actually pushes the passed data to the
// end of the script.  It automatically chooses canonical opcodes depending on
// the length of the data.  A zero length buffer will lead to a push of empty
// data onto the stack (OP_0).  No data limits are enforced with this function.
func (b *ScriptBuilder) addData(data []byte) *ScriptBuilder {
	dataLen := len(data)

	// When the data consists of a single number that can be represented
	// by one of the "small integer" opcodes, use that opcode instead of
	// a data push opcode followed by the number.
	if dataLen == 0 || dataLen == 1 && data[0] == 0 {
		b.ScriptInt = append(b.ScriptInt, opcode.OP_0)
		return b
	} else if dataLen == 1 && data[0] <= 16 {
		b.ScriptInt = append(b.ScriptInt, (opcode.OP_1-1)+data[0])
		return b
	} else if dataLen == 1 && data[0] == 0x81 {
		b.ScriptInt = append(b.ScriptInt, byte(opcode.OP_1NEGATE))
		return b
	}

	// Use one of the OP_DATA_# opcodes if the length of the data is small
	// enough so the data push instruction is only a single byte.
	// Otherwise, choose the smallest possible OP_PUSHDATA# opcode that
	// can represent the length of the data.
	if dataLen < opcode.OP_PUSHDATA1 {
		b.ScriptInt = append(b.ScriptInt, byte((opcode.OP_DATA_1-1)+dataLen))
	} else if dataLen <= 0xff {
		b.ScriptInt = append(b.ScriptInt, opcode.OP_PUSHDATA1, byte(dataLen))
	} else if dataLen <= 0xffff {
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(dataLen))
		b.ScriptInt = append(b.ScriptInt, opcode.OP_PUSHDATA2)
		b.ScriptInt = append(b.ScriptInt, buf...)
	} else {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(dataLen))
		b.ScriptInt = append(b.ScriptInt, opcode.OP_PUSHDATA4)
		b.ScriptInt = append(b.ScriptInt, buf...)
	}

	// Append the actual data.
	b.ScriptInt = append(b.ScriptInt, data...)

	return b
}

// AddFullData should not typically be used by ordinary users as it does not
// include the checks which prevent data pushes larger than the maximum allowed
// sizes which leads to scripts that can't be executed.  This is provided for
// testing purposes such as regression tests where sizes are intentionally made
// larger than allowed.
//
// Use AddData instead.
func (b *ScriptBuilder) AddFullData(data []byte) *ScriptBuilder {
	if b.ErrInt != nil {
		return b
	}

	return b.addData(data)
}

// AddData pushes the passed data to the end of the script.  It automatically
// chooses canonical opcodes depending on the length of the data.  A zero length
// buffer will lead to a push of empty data onto the stack (OP_0) and any push
// of data greater than MaxScriptElementSize will not modify the script since
// that is not allowed by the script engine.  Also, the script will not be
// modified if pushing the data would cause the script to exceed the maximum
// allowed script engine size.
func (b *ScriptBuilder) AddData(data []byte) *ScriptBuilder {
	if b.ErrInt != nil {
		return b
	}

	// Pushes that would cause the script to exceed the largest allowed
	// script size would result in a non-canonical script.
	dataSize := CanonicalDataSize(data)
	if len(b.ScriptInt)+dataSize > params.MaxScriptSize {
		str := fmt.Sprintf("adding %d bytes of data would exceed the "+
			"maximum allowed canonical script length of %d",
			dataSize, params.MaxScriptSize)
		b.ErrInt = ErrScriptNotCanonical.New(str, nil)
		return b
	}

	// Pushes larger than the max script element size would result in a
	// script that is not canonical.
	dataLen := len(data)
	if dataLen > params.MaxScriptElementSize {
		str := fmt.Sprintf("adding a data element of %d bytes would "+
			"exceed the maximum allowed script element size of %d",
			dataLen, params.MaxScriptElementSize)
		b.ErrInt = ErrScriptNotCanonical.New(str, nil)
		return b
	}

	return b.addData(data)
}

// AddInt64 pushes the passed integer to the end of the script.  The script will
// not be modified if pushing the data would cause the script to exceed the
// maximum allowed script engine size.
func (b *ScriptBuilder) AddInt64(val int64) *ScriptBuilder {
	if b.ErrInt != nil {
		return b
	}

	// Pushes that would cause the script to exceed the largest allowed
	// script size would result in a non-canonical script.
	if len(b.ScriptInt)+1 > params.MaxScriptSize {
		str := fmt.Sprintf("adding an integer would exceed the "+
			"maximum allow canonical script length of %d",
			params.MaxScriptSize)
		b.ErrInt = ErrScriptNotCanonical.New(str, nil)
		return b
	}

	// Fast path for small integers and OP_1NEGATE.
	if val == 0 {
		b.ScriptInt = append(b.ScriptInt, opcode.OP_0)
		return b
	}
	if val == -1 || (val >= 1 && val <= 16) {
		b.ScriptInt = append(b.ScriptInt, byte((opcode.OP_1-1)+val))
		return b
	}

	return b.AddData(scriptnum.ScriptNum(val).Bytes())
}

// Reset resets the script so it has no content.
func (b *ScriptBuilder) Reset() *ScriptBuilder {
	b.ScriptInt = b.ScriptInt[0:0]
	b.ErrInt = nil
	return b
}

// Script returns the currently built script.  When any errors occurred while
// building the script, the script will be returned up the point of the first
// error along with the error.
func (b *ScriptBuilder) Script() ([]byte, er.R) {
	return b.ScriptInt, b.ErrInt
}

// NewScriptBuilder returns a new instance of a script builder.  See
// ScriptBuilder for details.
func NewScriptBuilder() *ScriptBuilder {
	return &ScriptBuilder{
		ScriptInt: make([]byte, 0, DefaultScriptAlloc),
	}
}
