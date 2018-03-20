// Copyright (c) 2018 Kane York. Licensed under 2-Clause BSD.

package tokenizer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

var (
	errBadEscape = &ParseError{Type: TokenBadEscape, Message: "bad escape (backslash-newline) in input"}
)

// Tokenizer scans an input and emits tokens following the CSS Syntax Level 3
// specification.
type Tokenizer struct {
	r    *bufio.Reader
	err  error
	peek [3]byte

	// ErrorMode int

	tok Token
}

/*
const (
	// Default error mode - tokenization errors are represented as special tokens in the stream, and I/O errors are TokenError.
	ErrorModeTokens = iota
	ErrorModeFatal
)
*/

// Construct a Tokenizer from the given input.  Input need not be 'normalized'
// according to the spec (newlines changed to \n, zero bytes changed to
// U+FFFD).
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		r: bufio.NewReader(transform.NewReader(r, new(normalize))),
	}
}

// Scan for the next token.  If the tokenizer is in an error state, no input
// will be consumed.
func (z *Tokenizer) Scan() {
	defer func() {
		rec := recover()
		if rErr, ok := rec.(error); ok {
			// we only ever panic(err)
			z.err = rErr
			z.tok = Token{
				Type:  TokenError,
				Extra: &TokenExtraError{Err: z.err},
			}
		} else if rec != nil {
			panic(rec)
		}
	}()

	if z.err == nil {
		z.tok = z.consume()
	} else if z.err == io.EOF {
		z.tok = Token{
			Type: TokenEOF,
		}
	} else {
		z.tok = Token{
			Type:  TokenError,
			Value: z.err.Error(),
			Extra: &TokenExtraError{Err: z.err},
		}
	}
}

// Get the most recently scanned token.
func (z *Tokenizer) Token() Token {
	return z.tok
}

// Scan for the next token and return it.
func (z *Tokenizer) Next() Token {
	z.Scan()
	return z.tok
}

// Err returns the last input reading error to be encountered. It is filled
// when TokenError is returned.
func (z *Tokenizer) Err() error {
	return z.err
}

// repeek reads the next 3 bytes into the tokenizer. on EOF, the bytes are
// filled with zeroes.  (Null bytes in the input are preprocessed into U+FFFD.)
func (z *Tokenizer) repeek() {
	by, err := z.r.Peek(3)
	if err != nil && err != io.EOF {
		panic(err)
	}
	copy(z.peek[:], by)

	// zero fill on EOF
	i := len(by)
	for i < 3 {
		z.peek[i] = 0
		i++
	}
}

// §4.3.8
// up to 2 bytes
func isValidEscape(p []byte) bool {
	if len(p) < 2 {
		return false
	}
	if p[0] != '\\' {
		return false
	}
	if p[1] == '\n' {
		return false
	}
	return true
}

// §4.3.9
func isNameStart(p byte) bool {
	if p >= utf8.RuneSelf {
		return true // any high code points
	}
	if p == '_' {
		return true
	}
	if p >= 'A' && p <= 'Z' {
		return true
	}
	if p >= 'a' && p <= 'z' {
		return true
	}
	return false
}

func isNameCode(p byte) bool {
	if p >= utf8.RuneSelf {
		return true // any high code points
	}
	if p == '_' || p == '-' {
		return true
	}
	if p >= 'A' && p <= 'Z' {
		return true
	}
	if p >= 'a' && p <= 'z' {
		return true
	}
	if p >= '0' && p <= '9' {
		return true
	}
	return false
}

func isHexDigit(p byte) bool {
	if p >= 'A' && p <= 'F' {
		return true
	}
	if p >= 'a' && p <= 'f' {
		return true
	}
	if p >= '0' && p <= '9' {
		return true
	}
	return false
}

// up to 3 bytes
func isStartIdentifier(p []byte) bool {
	if p[0] == '-' {
		p = p[1:]
	}
	if isNameStart(p[0]) {
		return true
	} else if isValidEscape(p) {
		return true
	}
	return false
}

// §4.3.10
// up to 3 bytes
func isStartNumber(p []byte) bool {
	if p[0] == '+' || p[0] == '-' {
		p = p[1:]
	}
	if p[0] == '.' {
		p = p[1:]
	}
	if p[0] >= '0' && p[0] <= '9' {
		return true
	}
	return false
}

func isNonPrintable(by byte) bool {
	return (0 <= by && by <= 0x08) || (0x0B == by) || (0x0E <= by && by <= 0x1F) || (0x7F == by)
}

// repeek must be called before the following:

func (z *Tokenizer) nextIsEscape() bool {
	return isValidEscape(z.peek[:2])
}

func (z *Tokenizer) nextStartsIdentifier() bool {
	return isStartIdentifier(z.peek[:3])
}

func (z *Tokenizer) nextIsNumber() bool {
	return isStartNumber(z.peek[:3])
}

func (z *Tokenizer) nextCompare(vs string) bool {
	return string(z.peek[:len(vs)]) == vs
}

var premadeTokens = map[byte]Token{
	'$': Token{
		Type:  TokenSuffixMatch,
		Value: "$=",
	},
	'*': Token{
		Type:  TokenSubstringMatch,
		Value: "*=",
	},
	'^': Token{
		Type:  TokenPrefixMatch,
		Value: "^=",
	},
	'~': Token{
		Type:  TokenIncludes,
		Value: "~=",
	},
	'(': Token{Type: TokenOpenParen, Value: "("},
	')': Token{Type: TokenCloseParen, Value: ")"},
	'[': Token{Type: TokenOpenBracket, Value: "["},
	']': Token{Type: TokenCloseBracket, Value: "]"},
	'{': Token{Type: TokenOpenBrace, Value: "{"},
	'}': Token{Type: TokenCloseBrace, Value: "}"},
	':': Token{Type: TokenColon, Value: ":"},
	';': Token{Type: TokenSemicolon, Value: ";"},
	',': Token{Type: TokenComma, Value: ","},

	'\\': Token{Type: TokenBadEscape, Value: "\\"},

	'A': Token{Type: TokenDashMatch, Value: "|="},
	'B': Token{Type: TokenColumn, Value: "||"},
	'C': Token{Type: TokenCDC, Value: "-->"},
	'O': Token{Type: TokenCDO, Value: "<!--"},

	'E': Token{Type: TokenEOF},
}

// 4.3.1
func (z *Tokenizer) consume() Token {
	ch := z.nextByte()

	switch ch {
	case 0: // EOF
		return premadeTokens['E']
	case '\n', '\t', ' ':
		return z.consumeWhitespace(ch)
	case '"', '\'':
		return z.consumeString(ch)
	case '#':
		z.repeek()
		if isNameCode(z.peek[0]) || z.nextIsEscape() {
			e := &TokenExtraHash{
				IsIdentifier: z.nextStartsIdentifier(),
			}
			return Token{
				Type:  TokenHash,
				Extra: e,
				Value: z.consumeName(),
			}
		}
		break
	case '(', ')', ',', ':', ';', '[', ']', '{', '}':
		return premadeTokens[ch]
	case '$', '*', '^', '~':
		z.repeek()
		if z.peek[0] == '=' {
			z.r.Discard(1)
			return premadeTokens[ch]
		}
	case '|':
		z.repeek()
		if z.peek[0] == '=' {
			z.r.Discard(1)
			return premadeTokens['A']
		} else if z.peek[0] == '|' {
			z.r.Discard(1)
			return premadeTokens['B']
		}
	case '+':
		z.unreadByte()
		z.repeek()
		if z.nextIsNumber() {
			return z.consumeNumeric()
		}
		z.nextByte() // re-read, fall down to TokenDelim
	case '-':
		z.unreadByte()
		z.repeek()
		if z.nextIsNumber() {
			return z.consumeNumeric()
		}
		if z.nextStartsIdentifier() {
			return z.consumeIdentish()
		}
		if z.nextCompare("-->") {
			z.r.Discard(3)
			return premadeTokens['C']
		}
		z.nextByte() // re-read, fall down to TokenDelim
	case '.':
		z.unreadByte()
		z.repeek()
		if z.nextIsNumber() {
			return z.consumeNumeric()
		}
		z.nextByte() // re-read, fall down to TokenDelim
	case '/':
		z.repeek()
		if z.peek[0] == '*' {
			z.r.Discard(1)
			return z.consumeComment()
		}
	case '<':
		z.repeek()
		if z.nextCompare("!--") {
			z.r.Discard(3)
			return premadeTokens['O']
		}
	case '@':
		z.repeek()
		if z.nextStartsIdentifier() {
			s := z.consumeName()
			return Token{
				Type:  TokenAtKeyword,
				Value: s,
			}
		}
	case '\\':
		z.unreadByte()
		z.repeek()
		if z.nextIsEscape() {
			// input stream has a valid escape
			return z.consumeIdentish()
		}
		z.nextByte()
		// z.err = errBadEscape
		return premadeTokens['\\']
	case 'U', 'u':
		z.unreadByte()
		z.repeek()
		if z.peek[1] == '+' && (isHexDigit(z.peek[2]) || (z.peek[2] == '?')) {
			z.r.Discard(2) // (!) only discard the U+
			return z.consumeUnicodeRange()
		}
		break
	}

	if '0' <= ch && ch <= '9' {
		z.unreadByte()
		return z.consumeNumeric()
	}
	if isNameStart(ch) {
		z.unreadByte()
		return z.consumeIdentish()
	}
	return Token{
		Type:  TokenDelim,
		Value: string(rune(ch)),
	}
}

// return the next byte, with 0 on EOF and panicing on other errors
func (z *Tokenizer) nextByte() byte {
	if z.err == io.EOF {
		return 0
	}
	by, err := z.r.ReadByte()
	if err == io.EOF {
		z.err = io.EOF
		return 0
	} else if err != nil {
		panic(err)
	}
	return by
}

func (z *Tokenizer) unreadByte() {
	if z.err == io.EOF {
		// don't unread after EOF
		return
	}
	z.r.UnreadByte()
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

func isNotWhitespace(r rune) bool {
	return !isWhitespace(r)
}

func (z *Tokenizer) consumeWhitespace(ch byte) Token {
	const wsBufSize = 32

	sawNewline := false
	if ch == '\n' {
		sawNewline = true
	}

	for {
		// Consume whitespace in chunks of up to wsBufSize
		buf, err := z.r.Peek(wsBufSize)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if len(buf) == 0 {
			break // Reached EOF
		}
		// find first non-whitespace char, discard up to there
		idx := bytes.IndexFunc(buf, isNotWhitespace)
		if idx == 0 {
			break // Nothing to trim
		}
		if idx == -1 {
			idx = len(buf) // Entire buffer is spaces
		}
		if /* const */ ch != 0 {
			// only check for newlines when we're actually outputting a token
			nlIdx := bytes.IndexByte(buf[:idx], '\n')
			if nlIdx != -1 {
				sawNewline = true
			}
		}
		z.r.Discard(idx)
	}

	if sawNewline {
		return Token{
			Type:  TokenS,
			Value: "\n",
		}
	}
	return Token{
		Type:  TokenS,
		Value: " ",
	}
}

// 4.3.2
func (z *Tokenizer) consumeNumeric() Token {
	repr, notInteger := z.consumeNumericInner()
	e := &TokenExtraNumeric{
		NonInteger: notInteger,
	}
	t := Token{
		Type:  TokenNumber,
		Value: string(repr),
		Extra: e,
	}
	z.repeek()
	if z.nextStartsIdentifier() {
		t.Type = TokenDimension
		e.Dimension = z.consumeName()
	} else if z.peek[0] == '%' {
		z.r.Discard(1)
		t.Type = TokenPercentage
	}
	return t
}

// §4.3.3
func (z *Tokenizer) consumeIdentish() Token {
	s := z.consumeName()
	z.repeek()
	if z.peek[0] == '(' {
		z.r.Discard(1)
		if strings.EqualFold(s, "url") {
			return z.consumeURL()
		}
		return Token{
			Type:  TokenFunction,
			Value: s,
		}
	} else {
		return Token{
			Type:  TokenIdent,
			Value: s,
		}
	}
}

// §4.3.4
func (z *Tokenizer) consumeString(delim byte) Token {
	var frag []byte
	var by byte
	for {
		by = z.nextByte()
		if by == delim || by == 0 {
			// end of string, EOF
			return Token{
				Type:  TokenString,
				Value: string(frag),
			}
		} else if by == '\n' {
			z.unreadByte()
			/* z.err = */ er := &ParseError{
				Type:    TokenBadString,
				Message: "unterminated string",
			}
			return Token{
				Type:  TokenBadString,
				Value: string(frag),
				Extra: &TokenExtraError{Err: er},
			}
		} else if by == '\\' {
			z.unreadByte()
			z.repeek()
			if z.peek[1] == 0 {
				// escape @ EOF, ignore.
				z.nextByte() // '\'
			} else if z.peek[1] == '\n' {
				// valid escaped newline, ignore.
				z.nextByte() // '\'
				z.nextByte() // newline
			} else if true {
				// stream will always contain a valid escape here
				z.nextByte() // '\'
				cp := z.consumeEscapedCP()
				var tmp [utf8.UTFMax]byte
				n := utf8.EncodeRune(tmp[:], cp)
				frag = append(frag, tmp[:n]...)
			}
		} else {
			frag = append(frag, by)
		}
	}
}

// §4.3.5
// reader must be in the "url(" state
func (z *Tokenizer) consumeURL() Token {
	z.consumeWhitespace(0)
	z.repeek()
	if z.peek[0] == 0 {
		return Token{
			Type:  TokenURI,
			Value: "",
		}
	} else if z.peek[0] == '\'' || z.peek[0] == '"' {
		delim := z.peek[0]
		z.nextByte()
		t := z.consumeString(delim)
		if t.Type == TokenBadString {
			t.Type = TokenBadURI
			t.Value += z.consumeBadURL()
			/* z.err = */ pe := &ParseError{
				Type:    TokenBadURI,
				Message: "unterminated string in url()",
			}
			t.Extra = &TokenExtraError{
				Err: pe,
			}
			return t
		}
		t.Type = TokenURI
		z.consumeWhitespace(0)
		z.repeek()
		if z.peek[0] == ')' || z.peek[0] == 0 {
			z.nextByte()
			return t
		}
		t.Type = TokenBadURI
		t.Value += z.consumeBadURL()
		/* z.err = */ pe := &ParseError{
			Type:    TokenBadURI,
			Message: "url() with string missing close parenthesis",
		}
		t.Extra = &TokenExtraError{
			Err: pe,
		}
		return t
	}
	var frag []byte
	var by byte
	for {
		by = z.nextByte()
		if by == ')' || by == 0 {
			return Token{Type: TokenURI, Value: string(frag)}
		} else if isWhitespace(rune(by)) {
			z.consumeWhitespace(0)
			z.repeek()
			if z.peek[0] == ')' || z.peek[0] == 0 {
				z.nextByte() // ')'
				return Token{Type: TokenURI, Value: string(frag)}
			}
			/* z.err = */ pe := &ParseError{
				Type:    TokenBadURI,
				Message: "bare url() with internal whitespace",
			}
			return Token{
				Type:  TokenBadURI,
				Value: string(frag) + z.consumeBadURL(),
				Extra: &TokenExtraError{Err: pe},
			}
		} else if by == '\'' || by == '"' || by == '(' {
			/* z.err = */ pe := &ParseError{
				Type:    TokenBadURI,
				Message: fmt.Sprintf("bare url() with illegal character '%c'", by),
			}
			return Token{
				Type:  TokenBadURI,
				Value: string(frag) + z.consumeBadURL(),
				Extra: &TokenExtraError{Err: pe},
			}
		} else if isNonPrintable(by) {
			/* z.err = */ pe := &ParseError{
				Type:    TokenBadURI,
				Message: fmt.Sprintf("bare url() with unprintable character '%d'", by),
			}
			return Token{
				Type:  TokenBadURI,
				Value: string(frag) + z.consumeBadURL(),
				Extra: &TokenExtraError{Err: pe},
			}
		} else if by == '\\' {
			z.unreadByte()
			z.repeek()
			if z.nextIsEscape() {
				z.nextByte() // '\'
				cp := z.consumeEscapedCP()
				var tmp [utf8.UTFMax]byte
				n := utf8.EncodeRune(tmp[:], cp)
				frag = append(frag, tmp[:n]...)
			} else {
				/* z.err = */ pe := &ParseError{
					Type:    TokenBadURI,
					Message: fmt.Sprintf("bare url() with invalid escape"),
				}
				return Token{
					Type:  TokenBadURI,
					Value: string(frag) + z.consumeBadURL(),
					Extra: &TokenExtraError{Err: pe},
				}
			}
		} else {
			frag = append(frag, by)
		}
	}
}

// §4.3.6
func (z *Tokenizer) consumeUnicodeRange() Token {
	var sdigits [6]byte
	var by byte
	haveQuestionMarks := false
	i := 0
	for {
		by = z.nextByte()
		if i >= 6 {
			break // weird condition so that unreadByte() works
		}
		if by == '?' {
			sdigits[i] = by
			haveQuestionMarks = true
			i++
		} else if !haveQuestionMarks && isHexDigit(by) {
			sdigits[i] = by
			i++
		} else {
			break
		}
	}
	z.unreadByte()
	slen := i
	var edigits [6]byte
	var elen int
	z.repeek()

	if haveQuestionMarks {
		copy(edigits[:slen], sdigits[:slen])
		elen = slen
		for idx := range sdigits {
			if sdigits[idx] == '?' {
				sdigits[idx] = '0'
				edigits[idx] = 'F'
			}
		}
	} else if z.peek[0] == '-' && isHexDigit(z.peek[1]) {
		z.nextByte() // '-'
		i = 0
		for {
			by = z.nextByte()
			if i < 6 && isHexDigit(by) {
				edigits[i] = by
				i++
			} else {
				break
			}
		}
		z.unreadByte()
		elen = i
	} else {
		copy(edigits[:], sdigits[:])
		elen = slen
	}

	// 16 = hex, 32 = int32
	startCP, err := strconv.ParseInt(string(sdigits[:slen]), 16, 32)
	if err != nil {
		panic(fmt.Sprintf("ParseInt failure: %s", err))
	}
	endCP, err := strconv.ParseInt(string(edigits[:elen]), 16, 32)
	if err != nil {
		panic(fmt.Sprintf("ParseInt failure: %s", err))
	}
	e := &TokenExtraUnicodeRange{
		Start: rune(startCP),
		End:   rune(endCP),
	}
	return Token{
		Type:  TokenUnicodeRange,
		Value: e.String(),
		Extra: e,
	}
}

func (z *Tokenizer) consumeComment() Token {
	var frag []byte
	var by byte
	for {
		by = z.nextByte()
		if by == '*' {
			z.repeek()
			if z.peek[0] == '/' {
				z.nextByte() // '/'
				return Token{
					Type:  TokenComment,
					Value: string(frag),
				}
			}
		} else if by == 0 {
			return Token{
				Type:  TokenComment,
				Value: string(frag),
			}
		}
		frag = append(frag, by)
	}
}

// §4.3.7
// after the "\"
func (z *Tokenizer) consumeEscapedCP() rune {
	by := z.nextByte()
	if by == 0 {
		return utf8.RuneError
	} else if isHexDigit(by) {
		var digits = make([]byte, 6)
		digits[0] = by
		i := 1
		// (!) weird looping condition so that we UnreadByte() at the end
		for {
			by = z.nextByte()
			if i < 6 && isHexDigit(by) {
				digits[i] = by
				i++
			} else {
				break
			}
		}

		if isNotWhitespace(rune(by)) && by != 0 {
			z.unreadByte()
		}
		digits = digits[:i]
		// 16 = hex, 22 = bit width of unicode
		cpi, err := strconv.ParseInt(string(digits), 16, 32)
		if err != nil || cpi == 0 || cpi > utf8.MaxRune {
			return utf8.RuneError
		}
		return rune(cpi)
	} else {
		z.unreadByte()
		ru, _, err := z.r.ReadRune()
		if err == io.EOF {
			z.err = io.EOF
			return utf8.RuneError
		} else if err != nil {
			z.err = err
			panic(err)
		} else {
			return ru
		}
	}
}

// §4.3.11
func (z *Tokenizer) consumeName() string {
	var frag []byte
	var by byte
	for {
		by = z.nextByte()
		if by == '\\' {
			z.unreadByte()
			z.repeek()
			if z.nextIsEscape() {
				z.nextByte()
				cp := z.consumeEscapedCP()
				var tmp [utf8.UTFMax]byte
				n := utf8.EncodeRune(tmp[:], cp)
				frag = append(frag, tmp[:n]...)
				continue
			} else {
				return string(frag)
			}
		} else if isNameCode(by) {
			frag = append(frag, by)
			continue
		} else {
			z.unreadByte()
			return string(frag)
		}
	}
}

// §4.3.12
func (z *Tokenizer) consumeNumericInner() (repr []byte, notInteger bool) {
	by := z.nextByte()
	if by == '+' || by == '-' {
		repr = append(repr, by)
		by = z.nextByte()
	}
	consumeDigits := func() {
		for '0' <= by && by <= '9' {
			repr = append(repr, by)
			by = z.nextByte()
		}
		if by != 0 {
			// don't attempt to unread EOF
			z.unreadByte()
		}
	}

	consumeDigits()
	z.repeek()
	if z.peek[0] == '.' && '0' <= z.peek[1] && z.peek[1] <= '9' {
		notInteger = true

		by = z.nextByte() // '.'
		repr = append(repr, by)
		by = z.nextByte()
		consumeDigits()
		z.repeek()
	}
	// [eE][+-]?[0-9]
	if (z.peek[0] == 'e') || (z.peek[0] == 'E') {
		var n int
		if (z.peek[1] == '+' || z.peek[1] == '-') && ('0' <= z.peek[2] && z.peek[2] <= '9') {
			n = 3
		} else if '0' <= z.peek[1] && z.peek[1] <= '9' {
			n = 2
		}
		if n != 0 {
			notInteger = true
			repr = append(repr, z.peek[:n]...)
			z.r.Discard(n)
			by = z.nextByte()
			consumeDigits()
		}
	}

	return repr, notInteger
}

// §4.3.14
func (z *Tokenizer) consumeBadURL() string {
	var frag []byte
	var by byte
	for {
		by = z.nextByte()
		if by == ')' || by == 0 {
			return string(frag)
		} else if by == '\\' {
			z.unreadByte()
			z.repeek()
			if z.nextIsEscape() {
				z.nextByte() // '\'
				// Allow for escaped right paren "\)"
				cp := z.consumeEscapedCP()
				var tmp [utf8.UTFMax]byte
				n := utf8.EncodeRune(tmp[:], cp)
				frag = append(frag, tmp[:n]...)
				continue
			}
			z.nextByte() // '\'
		}
		frag = append(frag, by)
	}
}
