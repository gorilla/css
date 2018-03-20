// Copyright 2018 Kane York.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gorilla/css/tokenizer generates tokens for a CSS3 input.

It follows the CSS3 specification located at:

	http://www.w3.org/TR/css3-syntax/

To use it, create a new scanner for a given CSS input and call Next() until
the token returned is a "stop token":

	s := tokenizer.New(strings.NewReader(myCSS))
	for {
		token := s.Next()
		if token.Type.StopToken() {
			break
		}
		// Do something with the token...
	}

If the consumer wants to accept malformed input, change the check to the
following instead:

		token := s.Next()
		if token.Type == tokenizer.TokenEOF || token.Type == tokenizer.TokenError {
			break
		}

The three potential tokenization errors are a "bad-escape" (backslash-newline
outside a "string" or url() in the input), a "bad-string" (unescaped newline
inside a "string"), and a "bad-url" (a few different cases). Parsers can choose
to abort when seeing one of these errors, or ignore the declaration and attempt
to recover.

Returned tokens that carry extra information have a non-nil .Extra value. For
TokenError, TokenBadEscape, TokenBadString, and TokenBadURI, the
TokenExtraError type carries an `error` with informative text about the nature
of the error. For TokenNumber, TokenPercentage, and TokenDimension, the
TokenExtraNumeric specifies whether the number is integral, and for
TokenDimension, contains the unit string (e.g. "px"). For TokenUnicodeRange,
the TokenExtraUnicodeRange type contains the actual start and end values of the
range.

Note: the scanner doesn't perform lexical analysis or, in other words, it
doesn't care about the token context. It is intended to be used by a
lexer or parser.
*/
package tokenzier
