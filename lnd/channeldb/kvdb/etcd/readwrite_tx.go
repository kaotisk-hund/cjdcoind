// +build kvdb_etcd

package etcd

import (
	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/walletdb"
)

// readWriteTx holds a reference to the STM transaction.
type readWriteTx struct {
	// stm is the reference to the parent STM.
	stm STM

	// rootBucketID holds the sha256 hash of the root bucket id, which is used
	// for key space spearation.
	rootBucketID [bucketIDLength]byte

	// active is true if the transaction hasn't been committed yet.
	active bool
}

// newReadWriteTx creates an rw transaction with the passed STM.
func newReadWriteTx(stm STM, prefix string) *readWriteTx {
	return &readWriteTx{
		stm:          stm,
		active:       true,
		rootBucketID: makeBucketID([]byte(prefix)),
	}
}

// rooBucket is a helper function to return the always present
// pseudo root bucket.
func rootBucket(tx *readWriteTx) *readWriteBucket {
	return newReadWriteBucket(tx, tx.rootBucketID[:], tx.rootBucketID[:])
}

// ReadBucket opens the root bucket for read only access.  If the bucket
// described by the key does not exist, nil is returned.
func (tx *readWriteTx) ReadBucket(key []byte) walletdb.ReadBucket {
	return rootBucket(tx).NestedReadWriteBucket(key)
}

// Rollback closes the transaction, discarding changes (if any) if the
// database was modified by a write transaction.
func (tx *readWriteTx) Rollback() er.R {
	// If the transaction has been closed roolback will fail.
	if !tx.active {
		return walletdb.ErrTxClosed.Default()
	}

	// Rollback the STM and set the tx to inactive.
	tx.stm.Rollback()
	tx.active = false

	return nil
}

// ReadWriteBucket opens the root bucket for read/write access.  If the
// bucket described by the key does not exist, nil is returned.
func (tx *readWriteTx) ReadWriteBucket(key []byte) walletdb.ReadWriteBucket {
	return rootBucket(tx).NestedReadWriteBucket(key)
}

// CreateTopLevelBucket creates the top level bucket for a key if it
// does not exist.  The newly-created bucket it returned.
func (tx *readWriteTx) CreateTopLevelBucket(key []byte) (walletdb.ReadWriteBucket, er.R) {
	return rootBucket(tx).CreateBucketIfNotExists(key)
}

// DeleteTopLevelBucket deletes the top level bucket for a key.  This
// errors if the bucket can not be found or the key keys a single value
// instead of a bucket.
func (tx *readWriteTx) DeleteTopLevelBucket(key []byte) er.R {
	return rootBucket(tx).DeleteNestedBucket(key)
}

// Commit commits the transaction if not already committed. Will return
// error if the underlying STM fails.
func (tx *readWriteTx) Commit() er.R {
	// Commit will fail if the transaction is already committed.
	if !tx.active {
		return walletdb.ErrTxClosed.Default()
	}

	// Try committing the transaction.
	if err := tx.stm.Commit(); err != nil {
		return err
	}

	// Mark the transaction as not active after commit.
	tx.active = false

	return nil
}

// OnCommit sets the commit callback (overriding if already set).
func (tx *readWriteTx) OnCommit(cb func()) {
	tx.stm.OnCommit(cb)
}
