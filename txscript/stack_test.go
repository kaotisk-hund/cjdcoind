// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/txscript/scriptnum"
	"github.com/kaotisk-hund/cjdcoind/txscript/txscripterr"
)

// tstCheckScriptError ensures the type of the two passed errors are of the
// same type (either both nil or both of type Error) and their error codes
// match when not nil.
func tstCheckScriptError(gotErr, wantErr er.R) er.R {

	// Ensure the want error type is a script error.
	if wantErr != nil && !txscripterr.Err.Is(wantErr) {
		return er.Errorf("unexpected test error type %T", wantErr)
	}

	// Ensure the error code is of the expected type and the error
	// code matches the value specified in the test instance.
	if !er.FuzzyEquals(gotErr, wantErr) {
		return er.Errorf("wrong error - got %T (%[1]v), want %T",
			gotErr, wantErr)
	}
	return nil
}

// TestStack tests that all of the stack operations work as expected.
func TestStack(t *testing.T) {

	tests := []struct {
		name      string
		before    [][]byte
		operation func(*stack) er.R
		err       er.R
		after     [][]byte
	}{
		{
			"noop",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) er.R {
				return nil
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {5}},
		},
		{
			"peek underflow (byte)",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) er.R {
				_, err := s.PeekByteArray(5)
				return err
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"peek underflow (int)",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) er.R {
				_, err := s.PeekInt(5)
				return err
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"peek underflow (bool)",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) er.R {
				_, err := s.PeekBool(5)
				return err
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"pop",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) er.R {
				val, err := s.PopByteArray()
				if err != nil {
					return err
				}
				if !bytes.Equal(val, []byte{5}) {
					return er.New("not equal")
				}
				return err
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}},
		},
		{
			"pop everything",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) er.R {
				for i := 0; i < 5; i++ {
					_, err := s.PopByteArray()
					if err != nil {
						return err
					}
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"pop underflow",
			[][]byte{{1}, {2}, {3}, {4}, {5}},
			func(s *stack) er.R {
				for i := 0; i < 6; i++ {
					_, err := s.PopByteArray()
					if err != nil {
						return err
					}
				}
				return nil
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"pop bool",
			[][]byte{nil},
			func(s *stack) er.R {
				val, err := s.PopBool()
				if err != nil {
					return err
				}

				if val {
					return er.New("unexpected value")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"pop bool",
			[][]byte{{1}},
			func(s *stack) er.R {
				val, err := s.PopBool()
				if err != nil {
					return err
				}

				if !val {
					return er.New("unexpected value")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"pop bool",
			nil,
			func(s *stack) er.R {
				_, err := s.PopBool()
				return err
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"popInt 0",
			[][]byte{{0x0}},
			func(s *stack) er.R {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != 0 {
					return er.New("0 != 0 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt -0",
			[][]byte{{0x80}},
			func(s *stack) er.R {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != 0 {
					return er.New("-0 != 0 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt 1",
			[][]byte{{0x01}},
			func(s *stack) er.R {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != 1 {
					return er.New("1 != 1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt 1 leading 0",
			[][]byte{{0x01, 0x00, 0x00, 0x00}},
			func(s *stack) er.R {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != 1 {
					fmt.Printf("%v != %v\n", v, 1)
					return er.New("1 != 1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt -1",
			[][]byte{{0x81}},
			func(s *stack) er.R {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != -1 {
					return er.New("-1 != -1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"popInt -1 leading 0",
			[][]byte{{0x01, 0x00, 0x00, 0x80}},
			func(s *stack) er.R {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != -1 {
					fmt.Printf("%v != %v\n", v, -1)
					return er.New("-1 != -1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		// Triggers the multibyte case in asInt
		{
			"popInt -513",
			[][]byte{{0x1, 0x82}},
			func(s *stack) er.R {
				v, err := s.PopInt()
				if err != nil {
					return err
				}
				if v != -513 {
					fmt.Printf("%v != %v\n", v, -513)
					return er.New("1 != 1 on popInt")
				}
				return nil
			},
			nil,
			nil,
		},
		// Confirm that the asInt code doesn't modify the base data.
		{
			"peekint nomodify -1",
			[][]byte{{0x01, 0x00, 0x00, 0x80}},
			func(s *stack) er.R {
				v, err := s.PeekInt(0)
				if err != nil {
					return err
				}
				if v != -1 {
					fmt.Printf("%v != %v\n", v, -1)
					return er.New("-1 != -1 on popInt")
				}
				return nil
			},
			nil,
			[][]byte{{0x01, 0x00, 0x00, 0x80}},
		},
		{
			"PushInt 0",
			nil,
			func(s *stack) er.R {
				s.PushInt(scriptnum.ScriptNum(0))
				return nil
			},
			nil,
			[][]byte{{}},
		},
		{
			"PushInt 1",
			nil,
			func(s *stack) er.R {
				s.PushInt(scriptnum.ScriptNum(1))
				return nil
			},
			nil,
			[][]byte{{0x1}},
		},
		{
			"PushInt -1",
			nil,
			func(s *stack) er.R {
				s.PushInt(scriptnum.ScriptNum(-1))
				return nil
			},
			nil,
			[][]byte{{0x81}},
		},
		{
			"PushInt two bytes",
			nil,
			func(s *stack) er.R {
				s.PushInt(scriptnum.ScriptNum(256))
				return nil
			},
			nil,
			// little endian.. *sigh*
			[][]byte{{0x00, 0x01}},
		},
		{
			"PushInt leading zeros",
			nil,
			func(s *stack) er.R {
				// this will have the highbit set
				s.PushInt(scriptnum.ScriptNum(128))
				return nil
			},
			nil,
			[][]byte{{0x80, 0x00}},
		},
		{
			"dup",
			[][]byte{{1}},
			func(s *stack) er.R {
				return s.DupN(1)
			},
			nil,
			[][]byte{{1}, {1}},
		},
		{
			"dup2",
			[][]byte{{1}, {2}},
			func(s *stack) er.R {
				return s.DupN(2)
			},
			nil,
			[][]byte{{1}, {2}, {1}, {2}},
		},
		{
			"dup3",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) er.R {
				return s.DupN(3)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {1}, {2}, {3}},
		},
		{
			"dup0",
			[][]byte{{1}},
			func(s *stack) er.R {
				return s.DupN(0)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"dup-1",
			[][]byte{{1}},
			func(s *stack) er.R {
				return s.DupN(-1)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"dup too much",
			[][]byte{{1}},
			func(s *stack) er.R {
				return s.DupN(2)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"PushBool true",
			nil,
			func(s *stack) er.R {
				s.PushBool(true)

				return nil
			},
			nil,
			[][]byte{{1}},
		},
		{
			"PushBool false",
			nil,
			func(s *stack) er.R {
				s.PushBool(false)

				return nil
			},
			nil,
			[][]byte{nil},
		},
		{
			"PushBool PopBool",
			nil,
			func(s *stack) er.R {
				s.PushBool(true)
				val, err := s.PopBool()
				if err != nil {
					return err
				}
				if !val {
					return er.New("unexpected value")
				}

				return nil
			},
			nil,
			nil,
		},
		{
			"PushBool PopBool 2",
			nil,
			func(s *stack) er.R {
				s.PushBool(false)
				val, err := s.PopBool()
				if err != nil {
					return err
				}
				if val {
					return er.New("unexpected value")
				}

				return nil
			},
			nil,
			nil,
		},
		{
			"PushInt PopBool",
			nil,
			func(s *stack) er.R {
				s.PushInt(scriptnum.ScriptNum(1))
				val, err := s.PopBool()
				if err != nil {
					return err
				}
				if !val {
					return er.New("unexpected value")
				}

				return nil
			},
			nil,
			nil,
		},
		{
			"PushInt PopBool 2",
			nil,
			func(s *stack) er.R {
				s.PushInt(scriptnum.ScriptNum(0))
				val, err := s.PopBool()
				if err != nil {
					return err
				}
				if val {
					return er.New("unexpected value")
				}

				return nil
			},
			nil,
			nil,
		},
		{
			"Nip top",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) er.R {
				return s.NipN(0)
			},
			nil,
			[][]byte{{1}, {2}},
		},
		{
			"Nip middle",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) er.R {
				return s.NipN(1)
			},
			nil,
			[][]byte{{1}, {3}},
		},
		{
			"Nip low",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) er.R {
				return s.NipN(2)
			},
			nil,
			[][]byte{{2}, {3}},
		},
		{
			"Nip too much",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) er.R {
				// bite off more than we can chew
				return s.NipN(3)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			[][]byte{{2}, {3}},
		},
		{
			"keep on tucking",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) er.R {
				return s.Tuck()
			},
			nil,
			[][]byte{{1}, {3}, {2}, {3}},
		},
		{
			"a little tucked up",
			[][]byte{{1}}, // too few arguments for tuck
			func(s *stack) er.R {
				return s.Tuck()
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"all tucked up",
			nil, // too few arguments  for tuck
			func(s *stack) er.R {
				return s.Tuck()
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"drop 1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.DropN(1)
			},
			nil,
			[][]byte{{1}, {2}, {3}},
		},
		{
			"drop 2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.DropN(2)
			},
			nil,
			[][]byte{{1}, {2}},
		},
		{
			"drop 3",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.DropN(3)
			},
			nil,
			[][]byte{{1}},
		},
		{
			"drop 4",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.DropN(4)
			},
			nil,
			nil,
		},
		{
			"drop 4/5",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.DropN(5)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"drop invalid",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.DropN(0)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Rot1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.RotN(1)
			},
			nil,
			[][]byte{{1}, {3}, {4}, {2}},
		},
		{
			"Rot2",
			[][]byte{{1}, {2}, {3}, {4}, {5}, {6}},
			func(s *stack) er.R {
				return s.RotN(2)
			},
			nil,
			[][]byte{{3}, {4}, {5}, {6}, {1}, {2}},
		},
		{
			"Rot too little",
			[][]byte{{1}, {2}},
			func(s *stack) er.R {
				return s.RotN(1)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Rot0",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) er.R {
				return s.RotN(0)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Swap1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.SwapN(1)
			},
			nil,
			[][]byte{{1}, {2}, {4}, {3}},
		},
		{
			"Swap2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.SwapN(2)
			},
			nil,
			[][]byte{{3}, {4}, {1}, {2}},
		},
		{
			"Swap too little",
			[][]byte{{1}},
			func(s *stack) er.R {
				return s.SwapN(1)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Swap0",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) er.R {
				return s.SwapN(0)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Over1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.OverN(1)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {3}},
		},
		{
			"Over2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.OverN(2)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {1}, {2}},
		},
		{
			"Over too little",
			[][]byte{{1}},
			func(s *stack) er.R {
				return s.OverN(1)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Over0",
			[][]byte{{1}, {2}, {3}},
			func(s *stack) er.R {
				return s.OverN(0)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Pick1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.PickN(1)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {3}},
		},
		{
			"Pick2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.PickN(2)
			},
			nil,
			[][]byte{{1}, {2}, {3}, {4}, {2}},
		},
		{
			"Pick too little",
			[][]byte{{1}},
			func(s *stack) er.R {
				return s.PickN(1)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Roll1",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.RollN(1)
			},
			nil,
			[][]byte{{1}, {2}, {4}, {3}},
		},
		{
			"Roll2",
			[][]byte{{1}, {2}, {3}, {4}},
			func(s *stack) er.R {
				return s.RollN(2)
			},
			nil,
			[][]byte{{1}, {3}, {4}, {2}},
		},
		{
			"Roll too little",
			[][]byte{{1}},
			func(s *stack) er.R {
				return s.RollN(1)
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
		{
			"Peek bool",
			[][]byte{{1}},
			func(s *stack) er.R {
				// Peek bool is otherwise pretty well tested,
				// just check it works.
				val, err := s.PeekBool(0)
				if err != nil {
					return err
				}
				if !val {
					return er.New("invalid result")
				}
				return nil
			},
			nil,
			[][]byte{{1}},
		},
		{
			"Peek bool 2",
			[][]byte{nil},
			func(s *stack) er.R {
				// Peek bool is otherwise pretty well tested,
				// just check it works.
				val, err := s.PeekBool(0)
				if err != nil {
					return err
				}
				if val {
					return er.New("invalid result")
				}
				return nil
			},
			nil,
			[][]byte{nil},
		},
		{
			"Peek int",
			[][]byte{{1}},
			func(s *stack) er.R {
				// Peek int is otherwise pretty well tested,
				// just check it works.
				val, err := s.PeekInt(0)
				if err != nil {
					return err
				}
				if val != 1 {
					return er.New("invalid result")
				}
				return nil
			},
			nil,
			[][]byte{{1}},
		},
		{
			"Peek int 2",
			[][]byte{{0}},
			func(s *stack) er.R {
				// Peek int is otherwise pretty well tested,
				// just check it works.
				val, err := s.PeekInt(0)
				if err != nil {
					return err
				}
				if val != 0 {
					return er.New("invalid result")
				}
				return nil
			},
			nil,
			[][]byte{{0}},
		},
		{
			"pop int",
			nil,
			func(s *stack) er.R {
				s.PushInt(scriptnum.ScriptNum(1))
				// Peek int is otherwise pretty well tested,
				// just check it works.
				val, err := s.PopInt()
				if err != nil {
					return err
				}
				if val != 1 {
					return er.New("invalid result")
				}
				return nil
			},
			nil,
			nil,
		},
		{
			"pop empty",
			nil,
			func(s *stack) er.R {
				// Peek int is otherwise pretty well tested,
				// just check it works.
				_, err := s.PopInt()
				return err
			},
			txscripterr.ScriptError(txscripterr.ErrInvalidStackOperation, ""),
			nil,
		},
	}

	for _, test := range tests {
		// Setup the initial stack state and perform the test operation.
		s := stack{}
		for i := range test.before {
			s.PushByteArray(test.before[i])
		}
		err := test.operation(&s)

		// Ensure the error code is of the expected type and the error
		// code matches the value specified in the test instance.
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("%s: %v", test.name, e)
			continue
		}
		if err != nil {
			continue
		}

		// Ensure the resulting stack is the expected length.
		if int32(len(test.after)) != s.Depth() {
			t.Errorf("%s: stack depth doesn't match expected: %v "+
				"vs %v", test.name, len(test.after),
				s.Depth())
			continue
		}

		// Ensure all items of the resulting stack are the expected
		// values.
		for i := range test.after {
			val, err := s.PeekByteArray(s.Depth() - int32(i) - 1)
			if err != nil {
				t.Errorf("%s: can't peek %dth stack entry: %v",
					test.name, i, err)
				break
			}

			if !bytes.Equal(val, test.after[i]) {
				t.Errorf("%s: %dth stack entry doesn't match "+
					"expected: %v vs %v", test.name, i, val,
					test.after[i])
				break
			}
		}
	}
}
