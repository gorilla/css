package scanner

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

func Fuzz(b []byte) int {
	var tokens []Token

	tz := NewTokenizer(bytes.NewReader(b))
	for {
		tt := tz.Next()
		fmt.Printf("[OT] %v\n", tt)
		if tt.Type == TokenError {
			// We should not have reading errors
			panic(tt)
		} else if tt.Type == TokenEOF {
			break
		} else {
			tokens = append(tokens, tt)
		}
	}

	// Render and retokenize

	var wr TokenRenderer
	var rerenderBuf bytes.Buffer
	success := false
	defer func() {
		if !success {
			fmt.Println("RERENDER BUFFER:", rerenderBuf.String())
		}
	}()
	pr, pw := io.Pipe()
	defer pr.Close()

	go func() {
		writeTarget := io.MultiWriter(pw, &rerenderBuf)
		for _, v := range tokens {
			wr.WriteTokenTo(writeTarget, v)
		}
		pw.Close()
	}()

	tz = NewTokenizer(pr)
	i := 0
	for {
		for i < len(tokens) && tokens[i].Type == TokenComment {
			i++
		}
		tt := tz.Next()
		fmt.Printf("[RT] %v\n", tt)
		if tt.Type == TokenComment {
			// Ignore comments while comparing
			continue
		}
		if tt.Type == TokenError {
			panic(tt)
		}
		if tt.Type == TokenEOF {
			if i != len(tokens) {
				panic(fmt.Sprintf("unexpected EOF: got EOF from retokenizer, but original token stream is at %d/%d\n%v", i, len(tokens), tokens))
			} else {
				break
			}
		}
		if i == len(tokens) {
			panic(fmt.Sprintf("expected EOF: reached end of original token stream but got %v from retokenizer\n%v", tt, tokens))
		}

		ot := tokens[i]
		if tt.Type != ot.Type {
			panic(fmt.Sprintf("retokenizer gave %v, expected %v (.Type not equal)\n%v", tt, ot, tokens))
		}
		if tt.Value != ot.Value && !tt.Type.StopToken() {
			panic(fmt.Sprintf("retokenizer gave %v, expected %v (.Value not equal)\n%v", tt, ot, tokens))
		}
		if TokenExtraTypeLookup[tt.Type] != nil {
			if !reflect.DeepEqual(tt, ot) && !tt.Type.StopToken() {
				panic(fmt.Sprintf("retokenizer gave %v, expected %v (.Extra not equal)\n%v", tt, ot, tokens))
			}
		}
		i++
		continue
	}
	success = true
	return 1
}
