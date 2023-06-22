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
		if os.Getuid() != 0 {
			Skip("needs root")
		}
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Filedescriptors).Within(2 * time.Second).WithPolling(250 * time.Millisecond).
				ShouldNot(HaveLeakedFds(goodfds))
		})
	})

	It("accesses device information", func() {
		db := Successful(Open("platformbox.db"))
		defer db.Close()

		m, err := db.DeviceInfo()
		Expect(err).NotTo(HaveOccurred())
		Expect(m).To(HaveKeyWithValue("deviceName", "iedx12345"))
		Expect(m).To(HaveKeyWithValue("ownerEmail", "foo.bar@example.com"))
	})

})
