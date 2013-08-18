// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner

import (
	"testing"
)

func TestMatchers(t *testing.T) {
	// Just basic checks, not exhaustive at all.
	checkMatch := func(tt tokenType, s string) {
		r := matchers[tt]
		if !r.MatchString(s) {
			t.Errorf("did not match: %s", s)
		}
	}

	checkMatch(TokenIdent, "abcd")
	checkMatch(TokenString, "\"abcd\"")
	checkMatch(TokenString, "'abcd'")
	checkMatch(TokenHash, "#name")
	checkMatch(TokenNumber, "42'")
	checkMatch(TokenNumber, "4.2'")
	checkMatch(TokenNumber, ".42'")
	checkMatch(TokenPercentage, "42%")
	checkMatch(TokenPercentage, "4.2%")
	checkMatch(TokenPercentage, ".42%")
	checkMatch(TokenDimension, "42px")
	checkMatch(TokenURI, "url('http://www.google.com/')")
	checkMatch(TokenUnicodeRange, "U+0042")
	checkMatch(TokenCDO, "<!--")
	checkMatch(TokenCDC, "-->")
	checkMatch(TokenS, "   \n   \t   \n")
	checkMatch(TokenComment, "/* foo */")
	checkMatch(TokenFunction, "bar(")
	checkMatch(TokenIncludes, "~=")
	checkMatch(TokenDashMatch, "|=")
	checkMatch(TokenPrefixMatch, "^=")
	checkMatch(TokenSuffixMatch, "$=")
	checkMatch(TokenSubstringMatch, "*=")
	checkMatch(TokenChar, "{")
	checkMatch(TokenBOM, "\uFEFF")
}
