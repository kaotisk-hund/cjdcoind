package table

import (
	"testing"

	"github.com/kaotisk-hund/cjdcoind/goleveldb/leveldb/testutil"
)

func TestTable(t *testing.T) {
	testutil.RunSuite(t, "Table Suite")
}
