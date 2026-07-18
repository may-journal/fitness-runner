package clonedetect

import (
	"reflect"
	"testing"
)

func TestLex(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []Token
	}{
		{"identifiers and punctuation", "foo = bar(baz);", []Token{
			{"foo", 1}, {"=", 1}, {"bar", 1}, {"(", 1}, {"baz", 1}, {")", 1}, {";", 1},
		}},
		{"line comment stripped", "foo // hi there\nbar", []Token{
			{"foo", 1}, {"bar", 2},
		}},
		{"hash comment stripped", "foo # note\nbar", []Token{
			{"foo", 1}, {"bar", 2},
		}},
		{"block comment counts newlines", "a /* x\ny */ b", []Token{
			{"a", 1}, {"b", 2},
		}},
		{"html comment counts newlines", "a <!-- x\n--> b", []Token{
			{"a", 1}, {"b", 2},
		}},
		{"double-quoted string keeps content drops quotes", `a "hi \" there" b`, []Token{
			{"a", 1}, {`$str:hi \" there`, 1}, {"b", 1},
		}},
		{"quote style normalizes away", "a 'x' b\nc \"x\" d", []Token{
			{"a", 1}, {"$str:x", 1}, {"b", 1}, {"c", 2}, {"$str:x", 2}, {"d", 2},
		}},
		{"backtick string spans lines", "a `x\ny` b", []Token{
			{"a", 1}, {"$str:x\ny", 1}, {"b", 2},
		}},
		{"unterminated quote closes at end of line", "don't panic\nnext", []Token{
			{"don", 1}, {"$str:t panic", 1}, {"next", 2},
		}},
		{"numbers collapse", "x = 0x1F + 2.5e3", []Token{
			{"x", 1}, {"=", 1}, {"$num", 1}, {"+", 1}, {"$num", 1},
		}},
		{"ignore markers drop the region", "keep1\n// jscpd:ignore-start\nhidden secret\n// jscpd:ignore-end\nkeep2", []Token{
			{"keep1", 1}, {"keep2", 5},
		}},
		{"non-ascii rune is one token", "a → b", []Token{
			{"a", 1}, {"→", 1}, {"b", 1},
		}},
		{"empty source", "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Lex(tc.src)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Lex(%q) = %v, want %v", tc.src, got, tc.want)
			}
		})
	}
}

func TestCountLines(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{"empty", "", 0},
		{"one line no newline", "a", 1},
		{"one line with newline", "a\n", 1},
		{"three lines", "a\nb\nc\n", 3},
		{"trailing text after newline", "a\nb", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CountLines(tc.content); got != tc.want {
				t.Fatalf("CountLines(%q) = %d, want %d", tc.content, got, tc.want)
			}
		})
	}
}
