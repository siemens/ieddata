// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"unicode"
	"unicode/utf8"
)

// FirstLower returns a copy of the specified string with only its first
// character in lower case.
func FirstLower(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	return string(unicode.ToLower(r)) + s[size:]
}
