package architecture

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// reference is one source line that reaches another path: a Go import of
// a folder, or a Swift use of a symbol declared in a file. Condition is
// the build condition the line sits under, or empty.
type reference struct {
	At                    Location
	To, Symbol, Condition string
}

// what says how the line reaches the other path.
func (r reference) what() string {
	if r.Symbol != "" {
		return "names " + r.Symbol
	}
	return "imports " + r.To
}

func (r reference) under() string {
	if r.Condition == "" {
		return ""
	}
	return " under " + r.Condition
}

// read returns a tracked file's lines; nil when it cannot be read.
func (r repository) read(file string) []string {
	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(file)))
	if err != nil {
		return nil
	}
	return strings.Split(string(body), "\n")
}

// productSources lists the product files with ext, in path order.
func (m *Model) productSources(ext string) []string {
	var files []string
	for _, s := range m.Sources {
		if s.Category == "product" && path.Ext(s.Path) == ext {
			files = append(files, s.Path)
		}
	}
	return files
}

// scanner strips comments, and string literals when asked, from Go and
// Swift source, which share that syntax. Neither can create a dependency.
// Literals are always read, so a // or /* inside one starts no comment.
// A Go raw string, which may span lines, or a Swift escaped identifier is
// kept whole.
type scanner struct {
	literals         bool
	block, text, raw bool
}

// keep writes a span of string literal to out unless literals are stripped.
func (s *scanner) keep(text string, from, to int, out *strings.Builder) int {
	if !s.literals {
		out.WriteString(text[from:to])
	}
	return to
}

func (s *scanner) line(text string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		i = s.step(text, i, &out)
	}
	return out.String()
}

func (s *scanner) step(text string, i int, out *strings.Builder) int {
	switch {
	case s.block:
		return s.skipTo(text, i, "*/", &s.block)
	case s.text:
		return s.keep(text, i, s.skipTo(text, i, `"""`, &s.text), out)
	case s.raw:
		next := s.skipTo(text, i, "`", &s.raw)
		out.WriteString(text[i:next])
		return next
	}
	return s.code(text, i, out)
}

// skipTo moves past the end of close, clearing flag when it is found.
func (s *scanner) skipTo(text string, i int, close string, flag *bool) int {
	end := strings.Index(text[i:], close)
	if end < 0 {
		return len(text)
	}
	*flag = false
	return i + end + len(close)
}

func (s *scanner) code(text string, i int, out *strings.Builder) int {
	if next, ok := s.comment(text, i); ok {
		return next
	}
	if next, ok := s.quote(text, i); ok {
		return s.keep(text, i, next, out)
	}
	switch text[i] {
	case '\'':
		next := runeEnd(text, i)
		out.WriteString(text[i:next])
		return next
	case '`':
		s.raw = true
	}
	out.WriteByte(text[i])
	return i + 1
}

// runeEnd moves past a Go rune literal, kept whole so a quote or // inside
// one starts neither a string nor a comment.
func runeEnd(text string, i int) int {
	for j := i + 1; j < len(text); j++ {
		switch text[j] {
		case '\\':
			j++
		case '\'':
			return j + 1
		}
	}
	return len(text)
}

func (s *scanner) comment(text string, i int) (int, bool) {
	switch {
	case strings.HasPrefix(text[i:], "//"):
		return len(text), true
	case strings.HasPrefix(text[i:], "/*"):
		s.block = true
		return i + 2, true
	}
	return 0, false
}

func (s *scanner) quote(text string, i int) (int, bool) {
	switch {
	case text[i] != '"':
		return 0, false
	case strings.HasPrefix(text[i:], `"""`):
		s.text = true
		return i + 3, true
	}
	return s.literal(text, i+1), true
}

// literal moves past a one-line string literal, escapes included.
func (s *scanner) literal(text string, i int) int {
	for ; i < len(text); i++ {
		switch text[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	return i
}

// conditions tracks the Swift #if directives a line sits under.
type conditions struct {
	stack []string
}

// directive reads a conditional compilation line, comments already
// stripped, reporting whether the line was one.
func (c *conditions) directive(line string) bool {
	line = strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(line, "#if "):
		c.stack = append(c.stack, line)
	case strings.HasPrefix(line, "#elseif ") || line == "#else":
		c.replace(line)
	case line == "#endif":
		c.stack = c.stack[:max(len(c.stack)-1, 0)]
	default:
		return false
	}
	return true
}

func (c *conditions) replace(line string) {
	if len(c.stack) > 0 {
		c.stack[len(c.stack)-1] = line + " of " + c.stack[len(c.stack)-1]
	}
}

func (c *conditions) String() string {
	return strings.Join(c.stack, ", ")
}
