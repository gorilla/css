// Copyright 2018 Kane York.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// TokenType identifies the type of lexical tokens.
type TokenType int

// String returns a string representation of the token type.
func (t TokenType) String() string {
	return tokenNames[t]
}

// Stop tokens are TokenError, TokenEOF, TokenBadEscape,
// TokenBadString, TokenBadURI.  A consumer that does not want to tolerate
// parsing errors should stop parsing when this returns true.
func (t TokenType) StopToken() bool {
	return t == TokenError || t == TokenEOF || t == TokenBadEscape || t ==
		TokenBadString || t == TokenBadURI
}

// Simple tokens TODO figure out a useful definition for this.
func (t TokenType) SimpleToken() bool {
	if t.StopToken() {
		return false
	}
	if t == TokenHash || t == TokenNumber || t == TokenPercentage || t == TokenDimension || t == TokenUnicodeRange {
		return false
	}
	return true
}

// ParseError represents a CSS syntax error.
type ParseError struct {
	Type    TokenType
	Message string
	Loc     int
}

func (e *ParseError) Error() string {
	return e.Message
}

// Token represents a token in the CSS syntax.
type Token struct {
	Type  TokenType
	Value string
	// Extra data for the token beyond a simple string.
	// Will always be a pointer to a "Token*Extra" type in this package.
	Extra TokenExtra
}

// The complete list of tokens in CSS Syntax Level 3.
const (
	// Scanner flags.
	TokenError TokenType = iota
	TokenEOF
	// From now on, only tokens from the CSS specification.
	TokenIdent
	TokenFunction
	TokenDelim // Single character
	TokenAtKeyword
	TokenString
	TokenHash
	TokenNumber
	TokenPercentage
	TokenDimension
	TokenURI
	TokenUnicodeRange
	TokenCDO
	TokenCDC
	// Whitespace
	TokenS
	// CSS Syntax Level 3 removes comments from the token stream, but they are
	// preserved here.
	TokenComment

	// Error tokens
	TokenBadString
	TokenBadURI
	TokenBadEscape // a '\' right before a newline

	// Fixed-string tokens
	TokenIncludes
	TokenDashMatch
	TokenPrefixMatch
	TokenSuffixMatch
	TokenSubstringMatch
	TokenColumn
	TokenColon
	TokenSemicolon
	TokenComma
	TokenOpenBracket
	TokenCloseBracket
	TokenOpenParen
	TokenCloseParen
	TokenOpenBrace
	TokenCloseBrace
)

// backwards compatibility
const TokenChar = TokenDelim

// tokenNames maps tokenType's to their names.  Used for conversion to string.
var tokenNames = map[TokenType]string{
	TokenError:          "error",
	TokenEOF:            "EOF",
	TokenIdent:          "IDENT",
	TokenAtKeyword:      "ATKEYWORD",
	TokenString:         "STRING",
	TokenHash:           "HASH",
	TokenNumber:         "NUMBER",
	TokenPercentage:     "PERCENTAGE",
	TokenDimension:      "DIMENSION",
	TokenURI:            "URI",
	TokenUnicodeRange:   "UNICODE-RANGE",
	TokenCDO:            "CDO",
	TokenCDC:            "CDC",
	TokenS:              "S",
	TokenComment:        "COMMENT",
	TokenFunction:       "FUNCTION",
	TokenIncludes:       "INCLUDES",
	TokenDashMatch:      "DASHMATCH",
	TokenPrefixMatch:    "PREFIXMATCH",
	TokenSuffixMatch:    "SUFFIXMATCH",
	TokenSubstringMatch: "SUBSTRINGMATCH",
	TokenDelim:          "DELIM",
	TokenBadString:      "BAD-STRING",
	TokenBadURI:         "BAD-URI",
	TokenBadEscape:      "BAD-ESCAPE",
	TokenColumn:         "COLUMN",
	TokenColon:          "COLON",
	TokenSemicolon:      "SEMICOLON",
	TokenComma:          "COMMA",
	TokenOpenBracket:    "LEFT-BRACKET", // []
	TokenCloseBracket:   "RIGHT-BRACKET",
	TokenOpenParen:      "LEFT-PAREN", // ()
	TokenCloseParen:     "RIGHT-PAREN",
	TokenOpenBrace:      "LEFT-BRACE", // {}
	TokenCloseBrace:     "RIGHT-BRACE",
}

// TokenExtra fills the .Extra field of a token.  Consumers should perform a
// type cast to the proper type to inspect its data.
type TokenExtra interface {
	String() string
}

// TokenExtraTypeLookup provides a handy check for whether a given token type
// should contain extra data.
var TokenExtraTypeLookup = map[TokenType]interface{}{
	TokenError:        &TokenExtraError{},
	TokenBadEscape:    &TokenExtraError{},
	TokenBadString:    &TokenExtraError{},
	TokenBadURI:       &TokenExtraError{},
	TokenHash:         &TokenExtraHash{},
	TokenNumber:       &TokenExtraNumeric{},
	TokenPercentage:   &TokenExtraNumeric{},
	TokenDimension:    &TokenExtraNumeric{},
	TokenUnicodeRange: &TokenExtraUnicodeRange{},
}

// TokenExtraHash is attached to TokenHash.
type TokenExtraHash struct {
	IsIdentifier bool
}

func (e *TokenExtraHash) String() string {
	if e == nil || !e.IsIdentifier {
		return "unrestricted"
	} else {
		return "id"
	}
}

// TokenExtraNumeric is attached to TokenNumber, TokenPercentage, and
// TokenDimension.
type TokenExtraNumeric struct {
	// Value float64 // omitted from this implementation
	NonInteger bool
	Dimension  string
}

func (e *TokenExtraNumeric) String() string {
	if e == nil {
		return ""
	}
	return e.Dimension
}

// TokenExtraUnicodeRange is attached to a TokenUnicodeRange.
type TokenExtraUnicodeRange struct {
	Start rune
	End   rune
}

func (e *TokenExtraUnicodeRange) String() string {
	if e == nil {
		panic("TokenExtraUnicodeRange: unexpected nil pointer value")
	}

	if e.Start == e.End {
		return fmt.Sprintf("U+%04X", e.Start)
	} else {
		return fmt.Sprintf("U+%04X-%04X", e.Start, e.End)
	}
}

// TokenExtraError is attached to a TokenError and contains the same value as
// Tokenizer.Err(). See also the ParseError type and ParseError.Recoverable().
type TokenExtraError struct {
	Err error
}

// String returns the error text.
func (e *TokenExtraError) String() string {
	return e.Err.Error()
}

// Error implements error.
func (e *TokenExtraError) Error() string {
	return e.Err.Error()
}

// Cause implements errors.Causer.
func (e *TokenExtraError) Cause() error {
	return e.Err
}

// Returns the ParseError object, if present.
func (e *TokenExtraError) ParseError() *ParseError {
	pe, ok := e.Err.(*ParseError)
	if !ok {
		return nil
	}
	return pe
}

func escapeIdentifier(s string) string { return escapeIdent(s, 0) }
func escapeHashName(s string) string   { return escapeIdent(s, 1) }
func escapeDimension(s string) string  { return escapeIdent(s, 2) }

func needsHexEscaping(c byte, mode int) bool {
	if c < 0x20 {
		return true
	}
	if c >= utf8.RuneSelf {
		return false
	}
	if mode == 2 {
		if c == 'e' || c == 'E' {
			return true
		}
	}
	if c == '\\' {
		return true
	}
	if isNameCode(c) {
		return false
	}
	return true
}

func escapeIdent(s string, mode int) string {
	if s == "" {
		return ""
	}
	var buf bytes.Buffer
	buf.Grow(len(s))
	anyChanges := false

	var i int

	// Handle first character
	// dashes allowed at start only for TokenIdent-ish
	// eE not allowed at start for Dimension
	if mode != 1 {
		if !isNameStart(s[0]) && s[0] != '-' && s[0] != 'e' && s[0] != 'E' {
			if needsHexEscaping(s[0], mode) {
				fmt.Fprintf(&buf, "\\%X ", s[0])
				anyChanges = true
			} else {
				buf.WriteByte('\\')
				buf.WriteByte(s[0])
				anyChanges = true
			}
		} else if s[0] == 'e' || s[0] == 'E' {
			if mode == 2 {
				fmt.Fprintf(&buf, "\\%X ", s[0])
				anyChanges = true
			} else {
				buf.WriteByte(s[0])
			}
		} else if s[0] == '-' {
			if len(s) == 1 {
				return "\\-"
			} else if isNameStart(s[1]) {
				buf.WriteByte('-')
			} else {
				buf.WriteString("\\-")
				anyChanges = true
			}
		} else {
			buf.WriteByte(s[0])
		}
		i = 1
	} else {
		i = 0
	}
	// Write the rest of the name
	for ; i < len(s); i++ {
		if !isNameCode(s[i]) {
			fmt.Fprintf(&buf, "\\%X ", s[i])
			anyChanges = true
		} else {
			buf.WriteByte(s[i])
		}
	}

	if !anyChanges {
		return s
	}
	return buf.String()
}

func escapeString(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			buf.WriteString("\\\"")
			continue
		case '\n':
			buf.WriteString("\\0A ")
			continue
		case '\r':
			buf.WriteString("\\0D ")
			continue
		case '\\':
			buf.WriteString("\\\\")
			continue
		}
		if s[i] < utf8.RuneSelf && isNonPrintable(s[i]) {
			fmt.Fprintf(&buf, "\\%X", s[i])
			continue
		}
		buf.WriteByte(s[i])
	}
	buf.WriteByte('"')
	return buf.String()
}

func (t *Token) Render() string {
	var buf bytes.Buffer
	t.WriteTo(&buf)
	return buf.String()
}

func (t *Token) WriteTo(w io.Writer) {
	switch t.Type {
	case TokenError:
		return
	case TokenEOF:
		return
	case TokenIdent:
		fmt.Fprint(w, escapeIdentifier(t.Value))
	case TokenAtKeyword:
		fmt.Fprint(w, "@", escapeIdentifier(t.Value))
	case TokenDelim:
		if t.Value == "\\" {
			fmt.Fprint(w, "\\\n")
		} else {
			fmt.Fprint(w, t.Value)
		}
	case TokenHash:
		e := t.Extra.(*TokenExtraHash)
		io.WriteString(w, "#")
		if e.IsIdentifier {
			fmt.Fprint(w, escapeIdentifier(t.Value))
		} else {
			fmt.Fprint(w, escapeHashName(t.Value))
		}
	case TokenPercentage:
		fmt.Fprint(w, t.Value, "%")
	case TokenDimension:
		e := t.Extra.(*TokenExtraNumeric)
		fmt.Fprint(w, t.Value, escapeDimension(e.Dimension))
	case TokenString:
		io.WriteString(w, escapeString(t.Value))
	case TokenURI:
		io.WriteString(w, "url(")
		io.WriteString(w, escapeString(t.Value))
		io.WriteString(w, ")")
	case TokenUnicodeRange:
		io.WriteString(w, t.Extra.String())
	case TokenComment:
		io.WriteString(w, "/*")
		io.WriteString(w, t.Value)
		io.WriteString(w, "*/")
	case TokenFunction:
		io.WriteString(w, t.Value)
		io.WriteString(w, "(")

	case TokenBadEscape:
		io.WriteString(w, "\\\n")
	case TokenBadString:
		io.WriteString(w, "\"")
		io.WriteString(w, t.Value)
		io.WriteString(w, "\n")
	case TokenBadURI:
		io.WriteString(w, "url(")
		str := escapeString(t.Value)
		str = strings.TrimSuffix(str, "\"")
		io.WriteString(w, str)
		io.WriteString(w, "\n)")
	default:
		fmt.Fprint(w, t.Value)
	}
}

// TokenRenderer takes care of the comment insertion rules for serialization.
type TokenRenderer struct {
	lastToken Token
}

func (r *TokenRenderer) WriteTokenTo(w io.Writer, t Token) {
	var prevKey, curKey interface{}
	if r.lastToken.Type == TokenDelim {
		prevKey = r.lastToken.Value[0]
	} else {
		prevKey = r.lastToken.Type
	}
	if t.Type == TokenDelim {
		curKey = t.Value[0]
	} else {
		curKey = t.Type
	}

	m1, ok := commentInsertionRules[prevKey]
	if ok {
		if m1[curKey] {
			io.WriteString(w, "/**/")
		}
	}

	t.WriteTo(w)
	r.lastToken = t
}

var commentInsertionThruCDC = map[interface{}]bool{
	TokenIdent:        true,
	TokenFunction:     true,
	TokenURI:          true,
	TokenBadURI:       true,
	TokenNumber:       true,
	TokenPercentage:   true,
	TokenDimension:    true,
	TokenUnicodeRange: true,
	TokenCDC:          true,
	'-':               true,
	'(':               false,
}

var commentInsertionRules = map[interface{}]map[interface{}]bool{
	TokenIdent: map[interface{}]bool{
		TokenIdent:        true,
		TokenFunction:     true,
		TokenURI:          true,
		TokenBadURI:       true,
		'-':               true,
		TokenNumber:       true,
		TokenPercentage:   true,
		TokenDimension:    true,
		TokenUnicodeRange: true,
		TokenCDC:          true,
		'(':               true,
	},
	TokenAtKeyword: commentInsertionThruCDC,
	TokenHash:      commentInsertionThruCDC,
	TokenDimension: commentInsertionThruCDC,
	'#': map[interface{}]bool{
		TokenIdent:        true,
		TokenFunction:     true,
		TokenURI:          true,
		TokenBadURI:       true,
		TokenNumber:       true,
		TokenPercentage:   true,
		TokenDimension:    true,
		TokenUnicodeRange: true,
		TokenCDC:          false,
		'-':               true,
		'(':               false,
	},
	'-': map[interface{}]bool{
		TokenIdent:        true,
		TokenFunction:     true,
		TokenURI:          true,
		TokenBadURI:       true,
		TokenNumber:       true,
		TokenPercentage:   true,
		TokenDimension:    true,
		TokenUnicodeRange: true,
		TokenCDC:          false,
		'-':               false,
		'(':               false,
	},
	TokenNumber: map[interface{}]bool{
		TokenIdent:        true,
		TokenFunction:     true,
		TokenURI:          true,
		TokenBadURI:       true,
		TokenNumber:       true,
		TokenPercentage:   true,
		TokenDimension:    true,
		TokenUnicodeRange: true,
		TokenCDC:          false,
		'-':               false,
		'(':               false,
	},
	'@': map[interface{}]bool{
		TokenIdent:        true,
		TokenFunction:     true,
		TokenURI:          true,
		TokenBadURI:       true,
		TokenNumber:       false,
		TokenPercentage:   false,
		TokenDimension:    false,
		TokenUnicodeRange: true,
		TokenCDC:          false,
		'-':               true,
		'(':               false,
	},
	TokenUnicodeRange: map[interface{}]bool{
		TokenIdent:        true,
		TokenFunction:     true,
		TokenNumber:       true,
		TokenPercentage:   true,
		TokenDimension:    true,
		TokenUnicodeRange: false,
		'?':               true,
	},
	'.': map[interface{}]bool{
		TokenNumber:     true,
		TokenPercentage: true,
		TokenDimension:  true,
	},
	'+': map[interface{}]bool{
		TokenNumber:     true,
		TokenPercentage: true,
		TokenDimension:  true,
	},
	'$': map[interface{}]bool{
		'=': true,
	},
	'*': map[interface{}]bool{
		'=': true,
	},
	'^': map[interface{}]bool{
		'=': true,
	},
	'~': map[interface{}]bool{
		'=': true,
	},
	'|': map[interface{}]bool{
		'=': true,
		'|': true,
	},
	'/': map[interface{}]bool{
		'*': true,
	},
}
