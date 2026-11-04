// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("device info", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Filedescriptors).Within(2 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeakedFds(goodfds))
		})
	})

	It("reports an error for the wrong database", func() {
		Expect(openModifiedFixture("DROP TABLE device").DeviceInfo()).Error().To(
			MatchError(ContainSubstring("no such table: device")))
	})

	When("accessing an IED", func() {

		BeforeEach(func() {
			if os.Getuid() != 0 {
				Skip("needs root")
			}
		})

		It("accesses device information", func() {
			db := Successful(Open(PlatformBoxDb))
			defer func() { _ = db.Close() }()

			Expect(db.DeviceInfo()).To(SatisfyAll(
				HaveKeyWithValue("deviceName", "iedx12345"),
				HaveKeyWithValue("ownerEmail", "foo.bar@example.com")))
		})

	})

})
