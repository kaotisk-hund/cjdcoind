// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bdb

import (
	"fmt"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"

	"github.com/kaotisk-hund/cjdcoind/cjdcoinwallet/walletdb"
)

const (
	dbType = "bdb"
)

// parseArgs parses the arguments from the walletdb Open/Create methods.
func parseArgs(funcName string, args ...interface{}) (string, er.R) {
	if len(args) != 1 {
		return "", er.Errorf("invalid arguments to %s.%s -- "+
			"expected database path", dbType, funcName)
	}

	dbPath, ok := args[0].(string)
	if !ok {
		return "", er.Errorf("first argument to %s.%s is invalid -- "+
			"expected database path string", dbType, funcName)
	}

	return dbPath, nil
}

// openDBDriver is the callback provided during driver registration that opens
// an existing database for use.
func openDBDriver(dbPath string, noFreeListSync bool) (walletdb.DB, er.R) {
	return openDB(dbPath, false, noFreeListSync)
}

// createDBDriver is the callback provided during driver registration that
// creates, initializes, and opens a database for use.
func createDBDriver(dbPath string, noFreeListSync bool) (walletdb.DB, er.R) {
	return openDB(dbPath, true, noFreeListSync)
}

func init() {
	// Register the driver.
	driver := walletdb.Driver{
		DbType: dbType,
		Create: createDBDriver,
		Open:   openDBDriver,
	}
	if err := walletdb.RegisterDriver(driver); err != nil {
		panic(fmt.Sprintf("Failed to regiser database driver '%s': %v",
			dbType, err))
	}
}
