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

		Fuzz([]byte(s))
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
	checkMatch("42px", TokenDimension, "42") // TODO check the dimension stored in .Extra
	checkMatch("url(http://domain.com)", TokenURI, "http://domain.com")
	checkMatch("url( http://domain.com/uri/between/space )", TokenURI, "http://domain.com/uri/between/space")
	checkMatch("url('http://domain.com/uri/between/single/quote')", TokenURI, "http://domain.com/uri/between/single/quote")
	checkMatch(`url("http://domain.com/uri/between/double/quote")`, TokenURI, `http://domain.com/uri/between/double/quote`)
	checkMatch("url(http://domain.com/?parentheses=%28)", TokenURI, "http://domain.com/?parentheses=%28")
	checkMatch("url( http://domain.com/?parentheses=%28&between=space )", TokenURI, "http://domain.com/?parentheses=%28&between=space")
	checkMatch("url('http://domain.com/uri/(parentheses)/between/single/quote')", TokenURI, "http://domain.com/uri/(parentheses)/between/single/quote")
	checkMatch(`url("http://domain.com/uri/(parentheses)/between/double/quote")`, TokenURI, `http://domain.com/uri/(parentheses)/between/double/quote`)
	checkMatch(`url(http://domain.com/uri/\(bare%20escaped\)/parentheses)`, TokenURI, `http://domain.com/uri/(bare%20escaped)/parentheses`)
	checkMatch("url(http://domain.com/uri/1)url(http://domain.com/uri/2)",
		TokenURI, "http://domain.com/uri/1",
		TokenURI, "http://domain.com/uri/2",
	)
	checkMatch("url(http://domain.com/uri/1) url(http://domain.com/uri/2)",
		TokenURI, "http://domain.com/uri/1",
		TokenS, " ",
		TokenURI, "http://domain.com/uri/2",
	)
	checkMatch("U+0042", TokenUnicodeRange, "U+0042")
	checkMatch("U+FFFFFF", TokenUnicodeRange, "U+FFFFFF")
	checkMatch("U+??????", TokenUnicodeRange, "U+0000-FFFFFF")
	checkMatch("<!--", TokenCDO, "<!--")
	checkMatch("-->", TokenCDC, "-->")
	checkMatch("   \n   \t   \n", TokenS, "\n") // TODO - whitespace preservation
	checkMatch("/**/", TokenComment, "")
	checkMatch("/*foo*/", TokenComment, "foo")
	checkMatch("/* foo */", TokenComment, " foo ")
	checkMatch("bar(", TokenFunction, "bar")
	checkMatch("~=", TokenIncludes, "~=")
	checkMatch("|=", TokenDashMatch, "|=")
	checkMatch("||", TokenColumn, "||")
	checkMatch("^=", TokenPrefixMatch, "^=")
	checkMatch("$=", TokenSuffixMatch, "$=")
	checkMatch("*=", TokenSubstringMatch, "*=")
	checkMatch("{", TokenOpenBrace, "{")
	// checkMatch("\uFEFF", TokenBOM, "\uFEFF")
	checkMatch(`╯︵┻━┻"stuff"`, TokenIdent, "╯︵┻━┻", TokenString, "stuff")

	checkMatch("foo { bar: rgb(255, 0, 127); }",
		TokenIdent, "foo", TokenS, " ",
		TokenOpenBrace, "{", TokenS, " ",
		TokenIdent, "bar", TokenColon, ":", TokenS, " ",
		TokenFunction, "rgb",
		TokenNumber, "255", TokenComma, ",", TokenS, " ",
		TokenNumber, "0", TokenComma, ",", TokenS, " ",
		TokenNumber, "127", TokenCloseParen, ")",
		TokenSemicolon, ";", TokenS, " ",
		TokenCloseBrace, "}",
	)
}
