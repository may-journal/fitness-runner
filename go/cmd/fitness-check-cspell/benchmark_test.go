package main

import (
	"path/filepath"
	"testing"
)

func BenchmarkRepositoryScan(b *testing.B) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		walkFiles(root)
	}
}
