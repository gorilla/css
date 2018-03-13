package scanner

import (
	"bufio"
	stdErrors "errors"
	"golang.org/x/text/transform"
)

var (
	ErrBadEscape = &ParseError{Type: TokenBadEscape, Message: "bad escape (backslash-newline) in input"}
)

// Tokenizer scans an input and emits tokens following the CSS Syntax Level 3
// specification.
type Tokenizer struct {
	r    *bufio.Reader
	err  error
	peek [3]byte

	tok Token
}

// Construct a Tokenizer from the given input. Input need not be normalized.
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		r: bufio.NewReader(transform.NewReader(r, new(normalize))),
	}
}

// Scan for the next token.  If the tokenizer is in an error state, no input will be consumed.  See .AcknowledgeError().
func (z *Tokenizer) Scan() {
	defer func() {
		rec := recover()
		if rErr, ok := rec.(error); ok {
			z.err = rErr
		} else if rec != nil {
			panic(rec)
		}
	}()

	if z.err != nil {
		z.tok = z.next()
	}
}

// Return the current token.
func (z *Tokenizer) Token() Token {
	return t.tok
}

func (z *Tokenizer) Err() error {
	return t.err
}

// Acknowledge a returned error token.  This can only be called to clear TokenBadString, TokenBadURI, and TokenEscape.
func (z *Tokenizer) AcknowledgeError() {
	parseErr, ok := t.err.(*ParseError)
	if !ok {
		panic("cssparse: AcknowledgeError() called for a foreign error")
	}
}

// repeek reads the next 3 bytes into the tokenizer.
func (z *Tokenizer) repeek() {
	by, err := z.r.Peek(3)
	if err != nil {
		panic(err)
	}
	copy(z.peek, by)

	// zero fill on EOF
	i := len(by)
	for i < 3 {
		z.peek[i] = 0
		i++
	}
}

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

func isNameStart(p byte) bool {
	if p > 0x7F {
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

func isNameCode(p byte) {
	if p > 0x7F {
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
		Type:   TokenSuffixMatch,
		String: "$=",
	},
	'*': Token{
		Type:   TokenSubstringMatch,
		String: "*=",
	},
	'^': Token{
		Type:   TokenPrefixMatch,
		String: "^=",
	},
	'~': Token{
		Type:   TokenIncludeMatch,
		String: "~=",
	},
	'(': Token{Type: TokenOpenParen, String: "("},
	')': Token{Type: TokenCloseParen, String: ")"},
	'[': Token{Type: TokenOpenBracket, String: "["},
	']': Token{Type: TokenCloseBracket, String: "]"},
	'{': Token{Type: TokenOpenBrace, String: "{"},
	'}': Token{Type: TokenCloseBrace, String: "}"},
	':': Token{Type: TokenColon, String: ":"},
	';': Token{Type: TokenSemicolon, String: ";"},
	',': Token{Type: TokenComma, String: ","},

	'\\': Token{Type: TokenBadEscape, String: "\\"},

	'A': Token{Type: TokenDashMatch, String: "|="},
	'B': Token{Type: TokenColumn, String: "||"},
	'C': Token{Type: TokenCDC, String: "-->"},
	'O': Token{Type: TokenCDO, String: "<!--"},

	'E': Token{Type: TokenEOF},
}

func (z *Tokenizer) consume() Token {
	ch, err := z.r.ReadByte()
	if err == io.EOF {
		z.err = io.EOF
		return premadeTokens['E']
	} else if err != nil {
		z.err = err
		panic(err)
	}

	switch ch {
	case '\n', '\t', ' ':
		return z.consumeWhitespace(ch)
	case '"', '\'':
		return z.consumeString(ch)
	case '#':
		z.repeek()
		if isNameCode(z.peek[0]) || z.nextIsEscape() {
			e := &TokenExtraHash{}
			t := &Token{
				Type:  TokenHash,
				Extra: e,
			}
			if z.nextStartsIdentifier() {
				e.IsIdentifier = true
			}
			t.String = z.consumeName()
			return t
		}
	case '(', ')', ',', ':', ';', '[', ']', '{', '}':
		return premadeTokens[ch]
	case '$', '*', '^', '~':
		z.repeek()
		if z.peek[0] == "=" {
			z.r.Discard(1)
			return premadeTokens[ch]
		}
	case '|':
		z.repeek()
		if z.peek[0] == "=" {
			z.r.Discard(1)
			return premadeTokens['A']
		} else if z.peek[0] == "|" {
			z.r.Discard(1)
			return premadeTokens['B']
		}
	case '+':
		z.repeek()
		if z.nextIsNumber() {
			z.r.UnreadByte()
			return z.consumeNumber()
		}
	case '-':
		z.repeek()
		if z.nextIsNumber() {
			z.r.UnreadByte()
			return z.consumeNumber()
		}
		if z.nextStartsIdentifier() {
			z.r.UnreadByte()
			return z.consumeIdentish()
		}
		if z.nextCompare("->") {
			z.r.Discard(2)
			return premadeTokens['C']
		}
	case '.':
		z.repeek()
		if z.nextIsNumber() {
			z.r.UnreadByte()
			return z.consumeNumber()
		}
	case '/':
		z.repeek()
		if z.peek[0] == '*' {
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
				Type:   TokenAtKeyword,
				String: s,
			}
		}
	case '\\':
		z.repeek()
		if z.peek[0] != '\n' {
			// input stream has a valid escape
			z.r.UnreadByte()
			return z.consumeIdentish()
		}
		z.err = ErrBadEscape
		return premadeTokens['\\']
	case 'U', 'u':
		z.repeek()
		if z.peek[0] == '+' && ((z.peek[1] >= '0' && z.peek[1] <= '9') ||
			(z.peek[1] >= 'A' && z.peek[1] <= 'F') ||
			(z.peek[1] >= 'a' && z.peek[1] <= 'f') ||
			(z.peek[1] == '?')) {
			z.r.Discard(1) // (!) only discard the plus sign
			return z.consumeUnicodeRange()
		}
		break
	}

	if '0' <= ch && ch <= '9' {
		z.r.UnreadByte()
		return z.consumeNumber()
	}
	if isNameStart(ch) {
		z.r.UnreadByte()
		return z.consumeIdentish()
	}
	return Token{
		Type:   TokenDelim,
		String: string(rune(ch)),
	}
}
