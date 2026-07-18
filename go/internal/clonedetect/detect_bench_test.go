package clonedetect

import (
	"fmt"
	"testing"
)

// benchCorpus builds nFiles lexed files of lines x perLine distinct tokens
// each, prepending a shared clone block to every third file so the counting
// and marking phases have real work alongside the hashing.
func benchCorpus(nFiles, lines, perLine int) []File {
	clone := block("clone", 8, 10)
	files := make([]File, nFiles)
	for i := 0; i < nFiles; i++ {
		src := block(fmt.Sprintf("f%d", i), lines, perLine)
		if i%3 == 0 {
			src = clone + src
		}
		files[i] = fileOf(fmt.Sprintf("bench%d", i), src)
	}
	return files
}

func BenchmarkDetect(b *testing.B) {
	files := benchCorpus(120, 400, 8)
	opt := Options{MinLines: 5, MinTokens: 50}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Detect(files, opt)
	}
}
