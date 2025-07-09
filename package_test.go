// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/thediveo/morbyd"
	"github.com/thediveo/morbyd/build"
	"github.com/thediveo/morbyd/run"
	"github.com/thediveo/morbyd/session"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var sess *morbyd.Session
var fakecore *morbyd.Container

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
})

func TestIEDData(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ieddata package")
}
