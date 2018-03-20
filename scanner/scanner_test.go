// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner

import (
	"reflect"
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
			var tExtra TokenExtra
			if TokenExtraTypeLookup[tt] != nil {
				tExtra = ttList[i+2].(TokenExtra)
			}
			if tok := tz.Next(); tok.Type != tt {
				t.Errorf("did not match: %s (got %v, wanted %v)", s, tok, tt)
			} else if tok.Value != tVal {
				t.Errorf("did not match: %s (got %s, wanted %s): %v", s, tok.Value, tVal, tok)
			} else if tExtra != nil && !reflect.DeepEqual(tok.Extra, tExtra) {
				if tt.StopToken() && tt != TokenError && tt != TokenEOF {
					// mismatch ok
				} else {
					t.Errorf("did not match .Extra: %s (got %#v, wanted %#v): %v", s, tok.Extra, tExtra, tok)
				}
			}

			i += 2
			if TokenExtraTypeLookup[tt] != nil {
				i++
			}
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
	checkMatch("#name", TokenHash, "name", &TokenExtraHash{IsIdentifier: true})
	checkMatch("##name", TokenDelim, "#", TokenHash, "name", &TokenExtraHash{IsIdentifier: true})
	checkMatch("42''", TokenNumber, "42", &TokenExtraNumeric{}, TokenString, "")
	checkMatch("+42", TokenNumber, "+42", &TokenExtraNumeric{})
	checkMatch("-42", TokenNumber, "-42", &TokenExtraNumeric{})
	checkMatch("42.", TokenNumber, "42", &TokenExtraNumeric{}, TokenDelim, ".")
	checkMatch("42.0", TokenNumber, "42.0", &TokenExtraNumeric{NonInteger: true})
	checkMatch("4.2", TokenNumber, "4.2", &TokenExtraNumeric{NonInteger: true})
	checkMatch(".42", TokenNumber, ".42", &TokenExtraNumeric{NonInteger: true})
	checkMatch("+.42", TokenNumber, "+.42", &TokenExtraNumeric{NonInteger: true})
	checkMatch("-.42", TokenNumber, "-.42", &TokenExtraNumeric{NonInteger: true})
	checkMatch("42%", TokenPercentage, "42", &TokenExtraNumeric{})
	checkMatch("4.2%", TokenPercentage, "4.2", &TokenExtraNumeric{NonInteger: true})
	checkMatch(".42%", TokenPercentage, ".42", &TokenExtraNumeric{NonInteger: true})
	checkMatch("42px", TokenDimension, "42", &TokenExtraNumeric{Dimension: "px"}) // TODO check the dimension stored in .Extra
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
	checkMatch("U+0042", TokenUnicodeRange, "U+0042", &TokenExtraUnicodeRange{Start: 0x42, End: 0x42})
	checkMatch("U+FFFFFF", TokenUnicodeRange, "U+FFFFFF", &TokenExtraUnicodeRange{Start: 0xFFFFFF, End: 0xFFFFFF})
	checkMatch("U+??????", TokenUnicodeRange, "U+0000-FFFFFF", &TokenExtraUnicodeRange{Start: 0, End: 0xFFFFFF})
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
		TokenNumber, "255", &TokenExtraNumeric{}, TokenComma, ",", TokenS, " ",
		TokenNumber, "0", &TokenExtraNumeric{}, TokenComma, ",", TokenS, " ",
		TokenNumber, "127", &TokenExtraNumeric{}, TokenCloseParen, ")",
		TokenSemicolon, ";", TokenS, " ",
		TokenCloseBrace, "}",
	)
	// Fuzzing results
	checkMatch("ur(0", TokenFunction, "ur", TokenNumber, "0", &TokenExtraNumeric{})
	checkMatch("1\\15", TokenDimension, "1", &TokenExtraNumeric{Dimension: "\x15"})
	checkMatch("url(0t')", TokenBadURI, "0t", &TokenExtraError{})
	checkMatch("uri/", TokenIdent, "uri", TokenDelim, "/")
	checkMatch("\x00", TokenIdent, "\uFFFD")
	checkMatch("a\\0", TokenIdent, "a\uFFFD")
	checkMatch("b\\\\0", TokenIdent, "b\\0")
	checkMatch("00\\d", TokenDimension, "00", &TokenExtraNumeric{Dimension: "\r"})
	// note: \f is form feed, which is 0x0C
	checkMatch("\\0\\0\\C\\\f\\\\0",
		TokenIdent, "\uFFFD\uFFFD\x0C\x0C\\0")
	// String running to EOF is success, not badstring
	checkMatch("\"a0\\d", TokenString, "a0\x0D")
	checkMatch("\"a0\r", TokenBadString, "a0", &TokenExtraError{}, TokenS, "\n")
}
