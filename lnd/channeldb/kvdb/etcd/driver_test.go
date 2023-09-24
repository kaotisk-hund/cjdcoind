// +build kvdb_etcd

package etcd

import (
	"testing"

	"github.com/kaotisk-hund/cjdcoind/btcutil/util"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/walletdb"
	"github.com/stretchr/testify/require"
)

func TestOpenCreateFailure(t *testing.T) {
	t.Parallel()

	db, err := walletdb.Open(dbType)
	util.RequireErr(t, err)
	require.Nil(t, db)

	db, err = walletdb.Open(dbType, "wrong")
	util.RequireErr(t, err)
	require.Nil(t, db)

	db, err = walletdb.Create(dbType)
	util.RequireErr(t, err)
	require.Nil(t, db)

	db, err = walletdb.Create(dbType, "wrong")
	util.RequireErr(t, err)
	require.Nil(t, db)
}
