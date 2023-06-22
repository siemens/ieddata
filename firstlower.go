// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import "unicode"

// FirstLower returns a copy of the specified string with only its first
// character in lower case.
func FirstLower(s string) string {
	// Iterating over a string means iterating over its Unicode code points, not
	// over its bytes; see: https://gobyexample.com/range
	for idx, r := range s {
		return string(unicode.ToLower(r)) + s[idx+1:]
	}
	return s
}
