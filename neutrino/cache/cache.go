package cache

import (
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
)

var Err er.ErrorType = er.NewErrorType("cache.Err")

var (
	// ErrElementNotFound is returned when element isn't found in the cache.
	ErrElementNotFound = Err.CodeWithDetail("ErrElementNotFound",
		"unable to find element")
)

// Cache represents a generic cache.
type Cache interface {
	// Put stores the given (key,value) pair, replacing existing value if
	// key already exists. The return value indicates whether items had to
	// be evicted to make room for the new element.
	Put(key interface{}, value Value) (bool, er.R)

	// Get returns the value for a given key.
	Get(key interface{}) (Value, er.R)

	// Len returns number of elements in the cache.
	Len() int
}

// Value represents a value stored in the Cache.
type Value interface {
	// Size determines how big this entry would be in the cache. For
	// example, for a filter, it could be the size of the filter in bytes.
	Size() (uint64, er.R)
}
