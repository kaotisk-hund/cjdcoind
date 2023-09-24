package watchtower

import (
	"net"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/watchtower/lookout"
	"github.com/kaotisk-hund/cjdcoind/lnd/watchtower/wtserver"
)

// DB abstracts the persistent functionality required to run the watchtower
// daemon. It composes the database interfaces required by the lookout and
// wtserver subsystems.
type DB interface {
	lookout.DB
	wtserver.DB
}

// AddressNormalizer is a function signature that allows the tower to resolve
// TCP addresses on clear or onion networks.
type AddressNormalizer func(addrs []string, defaultPort string,
	resolver func(string, string) (*net.TCPAddr, er.R)) ([]net.Addr, er.R)
