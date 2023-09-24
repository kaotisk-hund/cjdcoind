// +build rpctest

package btcwallet

import (
	"github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/snacl"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/waddrmgr"
)

func init() {
	// Instruct waddrmgr to use the cranked down scrypt parameters when
	// creating new wallet encryption keys. This will speed up the itests
	// considerably.
	fastScrypt := waddrmgr.FastScryptOptions
	keyGen := func(passphrase *[]byte, config *waddrmgr.ScryptOptions) (
		*snacl.SecretKey, er.R) {

		return snacl.NewSecretKey(
			passphrase, fastScrypt.N, fastScrypt.R, fastScrypt.P,
		)
	}
	waddrmgr.SetSecretKeyGen(keyGen)
}
