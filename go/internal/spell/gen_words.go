//go:build ignore

// gen_words rewrites words.txt from the lists in dict/. go generate runs it
// from this folder; it imports only wordlist, so it works when words.txt is
// missing.
package main

import (
	"log"

	"github.com/may-journal/fitness-runner/go/internal/spell/wordlist"
)

func main() {
	if err := wordlist.Write("."); err != nil {
		log.Fatal(err)
	}
}
