// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bdb_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"

	"github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/walletdb"
	_ "github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/walletdb/bdb"
)

// dbType is the database type name for this driver.
const dbType = "bdb"

// TestCreateOpenFail ensures that errors related to creating and opening a
// database are handled properly.
func TestCreateOpenFail(t *testing.T) {
	// Ensure that attempting to open a database that doesn't exist returns
	// the expected error.
	wantErr := walletdb.ErrDbDoesNotExist.Default()
	if _, err := walletdb.Open(dbType, "noexist.db", true); !er.Equals(err, wantErr) {
		t.Errorf("Open: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

	// Ensure operations against a closed database return the expected
	// error.
	dbPath := "createfail.db"
	db, err := walletdb.Create(dbType, dbPath, true)
	if err != nil {
		t.Errorf("Create: unexpected error: %v", err)
		return
	}
	defer os.Remove(dbPath)
	db.Close()

	wantErr = walletdb.ErrDbNotOpen.Default()
	if _, err := db.BeginReadTx(); !er.Equals(err, wantErr) {
		t.Errorf("Namespace: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}
}

// TestPersistence ensures that values stored are still valid after closing and
// reopening the database.
func TestPersistence(t *testing.T) {
	// Create a new database to run tests against.
	dbPath := "persistencetest.db"
	db, err := walletdb.Create(dbType, dbPath, true)
	if err != nil {
		t.Errorf("Failed to create test database (%s) %v", dbType, err)
		return
	}
	defer os.Remove(dbPath)
	defer db.Close()

	// Create a namespace and put some values into it so they can be tested
	// for existence on re-open.
	storeValues := map[string]string{
		"ns1key1": "foo1",
		"ns1key2": "foo2",
		"ns1key3": "foo3",
	}
	ns1Key := []byte("ns1")
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) er.R {
		ns1, err := tx.CreateTopLevelBucket(ns1Key)
		if err != nil {
			return err
		}

		for k, v := range storeValues {
			if err := ns1.Put([]byte(k), []byte(v)); err != nil {
				return er.Errorf("Put: unexpected error: %v", err)
			}
		}

		return nil
	})
	if err != nil {
		t.Errorf("ns1 Update: unexpected error: %v", err)
		return
	}

	// Close and reopen the database to ensure the values persist.
	db.Close()
	db, err = walletdb.Open(dbType, dbPath, true)
	if err != nil {
		t.Errorf("Failed to open test database (%s) %v", dbType, err)
		return
	}
	defer db.Close()

	// Ensure the values previously stored in the 3rd namespace still exist
	// and are correct.
	err = walletdb.View(db, func(tx walletdb.ReadTx) er.R {
		ns1 := tx.ReadBucket(ns1Key)
		if ns1 == nil {
			return er.Errorf("ReadTx.ReadBucket: unexpected nil root bucket")
		}

		for k, v := range storeValues {
			gotVal := ns1.Get([]byte(k))
			if !reflect.DeepEqual(gotVal, []byte(v)) {
				return er.Errorf("Get: key '%s' does not "+
					"match expected value - got %s, want %s",
					k, gotVal, v)
			}
		}

		return nil
	})
	if err != nil {
		t.Errorf("ns1 View: unexpected error: %v", err)
		return
	}
}
