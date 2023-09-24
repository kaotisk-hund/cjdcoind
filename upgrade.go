// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/cjdcoinlog/log"
)

// dirEmpty returns whether or not the specified directory path is empty.
func dirEmpty(dirPath string) (bool, er.R) {
	f, errr := os.Open(dirPath)
	if errr != nil {
		return false, er.E(errr)
	}
	defer f.Close()

	// Read the names of a max of one entry from the directory.  When the
	// directory is empty, an io.EOF error will be returned, so allow it.
	names, errr := f.Readdirnames(1)
	if errr != nil && errr != io.EOF {
		return false, er.E(errr)
	}

	return len(names) == 0, nil
}

// oldBtcdHomeDir returns the OS specific home directory cjdcoind used prior to
// version 0.3.3.  This has since been replaced with btcutil.AppDataDir, but
// this function is still provided for the automatic upgrade path.
func oldBtcdHomeDir() string {
	// Search for Windows APPDATA first.  This won't exist on POSIX OSes.
	appData := os.Getenv("APPDATA")
	if appData != "" {
		return filepath.Join(appData, "cjdcoind")
	}

	// Fall back to standard HOME directory that works for most POSIX OSes.
	home := os.Getenv("HOME")
	if home != "" {
		return filepath.Join(home, ".cjdcoind")
	}

	// In the worst case, use the current directory.
	return "."
}

// upgradeDBPathNet moves the database for a specific network from its
// location prior to cjdcoind version 0.2.0 and uses heuristics to ascertain the old
// database type to rename to the new format.
func upgradeDBPathNet(oldDbPath, netName string) er.R {
	// Prior to version 0.2.0, the database was named the same thing for
	// both sqlite and leveldb.  Use heuristics to figure out the type
	// of the database and move it to the new path and name introduced with
	// version 0.2.0 accordingly.
	fi, err := os.Stat(oldDbPath)
	if err == nil {
		oldDbType := "sqlite"
		if fi.IsDir() {
			oldDbType = "leveldb"
		}

		// The new database name is based on the database type and
		// resides in a directory named after the network type.
		newDbRoot := filepath.Join(filepath.Dir(cfg.DataDir), netName)
		newDbName := blockDbNamePrefix + "_" + oldDbType
		if oldDbType == "sqlite" {
			newDbName = newDbName + ".db"
		}
		newDbPath := filepath.Join(newDbRoot, newDbName)

		// Create the new path if needed.
		errr := os.MkdirAll(newDbRoot, 0700)
		if errr != nil {
			return er.E(errr)
		}

		// Move and rename the old database.
		errr = os.Rename(oldDbPath, newDbPath)
		if errr != nil {
			return er.E(errr)
		}
	}

	return nil
}

// upgradeDBPaths moves the databases from their locations prior to cjdcoind
// version 0.2.0 to their new locations.
func upgradeDBPaths() er.R {
	// Prior to version 0.2.0, the databases were in the "db" directory and
	// their names were suffixed by "testnet" and "regtest" for their
	// respective networks.  Check for the old database and update it to the
	// new path introduced with version 0.2.0 accordingly.
	oldDbRoot := filepath.Join(oldBtcdHomeDir(), "db")
	upgradeDBPathNet(filepath.Join(oldDbRoot, "cjdcoind.db"), "mainnet")
	upgradeDBPathNet(filepath.Join(oldDbRoot, "cjdcoind_testnet.db"), "testnet")
	upgradeDBPathNet(filepath.Join(oldDbRoot, "cjdcoind_regtest.db"), "regtest")

	// Remove the old db directory.
	return er.E(os.RemoveAll(oldDbRoot))
}

// upgradeDataPaths moves the application data from its location prior to cjdcoind
// version 0.3.3 to its new location.
func upgradeDataPaths() er.R {
	// No need to migrate if the old and new home paths are the same.
	oldHomePath := oldBtcdHomeDir()
	newHomePath := defaultHomeDir
	if oldHomePath == newHomePath {
		return nil
	}

	// Only migrate if the old path exists and the new one doesn't.
	if fileExists(oldHomePath) && !fileExists(newHomePath) {
		// Create the new path.
		log.Infof("Migrating application home path from '%s' to '%s'",
			oldHomePath, newHomePath)
		errr := os.MkdirAll(newHomePath, 0700)
		if errr != nil {
			return er.E(errr)
		}

		// Move old cjdcoind.conf into new location if needed.
		oldConfPath := filepath.Join(oldHomePath, defaultConfigFilename)
		newConfPath := filepath.Join(newHomePath, defaultConfigFilename)
		if fileExists(oldConfPath) && !fileExists(newConfPath) {
			errr := os.Rename(oldConfPath, newConfPath)
			if errr != nil {
				return er.E(errr)
			}
		}

		// Move old data directory into new location if needed.
		oldDataPath := filepath.Join(oldHomePath, defaultDataDirname)
		newDataPath := filepath.Join(newHomePath, defaultDataDirname)
		if fileExists(oldDataPath) && !fileExists(newDataPath) {
			errr := os.Rename(oldDataPath, newDataPath)
			if errr != nil {
				return er.E(errr)
			}
		}

		// Remove the old home if it is empty or show a warning if not.
		ohpEmpty, err := dirEmpty(oldHomePath)
		if err != nil {
			return err
		}
		if ohpEmpty {
			errr := os.Remove(oldHomePath)
			if errr != nil {
				return er.E(errr)
			}
		} else {
			log.Warnf("Not removing '%s' since it contains files "+
				"not created by this application.  You may "+
				"want to manually move them or delete them.",
				oldHomePath)
		}
	}

	return nil
}

// doUpgrades performs upgrades to cjdcoind as new versions require it.
func doUpgrades() er.R {
	err := upgradeDBPaths()
	if err != nil {
		return err
	}
	return upgradeDataPaths()
}
