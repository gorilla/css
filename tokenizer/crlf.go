// Copyright (c) 2018 Kane York. Licensed under 2-Clause BSD.

package tokenizer

// The crlf package helps in dealing with files that have DOS-style CR/LF line
// endings.
//
// Copyright (c) 2015 Andy Balholm. Licensed under 2-Clause BSD.
//
// package crlf

import "golang.org/x/text/transform"

// Normalize takes CRLF, CR, or LF line endings in src, and converts them
// to LF in dst.
//
// cssparse: Also replace null bytes with U+FFFD REPLACEMENT CHARACTER.
type normalize struct {
	prev byte
}

const replacementCharacter = "\uFFFD"

func (n *normalize) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nDst < len(dst) && nSrc < len(src) {
		c := src[nSrc]
		switch c {
		case '\r':
			dst[nDst] = '\n'
		case '\n':
			if n.prev == '\r' {
				nSrc++
				n.prev = c
				continue
			}
			dst[nDst] = '\n'
		case 0:
			// nb: len(replacementCharacter) == 3
			if nDst+3 >= len(dst) {
				err = transform.ErrShortDst
				return
			}
			copy(dst[nDst:], replacementCharacter[:])
			nDst += 2
		default:
			dst[nDst] = c
		}
		n.prev = c
		nDst++
		nSrc++
	}
	if nSrc < len(src) {
		err = transform.ErrShortDst
	}
	return
}

func (n *normalize) Reset() {
	n.prev = 0
}
