package scanner

import "bytes"
import "fmt"

func Fuzz(b []byte) int {
	tz := NewTokenizer(bytes.NewReader(b))
	for {
		tt := tz.Next()
		fmt.Printf("%v\n", tt)
		if tt.Type.StopToken() {
			break
		}
	}
	return 1
}
