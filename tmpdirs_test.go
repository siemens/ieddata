// (c) Siemens AG 2026
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var _ = Describe("temporary database copy directories", func() {

	var self procIdentity
	var dir string

	BeforeEach(func() {
		self = Successful(selfIdentity())
		dir = Successful(os.MkdirTemp("", "ieddata-test-"))
		DeferCleanup(os.RemoveAll, dir)
	})

	// mkdir creates a temporary directory with the specified name and some
	// content, returning the directory's path.
	mkdir := func(name string) string {
		GinkgoHelper()
		path := filepath.Join(dir, name)
		Expect(os.Mkdir(path, 0o700)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(path, "db"), nil, 0o600)).To(Succeed())
		return path
	}

	It("parses directory names", func() {
		Expect(Allright(parseTmpDirName("ieddata-4026531836-42-123456-987"))).To(
			Equal(procIdentity{pidns: "4026531836", pid: "42", starttime: "123456"}))
	})

	DescribeTable("rejects unrelated directory names",
		func(name string) {
			_, ok := parseTmpDirName(name)
			Expect(ok).To(BeFalse())
		},
		Entry(nil, "foo"),
		Entry(nil, "ieddata-"),
		Entry(nil, "ieddata-1-2-3"),
		Entry(nil, "ieddata-1-2-3-4-5"),
		Entry(nil, "ieddata-x-2-3-4"),
		Entry(nil, "ieddata-1-..-3-4"),
		Entry(nil, "temp-db-copy-1-2-3-4"),
	)

	It("removes only stale directories of the same PID namespace", func() {
		own := mkdir(tmpDirPrefix + self.pidns + "-" + self.pid + "-" + self.starttime + "-1")
		otherNamespace := mkdir(tmpDirPrefix + self.pidns + "0-" + self.pid + "-0-2")
		unrelated := mkdir("foo")
		reusedPID := mkdir(tmpDirPrefix + self.pidns + "-" + self.pid + "-" + self.starttime + "0-3")
		// beyond the maximum possible pid_max of 2^22
		deadPID := mkdir(tmpDirPrefix + self.pidns + "-4194305-" + self.starttime + "-4")
		notADir := filepath.Join(dir, tmpDirPrefix+self.pidns+"-4194305-0-5")
		Expect(os.WriteFile(notADir, nil, 0o600)).To(Succeed())
		alive := mkdir(tmpDirPrefix + self.pidns + "-1-" + Successful(processStartTime("1")) + "-6")

		sweepStaleTmpDirs(dir, self)

		Expect(own).To(BeADirectory())
		Expect(otherNamespace).To(BeADirectory())
		Expect(unrelated).To(BeADirectory())
		Expect(notADir).To(BeARegularFile())
		Expect(alive).To(BeADirectory())
		Expect(reusedPID).NotTo(BeAnExistingFile())
		Expect(deadPID).NotTo(BeAnExistingFile())
	})

	It("doesn't remove stale directories of other users", func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
		foreign := mkdir(tmpDirPrefix + self.pidns + "-4194305-" + self.starttime + "-1")
		Expect(os.Chown(foreign, 65534, 65534)).To(Succeed())

		sweepStaleTmpDirs(dir, self)

		Expect(foreign).To(BeADirectory())
	})

	It("ignores a missing directory", func() {
		Expect(func() { sweepStaleTmpDirs(filepath.Join(dir, "nonexisting"), self) }).NotTo(Panic())
	})

})
