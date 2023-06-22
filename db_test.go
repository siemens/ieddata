// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"os"
	"time"

	"github.com/ory/dockertest/v3/docker"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("IED app engine database", func() {

	BeforeEach(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			pool.Client.HTTPClient.CloseIdleConnections()
			// There's a whale watcher in the background needing to wind, so we
			// give it a chance to get scheduled and run its course.
			Eventually(Filedescriptors).WithTimeout(2 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeakedFds(goodfds))
		})
	})

	It("sanitizes database names", func() {
		Expect(sanitize("abcxyz.0-9")).To(Equal("abcxyz.0-9"))
		Expect(sanitize("../abc?foo=bar")).To(Equal("__abc_foo_bar"))
	})

	It("fails for invalid container PID", func() {
		Expect(OpenInPID("foo.db", 0)).Error().To(MatchError(MatchRegexp(`invalid mount namespace reference`)))
	})

	It("fails for missing IE runtime container", func() {
		// now that we have a fake edge core container running for the whole
		// suite we need to move it out of the way for just this test...
		Expect(pool.Client.RenameContainer(docker.RenameContainerOptions{
			ID:   fakecore.Container.ID,
			Name: "off-" + EdgeIotCoreContainerName,
		})).To(Succeed())
		defer func() {
			Expect(pool.Client.RenameContainer(docker.RenameContainerOptions{
				ID:   fakecore.Container.ID,
				Name: EdgeIotCoreContainerName,
			})).To(Succeed())
		}()

		Expect(Open("foo.db")).Error().To(MatchError(MatchRegexp(`no .* runtime container`)))
	})

	It("fails for missing/invalid IED app engine database", func() {
		Expect(Open("foo.db")).Error().To(MatchError(ContainSubstring("/root/data")))
		Expect(Open("not.a.db")).Error().To(MatchError(ContainSubstring("unable to open database")))
	})

	It("accesses the app engine database", func() {
		var db *AppEngineDB
		Eventually(func() error {
			var err error
			db, err = Open("platformbox.db")
			return err
		}).WithTimeout(10 * time.Second).WithPolling(250 * time.Millisecond).
			Should(Succeed())
		defer db.Close()

		rows := Successful(db.Query("SELECT deviceKey, deviceValue from device"))
		defer rows.Close()
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
