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
	"os"
	"path"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/thediveo/morbyd/v2"
	"github.com/thediveo/morbyd/v2/build"
	"github.com/thediveo/morbyd/v2/exec"
	"github.com/thediveo/morbyd/v2/run"
	"github.com/thediveo/morbyd/v2/session"
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var sess *morbyd.Session
var fakecore *morbyd.Container

// Device info row that the fake core writes only into the WAL.
const walKey, walValue = "walKey", "walValue"

var _ = BeforeSuite(func(ctx context.Context) {
	if os.Getuid() != 0 {
		return
	}

	sess = Successful(morbyd.NewSession(ctx,
		session.WithAutoCleaning("test.ieddata=")))
	DeferCleanup(func(ctx context.Context) {
		sess.Close(ctx)
		sess = nil
	})

	By("building a fake edge core image if necessary")
	const imgref = "ieddata/fake-core"

	Expect(sess.BuildImage(ctx,
		"tests/sqlite-alpine-appengine-db",
		build.WithTag(imgref),
		build.WithBuildArg("APPENGDBPATH="+dbBaseDir),
		build.WithBuildArg("PLATFORMBOXDBNAME="+PlatformBoxDb),
	)).Error().NotTo(HaveOccurred(), "image build failed")

	By(fmt.Sprintf("deploying a fake edge core image %q", imgref))
	fakecore = Successful(sess.Run(ctx,
		imgref, run.WithName(EdgeIotCoreContainerName)))
	DeferCleanup(func(ctx context.Context) {
		fakecore.Kill(ctx)
	})
	Expect(fmt.Sprintf("/proc/%d/root/%s/%s",
		Successful(fakecore.PID(ctx)), dbBaseDir, PlatformBoxDb)).To(BeAnExistingFile())

	piper, pipew := io.Pipe()
	cmdsess := Successful(fakecore.Exec(ctx,
		exec.Command("sqlite3", path.Join(dbBaseDir, PlatformBoxDb)),
		exec.WithCombinedOutput(GinkgoWriter),
		exec.WithTTY(),
		exec.WithInput(piper)))
	DeferCleanup(func(ctx context.Context) {
		_ = pipew.Close()
		_ = piper.Close()
		_ = unix.Kill(Successful(cmdsess.PID(ctx)), unix.SIGINT)
	})
	Expect(fmt.Fprint(pipew, "PRAGMA journal_mode=WAL;\n")).Error().To(Succeed())
	Eventually(func() string {
		return fmt.Sprintf("/proc/%d/root/%s/%s", Successful(fakecore.PID(ctx)), dbBaseDir, PlatformBoxDb+"-wal")
	}).
		Within(5 * time.Second).ProbeEvery(100 * time.Millisecond).To(BeAnExistingFile())
	Expect(fmt.Fprintf(pipew, "INSERT INTO device(deviceKey, deviceValue) VALUES('%s', '%s');\n",
		walKey, walValue)).Error().To(Succeed())
})

func TestIEDData(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ieddata package")
}

// fixtureDB is the test database with apps and device information.
const fixtureDB = "tests/sqlite-alpine-appengine-db/test-apps-and-device.db"

// openModifiedFixture opens a copy of the test database after executing the
// specified SQL statements on it, closing it automatically at the end of the
// current spec.
func openModifiedFixture(stmts ...string) *AppEngineDB {
	GinkgoHelper()
	dir := Successful(os.MkdirTemp("", "ieddata-test-"))
	DeferCleanup(os.RemoveAll, dir)
	dbpath := filepath.Join(dir, "test.db")
	Expect(os.WriteFile(dbpath, Successful(os.ReadFile(fixtureDB)), 0o600)).To(Succeed())

	db := Successful(sqlx.Open(dbDriverName, dbpath))
	for _, stmt := range stmts {
		Expect(db.Exec(stmt)).Error().NotTo(HaveOccurred(), stmt)
	}
	Expect(db.Close()).To(Succeed())

	appdb := Successful(OpenFS(os.DirFS(dir), "test.db"))
	DeferCleanup(appdb.Close)
	return appdb
}

// changingFS simulates a database that gets modified while being copied, for
// the specified number of times.
type changingFS struct {
	fstest.MapFS
	changes int
	opens   int
}

func (c *changingFS) Open(name string) (fs.File, error) {
	if name == "test.db" {
		c.opens++
		if c.changes > 0 {
			c.changes--
			c.MapFS[name].ModTime = c.MapFS[name].ModTime.Add(time.Second)
		}
	}
	return c.MapFS.Open(name)
}

// failingFS returns files that fail reading.
type failingFS struct{ fstest.MapFS }

func (f failingFS) Open(name string) (fs.File, error) {
	file, err := f.MapFS.Open(name)
	if err != nil {
		return nil, err
	}
	return failingFile{file}, nil
}

type failingFile struct{ fs.File }

func (failingFile) Read([]byte) (int, error) { return 0, errors.New("read failure") }
