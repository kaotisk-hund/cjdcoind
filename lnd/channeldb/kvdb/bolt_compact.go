// The code in this file is an adapted version of the bbolt compact command
// implemented in this file:
// https://github.com/etcd-io/bbolt/blob/master/cmd/bbolt/main.go

package kvdb

import (
	"os"
	"path"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/healthcheck"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinlog/log"
	"go.etcd.io/bbolt"
)

const (
	// defaultResultFileSizeMultiplier is the default multiplier we apply to
	// the current database size to calculate how big it could possibly get
	// after compacting, in case the database is already at its optimal size
	// and compaction causes it to grow. This should normally not be the
	// case but we really want to avoid not having enough disk space for the
	// compaction, so we apply a safety margin of 10%.
	defaultResultFileSizeMultiplier = float64(1.1)

	// defaultTxMaxSize is the default maximum number of operations that
	// are allowed to be executed in a single transaction.
	defaultTxMaxSize = 65536

	// bucketFillSize is the fill size setting that is used for each new
	// bucket that is created in the compacted database. This setting is not
	// persisted and is therefore only effective for the compaction itself.
	// Because during the compaction we only append data a fill percent of
	// 100% is optimal for performance.
	bucketFillSize = 1.0
)

type compacter struct {
	srcPath   string
	dstPath   string
	txMaxSize int64
}

// execute opens the source and destination databases and then compacts the
// source into destination and returns the size of both files as a result.
func (cmd *compacter) execute() (int64, int64, er.R) {
	if cmd.txMaxSize == 0 {
		cmd.txMaxSize = defaultTxMaxSize
	}

	// Ensure source file exists.
	fi, errr := os.Stat(cmd.srcPath)
	if errr != nil {
		return 0, 0, er.Errorf("error determining source database "+
			"size: %v", errr)
	}
	initialSize := fi.Size()
	marginSize := float64(initialSize) * defaultResultFileSizeMultiplier

	// Before opening any of the databases, let's first make sure we have
	// enough free space on the destination file system to create a full
	// copy of the source DB (worst-case scenario if the compaction doesn't
	// actually shrink the file size).
	destFolder := path.Dir(cmd.dstPath)
	freeSpace, err := healthcheck.AvailableDiskSpace(destFolder)
	if err != nil {
		return 0, 0, er.Errorf("error determining free disk space on "+
			"%s: %v", destFolder, err)
	}
	log.Debugf("Free disk space on compaction destination file system: "+
		"%d bytes", freeSpace)
	if freeSpace < uint64(marginSize) {
		return 0, 0, er.Errorf("could not start compaction, "+
			"destination folder %s only has %d bytes of free disk "+
			"space available while we need at least %d for worst-"+
			"case compaction", destFolder, freeSpace, initialSize)
	}

	// Open source database. We open it in read only mode to avoid (and fix)
	// possible freelist sync problems.
	src, errr := bbolt.Open(cmd.srcPath, 0444, &bbolt.Options{
		ReadOnly: true,
	})
	if errr != nil {
		return 0, 0, er.Errorf("error opening source database: %v",
			errr)
	}
	defer func() {
		if err := src.Close(); err != nil {
			log.Errorf("Compact error: closing source DB: %v", err)
		}
	}()

	// Open destination database.
	dst, errr := bbolt.Open(cmd.dstPath, fi.Mode(), nil)
	if errr != nil {
		return 0, 0, er.Errorf("error opening destination database: "+
			"%v", errr)
	}
	defer func() {
		if err := dst.Close(); err != nil {
			log.Errorf("Compact error: closing dest DB: %v", err)
		}
	}()

	// Run compaction.
	if err := cmd.compact(dst, src); err != nil {
		return 0, 0, er.Errorf("error running compaction: %v", err)
	}

	// Report stats on new size.
	fi, errr = os.Stat(cmd.dstPath)
	if errr != nil {
		return 0, 0, er.Errorf("error determining destination "+
			"database size: %v", errr)
	} else if fi.Size() == 0 {
		return 0, 0, er.Errorf("zero db size")
	}

	return initialSize, fi.Size(), nil
}

// compact tries to create a compacted copy of the source database in a new
// destination database.
func (cmd *compacter) compact(dst, src *bbolt.DB) er.R {
	// Commit regularly, or we'll run out of memory for large datasets if
	// using one transaction.
	var size int64
	tx, err := dst.Begin(true)
	if err != nil {
		return er.E(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := cmd.walk(src, func(keys [][]byte, k, v []byte, seq uint64) er.R {
		// On each key/value, check if we have exceeded tx size.
		sz := int64(len(k) + len(v))
		if size+sz > cmd.txMaxSize && cmd.txMaxSize != 0 {
			// Commit previous transaction.
			if err := tx.Commit(); err != nil {
				return er.E(err)
			}

			// Start new transaction.
			tx, err = dst.Begin(true)
			if err != nil {
				return er.E(err)
			}
			size = 0
		}
		size += sz

		// Create bucket on the root transaction if this is the first
		// level.
		nk := len(keys)
		if nk == 0 {
			bkt, err := tx.CreateBucket(k)
			if err != nil {
				return er.E(err)
			}
			if err := bkt.SetSequence(seq); err != nil {
				return er.E(err)
			}
			return nil
		}

		// Create buckets on subsequent levels, if necessary.
		b := tx.Bucket(keys[0])
		if nk > 1 {
			for _, k := range keys[1:] {
				b = b.Bucket(k)
			}
		}

		// Fill the entire page for best compaction.
		b.FillPercent = bucketFillSize

		// If there is no value then this is a bucket call.
		if v == nil {
			bkt, err := b.CreateBucket(k)
			if err != nil {
				return er.E(err)
			}
			if err := bkt.SetSequence(seq); err != nil {
				return er.E(err)
			}
			return nil
		}

		// Otherwise treat it as a key/value pair.
		return er.E(b.Put(k, v))
	}); err != nil {
		return err
	}

	return er.E(tx.Commit())
}

// walkFunc is the type of the function called for keys (buckets and "normal"
// values) discovered by Walk. keys is the list of keys to descend to the bucket
// owning the discovered key/value pair k/v.
type walkFunc func(keys [][]byte, k, v []byte, seq uint64) er.R

// walk walks recursively the bolt database db, calling walkFn for each key it
// finds.
func (cmd *compacter) walk(db *bbolt.DB, walkFn walkFunc) er.R {
	return er.E(db.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			// This will log the top level buckets only to give the
			// user some sense of progress.
			log.Debugf("Compacting top level bucket %s", name)

			return er.Native(cmd.walkBucket(
				b, nil, name, nil, b.Sequence(), walkFn,
			))
		})
	}))
}

// walkBucket recursively walks through a bucket.
func (cmd *compacter) walkBucket(b *bbolt.Bucket, keyPath [][]byte, k, v []byte,
	seq uint64, fn walkFunc) er.R {

	// Execute callback.
	if err := fn(keyPath, k, v, seq); err != nil {
		return err
	}

	// If this is not a bucket then stop.
	if v != nil {
		return nil
	}

	// Iterate over each child key/value.
	keyPath = append(keyPath, k)
	return er.E(b.ForEach(func(k, v []byte) error {
		if v == nil {
			bkt := b.Bucket(k)
			return er.Native(cmd.walkBucket(
				bkt, keyPath, k, nil, bkt.Sequence(), fn,
			))
		}
		return er.Native(cmd.walkBucket(b, keyPath, k, v, b.Sequence(), fn))
	}))
}
