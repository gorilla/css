// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner

import (
	"strings"
	"testing"
)

func TestMatchers(t *testing.T) {
	// Just basic checks, not exhaustive at all.
	checkMatch := func(s string, ttList ...interface{}) {
		tz := NewTokenizer(strings.NewReader(s))

		i := 0
		for i < len(ttList) {
			tt := ttList[i].(TokenType)
			tVal := ttList[i+1].(string)
			if tok := tz.Next(); tok.Type != tt || tok.Value != tVal {
				t.Errorf("did not match: %s (got %s, wanted %s): %v", s, tok.Value, tVal, tok)
			}

			i += 2
		}

		if tok := tz.Next(); tok.Type != TokenEOF {
			t.Errorf("missing EOF after token %s, got %+v", s, tok)
			if tok := tz.Next(); tok.Type != TokenEOF {
				t.Errorf("double missing EOF after token %s, got %+v", s, tok)
			}
		}
	}

	checkMatch("abcd", TokenIdent, "abcd")
	checkMatch(`"abcd"`, TokenString, `abcd`)
	checkMatch(`"ab'cd"`, TokenString, `ab'cd`)
	checkMatch(`"ab\"cd"`, TokenString, `ab"cd`)
	checkMatch(`"ab\\cd"`, TokenString, `ab\cd`)
	checkMatch("'abcd'", TokenString, "abcd")
	checkMatch(`'ab"cd'`, TokenString, `ab"cd`)
	checkMatch(`'ab\'cd'`, TokenString, `ab'cd`)
	checkMatch(`'ab\\cd'`, TokenString, `ab\cd`)
	checkMatch("#name", TokenHash, "name")
	checkMatch("##name", TokenDelim, "#", TokenHash, "name")
	checkMatch("42''", TokenNumber, "42", TokenString, "")
	checkMatch("+42", TokenNumber, "+42")
	checkMatch("-42", TokenNumber, "-42")
	checkMatch("4.2", TokenNumber, "4.2")
	checkMatch(".42", TokenNumber, ".42")
	checkMatch("+.42", TokenNumber, "+.42")
	checkMatch("-.42", TokenNumber, "-.42")
	checkMatch("42%", TokenPercentage, "42")
	checkMatch("4.2%", TokenPercentage, "4.2")
	checkMatch(".42%", TokenPercentage, ".42")
	checkMatch("42px", TokenDimension, "42px")
	checkMatch("url(http://domain.com)", TokenURI, "url(http://domain.com)")
	checkMatch("url( http://domain.com/uri/between/space )", TokenURI, "url( http://domain.com/uri/between/space )")
	checkMatch("url('http://domain.com/uri/between/single/quote')", TokenURI, "url('http://domain.com/uri/between/single/quote')")
	checkMatch(`url("http://domain.com/uri/between/double/quote")`, TokenURI, `url("http://domain.com/uri/between/double/quote")`)
	checkMatch("url(http://domain.com/?parentheses=%28)", TokenURI, "url(http://domain.com/?parentheses=%28)")
	checkMatch("url( http://domain.com/?parentheses=%28&between=space )", TokenURI, "url( http://domain.com/?parentheses=%28&between=space )")
	checkMatch("url('http://domain.com/uri/(parentheses)/between/single/quote')", TokenURI, "url('http://domain.com/uri/(parentheses)/between/single/quote')")
	checkMatch(`url("http://domain.com/uri/(parentheses)/between/double/quote")`, TokenURI, `url("http://domain.com/uri/(parentheses)/between/double/quote")`)
	checkMatch("url(http://domain.com/uri/1)url(http://domain.com/uri/2)",
		TokenURI, "url(http://domain.com/uri/1)",
		TokenURI, "url(http://domain.com/uri/2)",
	)
	checkMatch("U+0042", TokenUnicodeRange, "U+0042")
	checkMatch("<!--", TokenCDO, "<!--")
	checkMatch("-->", TokenCDC, "-->")
	checkMatch("   \n   \t   \n", TokenS, "   \n   \t   \n")
	checkMatch("/* foo */", TokenComment, "/* foo */")
	checkMatch("bar(", TokenFunction, "bar")
	checkMatch("~=", TokenIncludes, "~=")
	checkMatch("|=", TokenDashMatch, "|=")
	checkMatch("^=", TokenPrefixMatch, "^=")
	checkMatch("$=", TokenSuffixMatch, "$=")
	checkMatch("*=", TokenSubstringMatch, "*=")
	checkMatch("{", TokenChar, "{")
	// checkMatch("\uFEFF", TokenBOM, "\uFEFF")
	checkMatch(`╯︵┻━┻"stuff"`, TokenIdent, "╯︵┻━┻", TokenString, `"stuff"`)
}
