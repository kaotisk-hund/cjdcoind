package verrpc

import (
	"fmt"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/lnrpc"
)

func init() {
	subServer := &lnrpc.SubServerDriver{
		SubServerName: subServerName,
		New: func(c lnrpc.SubServerConfigDispatcher) (lnrpc.SubServer,
			lnrpc.MacaroonPerms, er.R) {

			return &Server{}, macPermissions, nil
		},
	}

	// We'll register ourselves as a sub-RPC server within the global lnrpc
	// package namespace.
	if err := lnrpc.RegisterSubServer(subServer); err != nil {
		panic(fmt.Sprintf("failed to register sub server driver '%s': %v",
			subServerName, err))
	}
}
