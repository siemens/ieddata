// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"os"
	"path"
	"time"

	"github.com/thediveo/lxkns/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("IED app engine installed apps", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	It("reads installed app information", func() {
		// Use a local test database, so we don't need to rely on an (fake) edge
		// core running.
		cwd := Successful(os.Getwd())
		db := Successful(open(path.Join(cwd, "tests/sqlite-alpine-appengine-db/test-apps-and-device.db"), model.PIDType(os.Getpid())))
		defer db.Close()

		apps := Successful(db.Apps())
		Expect(apps).To(HaveLen(4))
		t0 := Successful(time.Parse("2006-01-02", "2021-01-01"))
		Expect(apps).To(HaveEach(SatisfyAll(
			HaveField("Id", Not(BeZero())),
			HaveField("Version", Not(BeZero())),
			HaveField("VersionId", Not(BeZero())),
			HaveField("ProjectId", Not(BeZero())),
			HaveField("Created.Unix()", BeNumerically(">=", t0.Unix())),
			HaveField("Id", Not(BeZero())),
			HaveField("IconPath", Not(BeZero())),
		)))
		Expect(apps).To(ContainElement(HaveField("IsDebuggingEnabled", 1)))
	})

})
