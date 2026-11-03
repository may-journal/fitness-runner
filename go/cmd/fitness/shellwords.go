package main

import (
	"regexp"
	"strings"
)

// shellWord is one word of a shell call. known is false when the shell would
// expand part of it, such as a variable, so its text is not what runs.
type shellWord struct {
	text  string
	known bool
}

// shellCall is one simple command: its words and its heredoc or here-string
// stdin, when it has one.
type shellCall struct {
	words []shellWord
	stdin *shellWord
}

// heredocInQuotes matches the $(cat <<'EOF' ... EOF) form agents use to pass
// a long body inside double quotes, capturing the delimiter.
var heredocInQuotes = regexp.MustCompile(`^\$\(cat <<-?\s*['"]?([A-Za-z_][A-Za-z0-9_]*)['"]?\n`)

// heredocStart matches a heredoc redirection, capturing its delimiter.
var heredocStart = regexp.MustCompile(`^<<-?\s*['"]?([A-Za-z_][A-Za-z0-9_]*)['"]?`)

// expandStarts are the bytes that, after $, start a shell expansion.
const expandStarts = "{(_@*#?$!abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// wordEnds are the bytes that end an unquoted word.
const wordEnds = " \t\n;&|()<>"

// shellCalls splits a command line into simple commands at ;, &, |, ( and
// ) and newlines, reading quotes, escapes and each heredoc as a POSIX shell
// does. It reads only what gh calls need and marks what it cannot expand.
func shellCalls(line string) []shellCall {
	p := shellParser{src: line}
	for p.i < len(p.src) {
		p.step(topLevel)
	}
	p.endCall()
	return p.calls
}

// shellParser walks a command line once, building its calls.
type shellParser struct {
	src    string
	i      int
	calls  []shellCall
	call   shellCall
	word   shellWord
	inWord bool
	// pending holds the delimiter of each heredoc whose body start after the
	// current line, each with the call that reads it.
	pending []pendingHeredoc
}

// pendingHeredoc is a heredoc whose body follows the current line.
type pendingHeredoc struct {
	delim string
	call  int
}

// topLevel handles the bytes with a meaning outside quotes, and inWord those
// that keep a word going; every other byte is plain text.
var topLevel, inWord map[byte]func(*shellParser)

func init() {
	inWord = map[byte]func(*shellParser){
		'\'': (*shellParser).singleQuoted, '"': (*shellParser).doubleQuoted, '\\': (*shellParser).escaped,
		'$': (*shellParser).expansion, '`': (*shellParser).backtick,
	}
	topLevel = map[byte]func(*shellParser){
		'\n': (*shellParser).newline, ' ': (*shellParser).blank, '\t': (*shellParser).blank, '<': (*shellParser).redirect,
	}
	for _, c := range []byte(";&|()") {
		topLevel[c] = (*shellParser).separator
	}
	for c, handle := range inWord {
		topLevel[c] = handle
	}
}

// step handles the byte at the cursor with its handler in table.
func (p *shellParser) step(table map[byte]func(*shellParser)) {
	if handle, ok := table[p.src[p.i]]; ok {
		handle(p)
		return
	}
	p.plain()
}

// plain adds the byte at the cursor to the word.
func (p *shellParser) plain() {
	p.add(p.src[p.i : p.i+1])
	p.i++
}

// add appends text to the current word.
func (p *shellParser) add(text string) {
	if !p.inWord {
		p.word, p.inWord = shellWord{known: true}, true
	}
	p.word.text += text
}

// unknown marks the current word as one the shell would expand.
func (p *shellParser) unknown() {
	p.add("")
	p.word.known = false
}

func (p *shellParser) endWord() {
	if p.inWord {
		p.call.words = append(p.call.words, p.word)
		p.inWord = false
	}
}

func (p *shellParser) endCall() {
	p.endWord()
	if len(p.call.words) > 0 || p.call.stdin != nil {
		p.calls = append(p.calls, p.call)
	}
	p.call = shellCall{}
}

func (p *shellParser) blank() {
	p.endWord()
	p.i++
}

func (p *shellParser) separator() {
	p.endCall()
	p.i++
}

// newline ends the call, then reads the heredoc bodies the line opened.
func (p *shellParser) newline() {
	p.endCall()
	p.i++
	for _, h := range p.pending {
		body, after, ok := heredocBody(p.src, p.i, h.delim)
		p.calls[h.call].stdin = &shellWord{text: body, known: ok}
		p.i = after
	}
	p.pending = nil
}

func (p *shellParser) singleQuoted() {
	end := strings.IndexByte(p.src[p.i+1:], '\'')
	if end < 0 {
		end = len(p.src) - p.i - 1
	}
	p.add(p.src[p.i+1 : p.i+1+end])
	p.i = min(len(p.src), p.i+end+2)
}

func (p *shellParser) doubleQuoted() {
	p.add("")
	p.i++
	for p.i < len(p.src) && p.src[p.i] != '"' {
		p.quotedStep()
	}
	p.i++
}

// quotedStep handles one byte inside double quotes.
func (p *shellParser) quotedStep() {
	switch p.src[p.i] {
	case '\\':
		p.quotedEscape()
	case '$':
		p.quotedDollar()
	case '`':
		p.backtick()
	default:
		p.plain()
	}
}

// quotedEscape reads a backslash inside double quotes, which escapes only
// $, `, ", \ and a newline.
func (p *shellParser) quotedEscape() {
	next := byteAt(p.src, p.i+1)
	switch {
	case next == '\n':
		p.i += 2
	case next != 0 && strings.IndexByte("$`\"\\", next) >= 0:
		p.add(string(next))
		p.i += 2
	default:
		p.plain()
	}
}

// quotedDollar reads $ inside double quotes: a cat heredoc body, another
// expansion, or a plain dollar sign.
func (p *shellParser) quotedDollar() {
	if heredocInQuotes.MatchString(p.src[p.i:]) {
		p.quotedHeredoc()
		return
	}
	p.expansion()
}

// quotedHeredoc reads a $(cat <<'EOF' ... EOF) body inside double quotes.
func (p *shellParser) quotedHeredoc() {
	m := heredocInQuotes.FindStringSubmatch(p.src[p.i:])
	body, after, ok := heredocBody(p.src, p.i+len(m[0]), m[1])
	end := strings.IndexByte(p.src[after:], ')')
	if !ok || end < 0 || strings.TrimSpace(p.src[after:after+end]) != "" {
		p.unknown()
		p.i = len(p.src)
		return
	}
	// Command substitution drops the body's trailing newlines.
	p.add(strings.TrimRight(body, "\n"))
	p.i = after + end + 1
}

func (p *shellParser) escaped() {
	if next := byteAt(p.src, p.i+1); next != '\n' && next != 0 {
		p.add(string(next))
	}
	p.i += 2
}

// expansion reads $: a variable or substitution the shell expands, or a
// plain dollar sign.
func (p *shellParser) expansion() {
	next := byteAt(p.src, p.i+1)
	if next != 0 && strings.IndexByte(expandStarts, next) >= 0 {
		p.unknown()
	} else {
		p.add("$")
	}
	p.i++
}

func (p *shellParser) backtick() {
	p.unknown()
	p.i++
}

// redirect reads <<< or << as stdin; any other < is plain.
func (p *shellParser) redirect() {
	switch {
	case strings.HasPrefix(p.src[p.i:], "<<<"):
		p.hereString()
	case strings.HasPrefix(p.src[p.i:], "<<"):
		p.heredoc()
	default:
		p.plain()
	}
}

// hereString reads <<< word as the call's stdin, which gains a newline.
func (p *shellParser) hereString() {
	p.endWord()
	p.i += 3
	p.skipBlanks()
	words := p.call.words
	p.call.words = nil
	for p.i < len(p.src) && strings.IndexByte(wordEnds, p.src[p.i]) < 0 {
		p.step(inWord)
	}
	p.endWord()
	p.call.stdin = &shellWord{}
	if len(p.call.words) > 0 {
		p.call.stdin = &shellWord{text: p.call.words[0].text + "\n", known: p.call.words[0].known}
	}
	p.call.words = words
}

// skipBlanks moves the cursor past spaces and tabs.
func (p *shellParser) skipBlanks() {
	for byteAt(p.src, p.i) == ' ' || byteAt(p.src, p.i) == '\t' {
		p.i++
	}
}

// heredoc records a << redirection, whose body follows the current line.
func (p *shellParser) heredoc() {
	p.endWord()
	p.call.stdin = &shellWord{}
	m := heredocStart.FindStringSubmatch(p.src[p.i:])
	if m == nil {
		p.i += 2
		return
	}
	p.i += len(m[0])
	p.pending = append(p.pending, pendingHeredoc{delim: m[1], call: len(p.calls)})
}

// heredocBody reads lines from start up to the line that is delim alone. It
// returns the body with its newlines and the index after that line.
func heredocBody(src string, start int, delim string) (body string, after int, ok bool) {
	for i := start; i < len(src); {
		line, next := src[i:], len(src)
		if end := strings.IndexByte(src[i:], '\n'); end >= 0 {
			line, next = src[i:i+end], i+end+1
		}
		if strings.TrimLeft(line, "\t") == delim {
			return src[start:i], next, true
		}
		i = next
	}
	return src[start:], len(src), false
}

// byteAt is src[i], or 0 past its end.
func byteAt(src string, i int) byte {
	if i < len(src) {
		return src[i]
	}
	return 0
}
