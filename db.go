// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"fmt"
	"path"
	"regexp"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/thediveo/lxkns/model"
	"github.com/thediveo/lxkns/ops/mountineer"

	_ "github.com/mattn/go-sqlite3" // pull in "sqlite3" driver
)

// PlatformBoxDb is the file name of the platform box database.
const PlatformBoxDb = "platformbox.db"

// Location of app engine-related SQLite database files.
const dbBaseDir = "/data/app_engine/db"

// Name of DB driver.
const dbDriverName = "sqlite3"

// AppEngineDB implements access to an IED app engine host database.
type AppEngineDB struct {
	*sqlx.DB
	iedRuntimeMnt *mountineer.Mountineer
	mu            sync.Mutex
}

// Open returns a new database “connection” to the specified app engine DB (in
// read-only mode), such as “platformboxdb”, or an error, if neither the IED
// runtime container nor its app engine database(s) could be located.
//
// The database name is sanitizied by keeping only alpha-numerically (ASCII)
// characters, as well as dots “.” (but not “..”), dashes “-”, and underscores
// “_”.
//
// Open hides the details of discovering the IED runtime container in a way that
// then gives direct file system access to the app engine DBs inside this
// container.
func Open(dbname string) (*AppEngineDB, error) {
	// no core, no cigar.
	corePID, err := edgeCoreContainerPID()
	if err != nil {
		return nil, err
	}
	return OpenInPID(dbname, corePID)
}

// OpenInPID works like Open, but additionally requires the PID of the container
// with the app engine DB(s) to be explicitly specified. Use OpenInPID when the
// IED's runtime container PID is already known, such as from an lxkns
// discovery, as so to skip the IE runtime container discovery.
func OpenInPID(dbname string, pid model.PIDType) (*AppEngineDB, error) {
	return open(path.Join(dbBaseDir, sanitize(dbname)), pid)
}

var onlyAlphaNumsAndMore = regexp.MustCompile(`[^a-zA-Z0-9\-_.]+`)
var noDotDots = regexp.MustCompile(`(\.\.)+`)

// sanitize the specified basename so it does not contain any characters
// triggering path traversal or SQLite driver options. Sanitize keeps only
// alpha-numerically (ASCII) characters, as well as dots “.” (but not “..”),
// dashes “-”, and underscores “_”.
func sanitize(basename string) string {
	return noDotDots.ReplaceAllString(onlyAlphaNumsAndMore.ReplaceAllString(basename, "_"), "_")
}

// open actually opens the SQLite database read-only specified by its full path,
// with the help of a mountineer. Separating this out gives us a chance to reuse
// it in some tests without the need for a correctly set-up fake edge runtime
// container. Please note that any caller must have sanitized the name parameter
// first.
func open(name string, pid model.PIDType) (*AppEngineDB, error) {
	// Call in the Mountineers to gain drive-by access to the core's container
	// live file system.
	iedRuntimeMnt, err := mountineer.New(
		model.NamespaceRef{fmt.Sprintf("/proc/%d/ns/mnt", pid)}, nil)
	if err != nil {
		return nil, err
	}

	dbpath, err := iedRuntimeMnt.Resolve(name)
	if err != nil {
		iedRuntimeMnt.Close()
		return nil, err
	}

	// As sql.Open might just "validate its parameters" and this might mean near
	// to nothing, we explicitly ping the database in order to see that it is
	// okay.
	db, err := sqlx.Open(dbDriverName, dbpath+"?mode=ro")
	if err == nil {
		err = db.Ping()
	}
	if err != nil {
		iedRuntimeMnt.Close()
		return nil, err
	}

	// Install a default mapper function that preserves camelCase, with the
	// first letter always being lowercase.
	db.MapperFunc(FirstLower)

	// Success, now wrap the sql.DB object in our AppEngineDB object, so that we
	// later correctly can clean up when the Close method gets called.
	return &AppEngineDB{
		iedRuntimeMnt: iedRuntimeMnt,
		DB:            db,
	}, nil
}

// Close closes the database connection and ensures to additionally dispose of
// the helper resources required to read from an SQLite database in another
// container.
func (db *AppEngineDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	err := db.DB.Close()
	if db.iedRuntimeMnt != nil {
		// The mountineer now can unmount and rest them horse.
		db.iedRuntimeMnt.Close()
	}
	return err
}
