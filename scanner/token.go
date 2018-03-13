// Copyright 2018 Kane York.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner

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

// Token represents a token in the CSS syntax.
type Token struct {
	Type   TokenType
	String string
	// Extra data for the token beyond a simple string.
	// Will always be a pointer to a "Token*Extra" type in this package.
	Extra TokenExtra
}

// The complete list of tokens in CSS Syntax Level 3.
const (
	// Scanner flags.
	TokenError tokenType = iota
	TokenEOF
	// From now on, only tokens from the CSS specification.
	TokenIdent
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
	TokenFunction
	TokenIncludes
	TokenDashMatch
	TokenPrefixMatch
	TokenSuffixMatch
	TokenSubstringMatch
	TokenColumn
	TokenDelim
	// Error tokens
	TokenBadString
	TokenBadURI
	TokenBadEscape // a '\' right before a newline
	// Single-character tokens
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
	TokenBOM:            "BOM",
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
	NonInteger bool
	Dimension  string
}

func (e *TokenExtraNumeric) String() string {
	if e == nil {
		return ""
	}
	if e.Dimension != "" {
		return e.Dimension
	}
	return ""
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
		return fmt.Sprintf("%0X", e.Start)
	} else {
		return fmt.Sprintf("%0X-%0X", e.Start, e.End)
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
