// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"fmt"
	"os"
	"time"

	"github.com/ory/dockertest/v3/docker"
	"github.com/thediveo/lxkns/model"
	"github.com/thediveo/lxkns/ops/mountineer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("IED runtime", func() {

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

	It("doesn't crash when IED runtime isn't present", func() {
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

		Expect(edgeCoreContainerPID()).Error().To(HaveOccurred())
	})

	It("finds the IED runtime", func() {
		By("looking for the IED runtime's PID...")
		pid := Successful(edgeCoreContainerPID())
		Expect(pid).NotTo(BeZero())

		By("...and container canary file")
		corefs := Successful(mountineer.New(model.NamespaceRef{fmt.Sprintf("/proc/%d/ns/mnt", pid)}, nil))
		defer corefs.Close()
		Expect(corefs.ReadFile("/canary")).To(
			WithTransform(func(actual []byte) string { return string(actual) }, Equal("HOLA\n")))
	})

})
