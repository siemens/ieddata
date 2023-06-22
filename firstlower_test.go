// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IED app engine database column names", func() {

	It("lower-cases only the first letter", func() {
		Expect(FirstLower("FooBar")).To(Equal("fooBar"))
		Expect(FirstLower("1FooBar")).To(Equal("1FooBar"))
		Expect(FirstLower("fooBar")).To(Equal("fooBar"))
		Expect(FirstLower("")).To(Equal(""))
	})

})
