// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson_test

import (
	"github.com/json-iterator/go"
	"testing"

	"github.com/kaotisk-hund/cjdcoind/btcjson"
)

// TestChainSvrCustomResults ensures any results that have custom marshalling
// work as inteded.
// and unmarshal code of results are as expected.
func TestChainSvrCustomResults(t *testing.T) {

	tests := []struct {
		name     string
		result   interface{}
		expected string
	}{
		{
			name: "custom vin marshal with coinbase",
			result: &btcjson.Vin{
				Coinbase: "021234",
				Sequence: 4294967295,
			},
			expected: `{"coinbase":"021234","sequence":4294967295}`,
		},
		{
			name: "custom vin marshal without coinbase",
			result: &btcjson.Vin{
				Txid: "123",
				Vout: 1,
				ScriptSig: &btcjson.ScriptSig{
					Asm: "0",
					Hex: "00",
				},
				Sequence: 4294967295,
			},
			expected: `{"txid":"123","vout":1,"scriptSig":{"asm":"0","hex":"00"},"sequence":4294967295}`,
		},
		{
			name: "custom vinprevout marshal with coinbase",
			result: &btcjson.VinPrevOut{
				Coinbase: "021234",
				Sequence: 4294967295,
			},
			expected: `{"coinbase":"021234","sequence":4294967295}`,
		},
		{
			name: "custom vinprevout marshal without coinbase",
			result: &btcjson.VinPrevOut{
				Txid: "123",
				Vout: 1,
				ScriptSig: &btcjson.ScriptSig{
					Asm: "0",
					Hex: "00",
				},
				PrevOut: &btcjson.PrevOut{
					Address:    "addr1",
					ValueCoins: 0,
					Svalue:     "0",
				},
				Sequence: 4294967295,
			},
			expected: `{"txid":"123","vout":1,"scriptSig":{"asm":"0","hex":"00"},"prevOut":{"address":"addr1","value":0,"svalue":"0"},"sequence":4294967295}`,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		marshalled, err := jsoniter.Marshal(test.result)
		if err != nil {
			t.Errorf("Test #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}
		if string(marshalled) != test.expected {
			t.Errorf("Test #%d (%s) unexpected marhsalled data - "+
				"got %s, want %s", i, test.name, marshalled,
				test.expected)
			continue
		}
	}
}
