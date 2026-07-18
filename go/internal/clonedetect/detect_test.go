package clonedetect

import (
	"fmt"
	"strings"
	"testing"
)

// block builds lines of distinct identifiers (prefix_row_col …) each
// terminated by ";", so token count per line = perLine + 1 and no
// two blocks with different prefixes share any window.
func block(prefix string, lines, perLine int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		for j := 0; j < perLine; j++ {
			fmt.Fprintf(&b, "%s_%d_%d ", prefix, i, j)
		}
		b.WriteString(";\n")
	}
	return b.String()
}

func fileOf(path, src string) File {
	return File{Path: path, Tokens: Lex(src), Lines: CountLines(src)}
}

func TestDetect(t *testing.T) {
	opt := Options{MinLines: 5, MinTokens: 50}
	// 6 lines x 11 tokens = 66 tokens: enough for a 50-token window.
	clone := block("c", 6, 10)
	cases := []struct {
		name      string
		files     []File
		wantDup   int
		wantTotal int
	}{
		{
			"cross-file clone counts the later copy once",
			[]File{
				fileOf("a", clone+block("ta", 4, 3)),
				fileOf("b", clone+block("tb", 4, 3)),
			},
			6, 20,
		},
		{
			"same-file clone counts the second copy",
			[]File{fileOf("a", clone+block("sep", 3, 3)+clone)},
			6, 15,
		},
		{
			"three copies count two",
			[]File{
				fileOf("a", clone),
				fileOf("b", clone+block("tb", 4, 3)),
				fileOf("c", clone+block("tc", 4, 3)),
			},
			12, 26,
		},
		{
			"duplicate under min tokens is free",
			[]File{
				fileOf("a", block("s", 6, 3)),
				fileOf("b", block("s", 6, 3)),
			},
			0, 12,
		},
		{
			"duplicate under min lines is free",
			[]File{
				fileOf("a", block("w", 3, 20)),
				fileOf("b", block("w", 3, 20)),
			},
			0, 6,
		},
		{
			"no duplicates",
			[]File{
				fileOf("a", block("x", 6, 10)),
				fileOf("b", block("y", 6, 10)),
			},
			0, 12,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stats := Detect(tc.files, opt)
			if stats.DuplicatedLines != tc.wantDup {
				t.Errorf("DuplicatedLines = %d, want %d", stats.DuplicatedLines, tc.wantDup)
			}
			if stats.TotalLines != tc.wantTotal {
				t.Errorf("TotalLines = %d, want %d", stats.TotalLines, tc.wantTotal)
			}
		})
	}
}

func TestPercentage(t *testing.T) {
	cases := []struct {
		name  string
		stats Stats
		want  float64
	}{
		{"zero total is zero", Stats{TotalLines: 0, DuplicatedLines: 0}, 0},
		{"five of one hundred", Stats{TotalLines: 100, DuplicatedLines: 5}, 5},
		{"six of sixty", Stats{TotalLines: 60, DuplicatedLines: 6}, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.stats.Percentage(); got != tc.want {
				t.Fatalf("Percentage() = %v, want %v", got, tc.want)
			}
		})
	}
}
