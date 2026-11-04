// (c) Siemens AG 2026
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("database names", func() {

	DescribeTable("accepts valid names",
		func(dbname string) {
			Expect(checkDBName(dbname)).To(Succeed())
		},
		Entry(nil, PlatformBoxDb),
		Entry(nil, "abcxyz.0-9"),
		Entry(nil, "foo_bar.db"),
		Entry(nil, ".hidden"),
	)

	DescribeTable("rejects invalid names",
		func(dbname string) {
			Expect(checkDBName(dbname)).To(MatchError(ErrInvalidDBName))
		},
		Entry(nil, ""),
		Entry(nil, "."),
		Entry(nil, ".."),
		Entry(nil, "../abc"),
		Entry(nil, "foo..db"),
		Entry(nil, "sub/dir.db"),
		Entry(nil, "abc?foo=bar"),
		Entry(nil, "foo#bar"),
		Entry(nil, "föö.db"),
	)

	It("rejects invalid names before opening", func(ctx context.Context) {
		Expect(OpenInPID("../etc/passwd", 0)).Error().To(MatchError(ErrInvalidDBName))
		Expect(OpenCtx(ctx, "../etc/passwd", WithDockerHost("unix:///nonexisting.sock"))).
			Error().To(MatchError(ErrInvalidDBName))
	})

})
