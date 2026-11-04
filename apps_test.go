// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"os"
	"path/filepath"
	"time"

	"github.com/thediveo/lxkns/model"
	"github.com/thediveo/procfsroot/wormholes"

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

	// Use a local test database, so we don't need to rely on an (fake) edge
	// core running.
	DescribeTable("reads installed app information",
		func(opener func(dbpath string) (*AppEngineDB, error)) {
			db := Successful(opener(Successful(filepath.Abs(fixtureDB))))
			defer func() { _ = db.Close() }()

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
		},
		Entry("OpenPathInPID", func(dbpath string) (*AppEngineDB, error) {
			return OpenPathInPID(dbpath, model.PIDType(os.Getpid()))
		}),
		Entry("OpenFS", func(dbpath string) (*AppEngineDB, error) {
			return OpenFS(wormholes.New(os.Getpid()), dbpath[1:])
		}),
	)

	It("handles NULL columns and uses the application's dates", func() {
		db := openModifiedFixture(
			`UPDATE application SET
				icon=NULL, companyWebAddress=NULL, runasservice=NULL, projectId=NULL`,
			`UPDATE applicationversions SET
				appVersion=NULL, composerFilePath=NULL, restRedirectUrl=NULL,
				redirectSection=NULL, metadata=NULL, serviceLabels=NULL,
				createdDate='2000-01-02 03:04:05',
				modifiedDate='2000-01-02 03:04:05'`)

		t0 := Successful(time.Parse("2006-01-02", "2021-01-01"))
		Expect(db.Apps()).To(SatisfyAll(
			HaveLen(4),
			HaveEach(SatisfyAll(
				HaveField("Id", Not(BeZero())),
				HaveField("IconPath", BeZero()),
				HaveField("CompanyURL", BeZero()),
				HaveField("RunAsService", BeFalse()),
				HaveField("ProjectId", BeZero()),
				HaveField("Version", BeZero()),
				HaveField("ServiceLabels", BeZero()),
				// the versions' dates have been set to 2000.
				HaveField("Created", BeTemporally(">=", t0)),
				HaveField("Modified", BeTemporally(">=", t0)),
			))))
	})

	It("tolerates missing and additional columns", func() {
		db := openModifiedFixture(
			"ALTER TABLE applicationversions DROP COLUMN serviceLabels",
			"ALTER TABLE application ADD COLUMN futureColumn TEXT DEFAULT 'future'")
		Expect(db.Apps()).To(SatisfyAll(
			HaveLen(4),
			HaveEach(SatisfyAll(
				HaveField("Id", Not(BeZero())),
				HaveField("Title", Not(BeZero())),
				HaveField("ServiceLabels", BeZero()),
			))))
	})

	DescribeTable("reports errors",
		func(expectedErr string, stmts ...string) {
			Expect(openModifiedFixture(stmts...).Apps()).Error().To(
				MatchError(ContainSubstring(expectedErr)))
		},
		Entry("for the wrong database", "no such table: application",
			"DROP TABLE application"),
		Entry("for mismatching column types", "sortWeight",
			"UPDATE application SET sortWeight='heavy'"),
		Entry("for empty app identifiers", "empty IE App identifier",
			"UPDATE applicationversions SET appId='' WHERE appId=(SELECT min(appId) FROM application)",
			"UPDATE application SET appId='' WHERE appId=(SELECT min(appId) FROM application)"),
	)

})
