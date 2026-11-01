package comments

import (
	"regexp"
	"strings"
)

// heredoc opens a shell here-document and captures its end word.
var heredoc = regexp.MustCompile(`<<-?\s*['"]?([A-Za-z_][A-Za-z0-9_]*)['"]?`)

// shellToken matches a quoted string, which may stay open past the line's
// end, an escaped byte, or a #.
var shellToken = regexp.MustCompile(`'[^']*'?|"(?:\\.|[^"\\])*"?|\\.|#`)

// shellNotes reads the # comments of a shell script. A # starts a comment
// at the start of a word outside quotes, and a here-document's body holds
// none.
func shellNotes(src []byte) []note {
	r := &shellRead{}
	for i, line := range lines(src) {
		r.line(i+1, line)
	}
	return r.notes
}

// shellRead carries an open quote and here-document over lines.
type shellRead struct {
	notes []note
	quote string
	end   string
}

func (r *shellRead) line(n int, line string) {
	if r.end != "" {
		r.endHeredoc(line)
		return
	}
	code := r.comment(n, line)
	if m := heredoc.FindStringSubmatch(code); m != nil && r.quote == "" {
		r.end = m[1]
	}
}

func (r *shellRead) endHeredoc(line string) {
	if strings.TrimSpace(line) == r.end {
		r.end = ""
	}
}

// comment records the comment on line, if any, and returns the code
// before it. A quote open from an earlier line goes in front, so its
// string reads whole.
func (r *shellRead) comment(n int, line string) string {
	text := r.quote + line
	hash, quote := shellHash(text)
	r.quote = quote
	if hash < 0 {
		return line
	}
	from := hash - (len(text) - len(line))
	r.notes = append(r.notes, at(n, line, from, len(line)))
	return line[:from]
}

// shellHash finds the # that starts a comment in text, or -1, and the
// quote left open at its end.
func shellHash(text string) (int, string) {
	for _, m := range shellToken.FindAllStringIndex(text, -1) {
		tok := text[m[0]:m[1]]
		if tok == "#" && wordStart(text, m[0]) {
			return m[0], ""
		} else if unclosed(tok) {
			return -1, tok[:1]
		}
	}
	return -1, ""
}

// unclosed reports a quoted string that runs past the line's end.
func unclosed(tok string) bool {
	return strings.Contains(`'"`, tok[:1]) && (len(tok) == 1 || tok[len(tok)-1] != tok[0])
}

// wordStart reports a byte at a line's start or after a space or a shell
// operator, where a # begins a comment.
func wordStart(line string, i int) bool {
	return i == 0 || strings.IndexByte(" \t;|&()", line[i-1]) >= 0
}

// yamlToken matches a quoted scalar or a #.
var yamlToken = regexp.MustCompile(`'[^']*'|"(?:\\.|[^"\\])*"|#`)

// blockScalar ends a YAML line that opens a literal or folded block, whose
// more-indented lines below are text.
var blockScalar = regexp.MustCompile(`[|>][-+0-9]*\s*$`)

// yamlNotes reads the # comments of YAML. A # starts a comment at a line's
// start or after a space, outside a quoted scalar, and a block scalar's
// text holds none.
func yamlNotes(src []byte) []note {
	r := &yamlRead{scalar: -1}
	for i, line := range lines(src) {
		r.line(i+1, line)
	}
	return r.notes
}

// yamlRead carries the indent of the line that opened a block scalar, or
// -1 outside one.
type yamlRead struct {
	notes  []note
	scalar int
}

func (r *yamlRead) line(n int, line string) {
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if r.inScalar(line, indent) {
		return
	}
	code := line
	if hash := yamlHash(line); hash >= 0 {
		r.notes = append(r.notes, at(n, line, hash, len(line)))
		code = line[:hash]
	}
	r.scalar = -1
	if blockScalar.MatchString(code) {
		r.scalar = indent
	}
}

// inScalar reports a blank or more-indented line of an open block scalar.
func (r *yamlRead) inScalar(line string, indent int) bool {
	return r.scalar >= 0 && (strings.TrimSpace(line) == "" || indent > r.scalar)
}

// yamlHash finds the # that starts a comment on line, or -1.
func yamlHash(line string) int {
	for _, m := range yamlToken.FindAllStringIndex(line, -1) {
		if line[m[0]] == '#' && spaceBefore(line, m[0]) {
			return m[0]
		}
	}
	return -1
}

// spaceBefore reports a byte at a line's start or after a space or tab.
func spaceBefore(line string, i int) bool {
	return i == 0 || line[i-1] == ' ' || line[i-1] == '\t'
}
