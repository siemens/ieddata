// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/thediveo/lxkns/model"
	"github.com/thediveo/procfsroot/wormholes"
	_ "modernc.org/sqlite"
)

// PlatformBoxDb is the file name of the platform box database.
const PlatformBoxDb = "platformbox.db"

// Location of app engine-related SQLite database files.
const dbBaseDir = "/data/app_engine/db"

// Name of DB driver.
const dbDriverName = "sqlite"

// openTimeout limits how long Open waits for discovering the IED runtime
// container.
const openTimeout = 30 * time.Second

// maxCopyAttempts limits how often to copy a database that keeps changing
// while being copied.
const maxCopyAttempts = 3

// AppEngineDB implements access to an IED app engine host database.
//
// Always use [AppEngineDB.Close] instead of the embedded DB's Close, as only
// the former removes the temporary database copy.
type AppEngineDB struct {
	*sqlx.DB
	tmpDir  string // holds the database copy and its WAL/SHM files
	closeMu sync.Mutex
}

// Open returns a new database “connection” to the specified app engine DB, such
// as “platformbox.db”, or an error, if neither the IED runtime container nor its
// app engine database(s) could be located.
//
// The database name must consist only of ASCII letters and digits, dots “.”
// (but not “..”), dashes “-”, and underscores “_”; otherwise, Open returns
// an error wrapping [ErrInvalidDBName].
//
// Open hides the details of discovering the IED runtime container in a way that
// then gives direct file system access to the app engine DBs inside this
// container. Open gives up discovering the IED runtime container after 30s;
// use [OpenCtx] for a different timeout or cancellation.
//
// Open reads from a temporary copy of the database; see [OpenFS] for details.
func Open(dbname string) (*AppEngineDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), openTimeout)
	defer cancel()
	return OpenCtx(ctx, dbname)
}

// OpenCtx works like Open, but uses the passed context instead of a 30s timeout
// for discovering the IED runtime container.
//
// The context covers only this discovery, not opening the database itself nor
// any queries on the returned database; cancelling the context after OpenCtx
// returned has no effect on the returned database.
func OpenCtx(ctx context.Context, dbname string, opts ...Option) (*AppEngineDB, error) {
	if err := checkDBName(dbname); err != nil {
		return nil, err
	}
	options := options{dockerHost: defaultDockerHost}
	for _, opt := range opts {
		opt(&options)
	}
	// no core, no cigar.
	corePID, err := edgeCoreContainerPID(ctx, options.dockerHost)
	if err != nil {
		return nil, err
	}
	return OpenInPID(dbname, corePID)
}

// Option configures [OpenCtx].
type Option func(*options)

type options struct {
	dockerHost string
}

// WithDockerHost sets the Docker API endpoint for discovering the IED runtime
// container, such as “unix:///run/docker.sock”. It defaults to
// “unix:///proc/1/root/run/docker.sock”, that is, the host's Docker socket as
// seen from within the initial PID namespace.
func WithDockerHost(host string) Option {
	return func(o *options) { o.dockerHost = host }
}

// OpenInPID works like Open, but additionally requires the PID of the container
// with the app engine DB(s) to be explicitly specified. Use OpenInPID when the
// IED's runtime container PID is already known, such as from an lxkns
// discovery, as so to skip the IE runtime container discovery.
//
// As Open creates a new container engine watcher each time to discover the IED
// runtime container, callers that open databases often and already run their
// own [github.com/thediveo/whalewatcher/v2/watcher.Watcher] should look up the
// IED runtime container themselves instead:
//
//	core := w.Portfolio().Container(ieddata.EdgeIotCoreContainerName)
//	if core == nil {
//		// no IED runtime container (yet)
//	}
//	db, err := ieddata.OpenInPID(ieddata.PlatformBoxDb, model.PIDType(core.PID))
func OpenInPID(dbname string, pid model.PIDType) (*AppEngineDB, error) {
	if err := checkDBName(dbname); err != nil {
		return nil, err
	}
	return OpenPathInPID(path.Join(dbBaseDir, dbname), pid)
}

// OpenPathInPID works like OpenInPID but takes a full path and database name as
// well as a PID within which to resolve the pathname. Use this primarily in
// (unit) tests where spinning up a fake IED runtime container is overkill.
//
// Deprecated: use [OpenFS] with a [wormholes.FS] instead.
func OpenPathInPID(pathname string, pid model.PIDType) (*AppEngineDB, error) {
	return open(wormholes.New(int(pid)), strings.TrimPrefix(pathname, "/"))
}

// OpenFS works like Open but opens a temporary copy of the specified database
// in the passed fs.FS.
//
// The copy, including any WAL file, is a snapshot taken at opening time, so
// later changes to the database aren't visible. When the database changes
// while being copied, OpenFS retries copying a few times, finally using the
// last copy anyway: SQLite ignores an incompletely copied WAL tail, and a badly
// torn database most likely fails opening. The copy is placed in [os.TempDir]
// and only removed when calling [AppEngineDB.Close]; copies left behind by
// crashed processes get removed when opening the first database.
func OpenFS(fsys fs.FS, pathname string) (*AppEngineDB, error) {
	return open(fsys, pathname)
}

// ErrInvalidDBName indicates a database name with characters other than ASCII
// letters and digits, “.”, “-”, and “_”, or containing “..”.
var ErrInvalidDBName = errors.New("invalid database name")

var validDBName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// checkDBName returns an error wrapping ErrInvalidDBName if the specified
// database name could cause path traversal or set SQLite driver options.
func checkDBName(dbname string) error {
	if !validDBName.MatchString(dbname) || strings.Contains(dbname, "..") || dbname == "." {
		return fmt.Errorf("%w: %q", ErrInvalidDBName, dbname)
	}
	return nil
}

// open opens a temporary copy of the SQLite database at the given path (that
// must follow fs.FS semantics and thus not start with "/") on the passed fs.FS.
// Copying keeps SQLite from resolving the “/proc/[PID]/root” magic symlink to
// “/” when opening a database in another container, as well as from creating
// WAL and SHM files next to the original database.
func open(fsys fs.FS, pathname string) (_ *AppEngineDB, err error) {
	sweepStaleTmpDirsOnce()
	tmpDir, err := os.MkdirTemp("", tmpDirPattern())
	if err != nil {
		return nil, fmt.Errorf("unable to open database, reason: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tmpDir)
		}
	}()
	dbpath := filepath.Join(tmpDir, "db")
	if err := copyDB(fsys, pathname, dbpath); err != nil {
		return nil, fmt.Errorf("unable to open database, reason: %w", err)
	}

	return connect(dbpath, tmpDir)
}

// fileState identifies a file's contents by its size and modification time.
type fileState struct{ size, modTime int64 }

// statFile returns the state of the named file, or the zero state if the file
// cannot be stat'ed.
func statFile(fsys fs.FS, name string) fileState {
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return fileState{}
	}
	return fileState{size: info.Size(), modTime: info.ModTime().UnixNano()}
}

// copyDB copies the database and its WAL file, if any, retrying when either
// changed while being copied, but keeping the last copy in any case.
func copyDB(fsys fs.FS, pathname, dbpath string) error {
	for attempt := 1; ; attempt++ {
		dbBefore, walBefore := statFile(fsys, pathname), statFile(fsys, pathname+"-wal")
		if err := copyFile(fsys, pathname, dbpath); err != nil {
			return err
		}
		if err := copyFile(fsys, pathname+"-wal", dbpath+"-wal"); err != nil {
			_ = os.Remove(dbpath + "-wal")
		}
		if attempt == maxCopyAttempts ||
			(statFile(fsys, pathname) == dbBefore && statFile(fsys, pathname+"-wal") == walBefore) {
			return nil
		}
		_ = os.Remove(dbpath)
		_ = os.Remove(dbpath + "-wal")
	}
}

// connect opens the database at dbpath and checks that it actually is a
// database. tmpDir gets removed when closing the database.
func connect(dbpath, tmpDir string) (*AppEngineDB, error) {
	// A URI keeps SQLite from interpreting “?” and “#” in the path.
	db, err := sqlx.Open(dbDriverName, (&url.URL{Scheme: "file", Path: dbpath}).String())
	if err != nil {
		return nil, fmt.Errorf("unable to open database, reason: %w", err)
	}
	// Unlike Ping, this fails for files that aren't databases.
	var schemaVersion int
	if err := db.QueryRow("PRAGMA schema_version").Scan(&schemaVersion); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("unable to open database, reason: %w", err)
	}

	// Install a default mapper function that preserves camelCase, with the
	// first letter always being lowercase.
	db.MapperFunc(FirstLower)

	return &AppEngineDB{
		DB:     db,
		tmpDir: tmpDir,
	}, nil
}

// copyFile copies src from fsys to a newly created (non-existing) dst file.
func copyFile(fsys fs.FS, src, dst string) error {
	srcf, err := fsys.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcf.Close() }()
	dstf, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dstf, srcf); err != nil {
		_ = dstf.Close()
		return err
	}
	return dstf.Close()
}

// Close closes the database connection and ensures to additionally dispose of
// the helper resources required to read from an SQLite database in another
// container.
func (db *AppEngineDB) Close() error {
	db.closeMu.Lock()
	defer db.closeMu.Unlock()
	err := db.DB.Close()
	if db.tmpDir != "" {
		_ = os.RemoveAll(db.tmpDir)
		db.tmpDir = ""
	}
	return err
}
