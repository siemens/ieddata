// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("IED app engine database", func() {

	BeforeEach(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
		goodgos := Goroutines()
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			// There's a whale watcher in the background needing to wind, so we
			// give it a chance to get scheduled and run its course.
			Eventually(Goroutines).WithTimeout(2 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	Context("ensure that the sqlite database driver infrastructure is working", func() {

		DescribeTable("opening db via different paths",
			func(fn func() (*sqlx.DB, error)) {
				db, err := fn()
				Expect(err).NotTo(HaveOccurred())
				defer func() { _ = db.Close() }()
				Expect(db.Ping()).To(Succeed())
				appdb := &AppEngineDB{DB: db}
				apps := Successful(appdb.Apps())
				Expect(apps).To(ContainElements(
					And(HaveField("Title", "AppA"),
						HaveField("RepositoryName", "aaa")),
					And(HaveField("Title", "AppC"),
						HaveField("RepositoryName", "ccc"))))
			},
			Entry("plain", func() (*sqlx.DB, error) {
				return sqlx.Open(dbDriverName, "tests/sqlite-alpine-appengine-db/test-apps-and-device.db")
			}),
			Entry("via /proc/$PID/root", func() (*sqlx.DB, error) {
				dbpath := path.Clean(fmt.Sprintf("/proc/%d/root/", os.Getpid()) +
					Successful(unix.Getwd()) +
					"/tests/sqlite-alpine-appengine-db/test-apps-and-device.db")
				return sqlx.Open(dbDriverName, dbpath)
			}),
		)

	})

	It("sanitizes database names", func() {
		Expect(sanitize("abcxyz.0-9")).To(Equal("abcxyz.0-9"))
		Expect(sanitize("../abc?foo=bar")).To(Equal("__abc_foo_bar"))
	})

	It("fails for invalid container PID", func() {
		Expect(OpenInPID("foo.db", 0)).Error().To(MatchError(MatchRegexp(`no such file or directory`)))
	})

	It("fails for missing IE runtime container", func(ctx context.Context) {
		Expect(fakecore.Rename(ctx, "off-"+EdgeIotCoreContainerName)).To(Succeed())
		DeferCleanup(func(ctx context.Context) {
			Expect(fakecore.Rename(ctx, EdgeIotCoreContainerName)).To(Succeed())
		})

		Expect(Open("foo.db")).Error().To(MatchError(MatchRegexp(`no .* runtime container`)))
	})

	It("fails for missing/invalid IED app engine database", func() {
		Expect(Open("foo.db")).Error().To(MatchError(ContainSubstring("/root/data")))
		Expect(Open("not.a.db")).Error().To(MatchError(ContainSubstring("unable to open database")))
	})

	It("accesses the app engine database", func() {
		db := Successful(Open(PlatformBoxDb))
		defer func() { _ = db.Close() }()

		rows := Successful(db.Query("SELECT deviceKey, deviceValue from device"))
		defer func() { Expect(rows.Close()).To(Succeed()) }()
		m := map[string]string{}
		for rows.Next() {
			var key, value string
			Expect(rows.Scan(&key, &value)).To(Succeed())
			m[key] = value
		}
		Expect(m).To(HaveKeyWithValue("deviceName", "iedx12345"))
		Expect(m).To(HaveKeyWithValue("ownerEmail", "foo.bar@example.com"))
	})

})
