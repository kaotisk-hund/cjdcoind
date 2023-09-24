// +build goleveldbtests

package leveldb

import (
	"testing"

	"github.com/kaotisk-hund/cjdcoind/goleveldb/leveldb/testutil"
)

func TestLevelDB(t *testing.T) {
	testutil.RunSuite(t, "LevelDB Suite")
}
