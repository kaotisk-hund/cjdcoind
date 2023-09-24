package mempool

import (
	"os"
	"testing"

	"github.com/kaotisk-hund/cjdcoind/chaincfg/globalcfg"
)

func TestMain(m *testing.M) {
	globalcfg.SelectConfig(globalcfg.BitcoinDefaults())
	os.Exit(m.Run())
}
