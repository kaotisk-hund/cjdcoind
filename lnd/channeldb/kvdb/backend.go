package kvdb

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinlog/log"
	_ "github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/walletdb/bdb" // Import to register backend.
)

const (
	// DefaultTempDBFileName is the default name of the temporary bolt DB
	// file that we'll use to atomically compact the primary DB file on
	// startup.
	DefaultTempDBFileName = "temp-dont-use.db"

	// LastCompactionFileNameSuffix is the suffix we append to the file name
	// of a database file to record the timestamp when the last compaction
	// occurred.
	LastCompactionFileNameSuffix = ".last-compacted"
)

var (
	byteOrder = binary.BigEndian
)

// fileExists returns true if the file exists, and false otherwise.
func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// BoltBackendConfig is a struct that holds settings specific to the bolt
// database backend.
type BoltBackendConfig struct {
	// DBPath is the directory path in which the database file should be
	// stored.
	DBPath string

	// DBFileName is the name of the database file.
	DBFileName string

	// NoFreelistSync, if true, prevents the database from syncing its
	// freelist to disk, resulting in improved performance at the expense of
	// increased startup time.
	NoFreelistSync bool

	// AutoCompact specifies if a Bolt based database backend should be
	// automatically compacted on startup (if the minimum age of the
	// database file is reached). This will require additional disk space
	// for the compacted copy of the database but will result in an overall
	// lower database size after the compaction.
	AutoCompact bool

	// AutoCompactMinAge specifies the minimum time that must have passed
	// since a bolt database file was last compacted for the compaction to
	// be considered again.
	AutoCompactMinAge time.Duration
}

// GetBoltBackend opens (or creates if doesn't exits) a bbolt backed database
// and returns a kvdb.Backend wrapping it.
func GetBoltBackend(cfg *BoltBackendConfig) (Backend, er.R) {
	dbFilePath := filepath.Join(cfg.DBPath, cfg.DBFileName)

	// Is this a new database?
	if !fileExists(dbFilePath) {
		if !fileExists(cfg.DBPath) {
			if err := os.MkdirAll(cfg.DBPath, 0700); err != nil {
				return nil, er.E(err)
			}
		}

		return Create(BoltBackendName, dbFilePath, cfg.NoFreelistSync)
	}

	// This is an existing database. We might want to compact it on startup
	// to free up some space.
	if cfg.AutoCompact {
		if err := compactAndSwap(cfg); err != nil {
			return nil, err
		}
	}

	return Open(BoltBackendName, dbFilePath, cfg.NoFreelistSync)
}

// compactAndSwap will attempt to write a new temporary DB file to disk with
// the compacted database content, then atomically swap (via rename) the old
// file for the new file by updating the name of the new file to the old.
func compactAndSwap(cfg *BoltBackendConfig) er.R {
	sourceName := cfg.DBFileName

	// If the main DB file isn't set, then we can't proceed.
	if sourceName == "" {
		return er.Errorf("cannot compact DB with empty name")
	}
	sourceFilePath := filepath.Join(cfg.DBPath, sourceName)
	tempDestFilePath := filepath.Join(cfg.DBPath, DefaultTempDBFileName)

	// Let's find out how long ago the last compaction of the source file
	// occurred and possibly skip compacting it again now.
	lastCompactionDate, err := lastCompactionDate(sourceFilePath)
	if err != nil {
		return er.Errorf("cannot determine last compaction date of "+
			"source DB file: %v", err)
	}
	compactAge := time.Since(lastCompactionDate)
	if cfg.AutoCompactMinAge != 0 && compactAge <= cfg.AutoCompactMinAge {
		log.Infof("Not compacting database file at %v, it was last "+
			"compacted at %v (%v ago), min age is set to %v",
			sourceFilePath, lastCompactionDate,
			compactAge.Truncate(time.Second), cfg.AutoCompactMinAge)
		return nil
	}

	log.Infof("Compacting database file at %v", sourceFilePath)

	// If the old temporary DB file still exists, then we'll delete it
	// before proceeding.
	if _, err := os.Stat(tempDestFilePath); err == nil {
		log.Infof("Found old temp DB @ %v, removing before swap",
			tempDestFilePath)

		err = os.Remove(tempDestFilePath)
		if err != nil {
			return er.Errorf("unable to remove old temp DB file: "+
				"%v", err)
		}
	}

	// Now that we know the staging area is clear, we'll create the new
	// temporary DB file and close it before we write the new DB to it.
	tempFile, errr := os.Create(tempDestFilePath)
	if errr != nil {
		return er.Errorf("unable to create temp DB file: %v", errr)
	}
	if err := tempFile.Close(); err != nil {
		return er.Errorf("unable to close file: %v", err)
	}

	// With the file created, we'll start the compaction and remove the
	// temporary file all together once this method exits.
	defer func() {
		// This will only succeed if the rename below fails. If the
		// compaction is successful, the file won't exist on exit
		// anymore so no need to log an error here.
		_ = os.Remove(tempDestFilePath)
	}()
	c := &compacter{
		srcPath: sourceFilePath,
		dstPath: tempDestFilePath,
	}
	initialSize, newSize, err := c.execute()
	if err != nil {
		return er.Errorf("error during compact: %v", err)
	}

	log.Infof("DB compaction of %v successful, %d -> %d bytes (gain=%.2fx)",
		sourceFilePath, initialSize, newSize,
		float64(initialSize)/float64(newSize))

	// We try to store the current timestamp in a file with the suffix
	// .last-compacted so we can figure out how long ago the last compaction
	// was. But since this shouldn't fail the compaction process itself, we
	// only log the error. Worst case if this file cannot be written is that
	// we compact on every startup.
	err = updateLastCompactionDate(sourceFilePath)
	if err != nil {
		log.Warnf("Could not update last compaction timestamp in "+
			"%s%s: %v", sourceFilePath,
			LastCompactionFileNameSuffix, err)
	}

	log.Infof("Swapping old DB file from %v to %v", tempDestFilePath,
		sourceFilePath)

	// Finally, we'll attempt to atomically rename the temporary file to
	// the main back up file. If this succeeds, then we'll only have a
	// single file on disk once this method exits.
	return er.E(os.Rename(tempDestFilePath, sourceFilePath))
}

// lastCompactionDate returns the date the given database file was last
// compacted or a zero time.Time if no compaction was recorded before. The
// compaction date is read from a file in the same directory and with the same
// name as the DB file, but with the suffix ".last-compacted".
func lastCompactionDate(dbFile string) (time.Time, er.R) {
	zeroTime := time.Unix(0, 0)

	tsFile := fmt.Sprintf("%s%s", dbFile, LastCompactionFileNameSuffix)
	if !fileExists(tsFile) {
		return zeroTime, nil
	}

	tsBytes, err := ioutil.ReadFile(tsFile)
	if err != nil {
		return zeroTime, er.E(err)
	}

	tsNano := byteOrder.Uint64(tsBytes)
	return time.Unix(0, int64(tsNano)), nil
}

// updateLastCompactionDate stores the current time as a timestamp in a file
// in the same directory and with the same name as the DB file, but with the
// suffix ".last-compacted".
func updateLastCompactionDate(dbFile string) er.R {
	var tsBytes [8]byte
	byteOrder.PutUint64(tsBytes[:], uint64(time.Now().UnixNano()))

	tsFile := fmt.Sprintf("%s%s", dbFile, LastCompactionFileNameSuffix)
	return er.E(ioutil.WriteFile(tsFile, tsBytes[:], 0600))
}

// GetTestBackend opens (or creates if doesn't exist) a bbolt or etcd
// backed database (for testing), and returns a kvdb.Backend and a cleanup
// func. Whether to create/open bbolt or embedded etcd database is based
// on the TestBackend constant which is conditionally compiled with build tag.
// The passed path is used to hold all db files, while the name is only used
// for bbolt.
func GetTestBackend(path, name string) (Backend, func(), er.R) {
	empty := func() {}

	if TestBackend == BoltBackendName {
		db, err := GetBoltBackend(&BoltBackendConfig{
			DBPath:         path,
			DBFileName:     name,
			NoFreelistSync: true,
		})
		if err != nil {
			return nil, nil, err
		}
		return db, empty, nil
	} else if TestBackend == EtcdBackendName {
		return GetEtcdTestBackend(path, name)
	}

	return nil, nil, er.Errorf("unknown backend")
}