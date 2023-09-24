// +build bitcoind
// +build notxindex

package lntest

import (
	"github.com/kaotisk-hund/cjdcoind/chaincfg"
)

// NewBackend starts a bitcoind node without the txindex enabled and returns a
// BitoindBackendConfig for that node.
func NewBackend(miner string, netParams *chaincfg.Params) (
	*BitcoindBackendConfig, func() error, er.R) {

	extraArgs := []string{
		"-debug",
		"-regtest",
		"-disablewallet",
	}

	return newBackend(miner, netParams, extraArgs)
}
