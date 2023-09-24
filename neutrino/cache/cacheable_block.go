package cache

import (
	"github.com/kaotisk-hund/cjdcoind/btcutil"
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
)

// CacheableBlock is a wrapper around the btcutil.Block type which provides a
// Size method used by the cache to target certain memory usage.
type CacheableBlock struct {
	*btcutil.Block
}

// Size returns size of this block in bytes.
func (c *CacheableBlock) Size() (uint64, er.R) {
	return uint64(c.Block.MsgBlock().SerializeSize()), nil
}
