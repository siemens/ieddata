// (c) Siemens AG 2026
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"io/fs"
	"os"
	"testing/fstest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

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

var _ = Describe("copying databases", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	DescribeTable("retries copying a changing database",
		func(changes, expectedOpens int) {
			fsys := &changingFS{
				MapFS: fstest.MapFS{"test.db": &fstest.MapFile{
					Data: Successful(os.ReadFile("tests/sqlite-alpine-appengine-db/test-apps-and-device.db")),
				}},
				changes: changes,
			}
			db := Successful(OpenFS(fsys, "test.db"))
			defer func() { _ = db.Close() }()
			Expect(fsys.opens).To(Equal(expectedOpens))
			Expect(db.Apps()).To(HaveLen(4))
		},
		Entry("unchanged", 0, 1),
		Entry("changed once", 1, 2),
		Entry("changing all the time", 42, maxCopyAttempts),
	)

})
