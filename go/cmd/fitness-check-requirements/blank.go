package main

import "github.com/may-journal/fitness-runner/go/internal/swift"

// blankSwift shares Swift lexical rules with architecture manifest parsing.
func blankSwift(src string) string {
	code, _ := swift.Mask(src)
	return code
}
