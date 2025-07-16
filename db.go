// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/thediveo/lxkns/model"
	"github.com/thediveo/procfsroot"

	// _ "github.com/mattn/go-sqlite3" // pull in "sqlite3" driver
	_ "modernc.org/sqlite"
)

// PlatformBoxDb is the file name of the platform box database.
const PlatformBoxDb = "platformbox.db"

// Location of app engine-related SQLite database files.
const dbBaseDir = "/data/app_engine/db"

// Name of DB driver.
const dbDriverName = "sqlite"

// AppEngineDB implements access to an IED app engine host database.
type AppEngineDB struct {
	*sqlx.DB
	copiedDatabasePath string
	closeMu            sync.Mutex
}

// Open returns a new database “connection” to the specified app engine DB, such
// as “platformboxdb”, or an error, if neither the IED runtime container nor its
// app engine database(s) could be located.
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
//
// As it turns out there are some situations which we don't yet fully understand
// but that causes opening the database via a proc path to fail, even if it
// succeeds in other situations. Interestingly, the termdbms sqlite3 TUI works
// in all these situation and an analysis of its source code base reveals that
// it simply makes a copy of the original database, see
// https://github.com/mathaou/termdbms/blob/be6f397196077cc7c9ced86e6460470e3b223f3e/main.go#L132.
//
// Well, what's good for the goose is good for the gander, so copy it is. Sigh.
func open(name string, pid model.PIDType) (*AppEngineDB, error) {
	rootpath := fmt.Sprintf("/proc/%d/root", pid)
	dbpath, err := procfsroot.EvalSymlinks(name, rootpath, procfsroot.EvalFullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot determine full database path, reason: %w", err)
	}
	dbpath = path.Join(rootpath, dbpath)

	// Make a temporary copy of the database so we can open it successfully in
	// all our cases.
	origdbf, err := os.Open(dbpath)
	if err != nil {
		return nil, fmt.Errorf("unable to open database, reason: %w", err)
	}
	defer func() { _ = origdbf.Close() }()
	tmpdbf, err := os.CreateTemp("", "temp-db-copy-*")
	if err != nil {
		return nil, fmt.Errorf("unable to open database, reason: %w", err)
	}
	defer func() { _ = tmpdbf.Close() }()
	if _, err := io.Copy(tmpdbf, origdbf); err != nil {
		_ = os.Remove(tmpdbf.Name())
		return nil, fmt.Errorf("unable to open database, reason: %w", err)
	}

	// As sql.Open might just "validate its parameters" and this might mean near
	// to nothing, we explicitly ping the database in order to see that it is
	// okay.
	dbpath = tmpdbf.Name()
	_ = tmpdbf.Close()
	db, err := sqlx.Open(dbDriverName, dbpath)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Install a default mapper function that preserves camelCase, with the
	// first letter always being lowercase.
	db.MapperFunc(FirstLower)

	// Success, now wrap the sql.DB object in our AppEngineDB object, so that we
	// later correctly can clean up when the Close method gets called.
	return &AppEngineDB{
		DB:                 db,
		copiedDatabasePath: dbpath,
	}, nil
}

// Close closes the database connection and ensures to additionally dispose of
// the helper resources required to read from an SQLite database in another
// container.
func (db *AppEngineDB) Close() error {
	db.closeMu.Lock()
	defer db.closeMu.Unlock()
	err := db.DB.Close()
	if db.copiedDatabasePath != "" {
		_ = os.Remove(db.copiedDatabasePath)
		db.copiedDatabasePath = ""
	}
	return err
}
