package memdb

import (
	"testing"

	"github.com/kaotisk-hund/cjdcoind/goleveldb/leveldb/testutil"
)

func TestMemDB(t *testing.T) {
	testutil.RunSuite(t, "MemDB Suite")
}
