package java

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// escapes are the single-letter escapes a properties file reads.
var escapes = map[byte]byte{'t': '\t', 'n': '\n', 'r': '\r', 'f': '\f'}

// blanks are the whitespace characters a properties file skips.
const blanks = " \t\f"

// Properties reads a Java properties file the way java.util.Properties does,
// or none for a missing file.
func Properties(file string) map[string]string {
	body, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	return parseProperties(string(body))
}

// parseProperties reads the key and value of each logical line of text.
func parseProperties(text string) map[string]string {
	var j joiner
	for _, raw := range strings.Split(text, "\n") {
		j.add(raw)
	}
	out := map[string]string{}
	for _, line := range j.lines() {
		key, value := splitProperty(line)
		out[unescape(key)] = unescape(value)
	}
	return out
}

// joiner joins the physical lines of a properties file into logical ones.
type joiner struct {
	out     []string
	pending string
	open    bool
}

// add takes the next physical line, skipping comments and continuing after an
// odd run of trailing backslashes.
func (j *joiner) add(raw string) {
	line := strings.TrimLeft(strings.TrimRight(raw, "\r"), blanks)
	if !j.open && blankOrComment(line) {
		return
	}
	if trailingBackslashes(line)%2 == 1 {
		j.pending += line[:len(line)-1]
		j.open = true
		return
	}
	j.out = append(j.out, j.pending+line)
	j.pending, j.open = "", false
}

// lines lists the logical lines, including a continuation left open at
// the end of the file.
func (j *joiner) lines() []string {
	if j.open {
		return append(j.out, j.pending)
	}
	return j.out
}

// trailingBackslashes counts the backslashes ending line.
func trailingBackslashes(line string) int {
	return len(line) - len(strings.TrimRight(line, `\`))
}

// splitProperty splits a logical line at its first unescaped `=`, `:` or
// whitespace, which with surrounding blanks is the separator.
func splitProperty(line string) (key, value string) {
	i := keyEnd(line)
	rest := strings.TrimLeft(line[i:], blanks)
	if rest != "" && (rest[0] == '=' || rest[0] == ':') {
		rest = strings.TrimLeft(rest[1:], blanks)
	}
	return line[:i], rest
}

// keyEnd finds where a property's key ends.
func keyEnd(line string) int {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '=', ':', ' ', '\t', '\f':
			return i
		}
	}
	return len(line)
}

// unescape resolves the backslash escapes of a key or value.
func unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i = writeEscape(&b, s, i+1)
	}
	return b.String()
}

// writeEscape writes the character the escape at s[i] names, and returns
// the index of the escape's last character.
func writeEscape(b *strings.Builder, s string, i int) int {
	if c, ok := escapes[s[i]]; ok {
		b.WriteByte(c)
		return i
	}
	if r, ok := unicodeEscape(s, i); ok {
		b.WriteRune(r)
		return i + 4
	}
	b.WriteByte(s[i])
	return i
}

// unicodeEscape reads the \uXXXX escape whose u is s[i].
func unicodeEscape(s string, i int) (rune, bool) {
	if s[i] != 'u' || i+4 >= len(s) {
		return 0, false
	}
	n, err := strconv.ParseUint(s[i+1:i+5], 16, 32)
	return rune(n), err == nil
}

// blankOrComment reports whether a trimmed line is empty or a comment.
func blankOrComment(line string) bool {
	return line == "" || line[0] == '#' || line[0] == '!'
}

// GradleProperties reads the Gradle properties a build sees: the build
// root's gradle.properties, overridden by the Gradle user home's.
func GradleProperties(root string, build Project) map[string]string {
	out := map[string]string{}
	for _, file := range []string{
		filepath.Join(root, filepath.FromSlash(build.Dir), "gradle.properties"),
		GradleUserFile("gradle.properties"),
	} {
		for key, value := range Properties(file) {
			out[key] = value
		}
	}
	return out
}

// GradleUserFile joins parts under Gradle's user home, or is empty with
// no home.
func GradleUserFile(parts ...string) string {
	dir := userHome()
	if dir == "" {
		return ""
	}
	return filepath.Join(append([]string{dir}, parts...)...)
}

// userHome is Gradle's user home: GRADLE_USER_HOME, else ~/.gradle. It is
// empty with no home to find, so no file is read from it.
func userHome() string {
	if dir := os.Getenv("GRADLE_USER_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gradle")
}
