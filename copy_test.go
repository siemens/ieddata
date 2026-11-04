// (c) Siemens AG 2026
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

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
					Data: Successful(os.ReadFile(fixtureDB)),
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

	It("copies into a temporary directory identifying this process", func() {
		parentDir := Successful(os.MkdirTemp("", "ieddata-test-"))
		DeferCleanup(os.RemoveAll, parentDir)
		tmpDir := filepath.Join(parentDir, "with?query#and-fragment")
		Expect(os.Mkdir(tmpDir, 0o700)).To(Succeed())
		GinkgoT().Setenv("TMPDIR", tmpDir)

		db := Successful(OpenFS(os.DirFS(filepath.Dir(fixtureDB)), filepath.Base(fixtureDB)))
		Expect(db.Apps()).To(HaveLen(4))
		Expect(Allright(parseTmpDirName(filepath.Base(db.tmpDir)))).To(Equal(Successful(selfIdentity())))

		copyDir := db.tmpDir
		Expect(db.Close()).To(Succeed())
		Expect(copyDir).NotTo(BeAnExistingFile())
	})

	DescribeTable("leaves no temporary directory behind when copying fails",
		func(fsys fs.FS, expectedErr string) {
			tmpDir := Successful(os.MkdirTemp("", "ieddata-test-"))
			DeferCleanup(os.RemoveAll, tmpDir)
			GinkgoT().Setenv("TMPDIR", tmpDir)

			Expect(OpenFS(fsys, "test.db")).Error().To(MatchError(SatisfyAll(
				ContainSubstring("unable to open database"),
				ContainSubstring(expectedErr))))
			Expect(os.ReadDir(tmpDir)).To(BeEmpty())
		},
		Entry("missing database", fstest.MapFS{}, "file does not exist"),
		Entry("read failure", failingFS{fstest.MapFS{"test.db": &fstest.MapFile{Data: []byte("x")}}}, "read failure"),
		Entry("not a database", fstest.MapFS{"test.db": &fstest.MapFile{Data: []byte("not a database")}}, "not a database"),
	)

	It("fails when the temporary directory cannot be created", func() {
		GinkgoT().Setenv("TMPDIR", "/nonexisting-tmpdir")
		Expect(OpenFS(fstest.MapFS{}, "test.db")).Error().To(MatchError(SatisfyAll(
			ContainSubstring("unable to open database"),
			ContainSubstring("/nonexisting-tmpdir"))))
	})

})
