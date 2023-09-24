package migtest

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/lnd/channeldb/kvdb"
)

// MakeDB creates a new instance of the ChannelDB for testing purposes. A
// callback which cleans up the created temporary directories is also returned
// and intended to be executed after the test completes.
func MakeDB() (kvdb.Backend, func(), er.R) {
	// Create temporary database for mission control.
	file, errr := ioutil.TempFile("", "*.db")
	if errr != nil {
		return nil, nil, er.E(errr)
	}

	dbPath := file.Name()
	db, err := kvdb.Open(kvdb.BoltBackendName, dbPath, true)
	if err != nil {
		return nil, nil, err
	}

	cleanUp := func() {
		db.Close()
		os.RemoveAll(dbPath)
	}

	return db, cleanUp, nil
}

// ApplyMigration is a helper test function that encapsulates the general steps
// which are needed to properly check the result of applying migration function.
func ApplyMigration(t *testing.T,
	beforeMigration, afterMigration, migrationFunc func(tx kvdb.RwTx) er.R,
	shouldFail bool) {

	cdb, cleanUp, err := MakeDB()
	defer cleanUp()
	if err != nil {
		t.Fatal(err)
	}

	// beforeMigration usually used for populating the database
	// with test data.
	err = kvdb.Update(cdb, beforeMigration, func() {})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r != nil {
			err = newError(r)
		}

		if err == nil && shouldFail {
			t.Fatal("error wasn't received on migration stage")
		} else if err != nil && !shouldFail {
			t.Fatalf("error was received on migration stage: %v", err)
		}

		// afterMigration usually used for checking the database state and
		// throwing the error if something went wrong.
		err = kvdb.Update(cdb, afterMigration, func() {})
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Apply migration.
	err = kvdb.Update(cdb, migrationFunc, func() {})
	if err != nil {
		t.Logf("migration error: %v", err)
	}
}

func newError(e interface{}) er.R {
	var err er.R
	switch e := e.(type) {
	case er.R:
		err = e
	case error:
		err = er.E(e)
	default:
		err = er.Errorf("%v", e)
	}

	return err
}
