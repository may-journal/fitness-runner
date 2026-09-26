// Command fitness-check-markdown-no-bold-italic validates that markdown
// files do not use **bold**, __bold__, *italic*, or _italic_ emphasis — the
// Go port of the markdown-no-bold-italic check. Emphasis inside fenced code
// blocks, inline code, and markdown links is exempt, as is the root
// CHANGELOG.md. The italic patterns carry JavaScript look-behind and
// look-ahead assertions the TS original relied on, so their scan is
// hand-rolled here; the flagged snippet is quoted byte-for-byte like
// JSON.stringify.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "markdown-no-bold-italic"},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	if res, handled, err := bodycheck.RunDoc(root, args, func(_, content string) []string {
		return validateFile("(description)", content)
	}); handled || err != nil {
		return res, err
	}
	return walkFiles(root)
}

// walkFiles flags bold/italic in every .md file under root except the
// changelog.
func walkFiles(root string) (checkkit.Result, error) {
	var errors []string
	filesChecked := 0
	for _, file := range walkfs.FilesByExt(root, ".md") {
		if file == "CHANGELOG.md" {
			continue
		}
		filesChecked++
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return checkkit.Result{}, err
		}
		errors = append(errors, validateFile(file, string(raw))...)
	}
	if len(errors) > 0 {
		return checkkit.Fail(filesChecked, errors...), nil
	}
	return checkkit.Pass(filesChecked), nil
}

var (
	// fencedRe matches ```…``` blocks, non-greedy across lines.
	fencedRe = regexp.MustCompile("(?s)```.*?```")
	// inlineCodeRe matches `…` inline code spans.
	inlineCodeRe = regexp.MustCompile("`[^`]*`")
	// linkRe matches [text](url) so underscores in URLs or link text are
	// not flagged as italic.
	linkRe = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)
	// boldAsteriskRe matches **bold** (asterisk).
	boldAsteriskRe = regexp.MustCompile(`\*\*[^*]*\*\*`)
	// boldUnderscoreRe matches __bold__ (underscore).
	boldUnderscoreRe = regexp.MustCompile(`__[^_]*__`)
)

// stripCodeForEmphasisCheck removes fenced code blocks (```...```) and
// inline code (`...`) so emphasis inside code is not flagged.
func stripCodeForEmphasisCheck(content string) string {
	return inlineCodeRe.ReplaceAllString(fencedRe.ReplaceAllString(content, "\n"), " ")
}

// hit is one disallowed emphasis occurrence: the rule kind and the matched
// snippet.
type hit struct {
	kind  string
	match string
}

// findDisallowedEmphasis returns all disallowed markdown emphasis matches in
// content (bold/italic); ignores content inside code and links. Hits are
// grouped by rule in the TS order: **bold**, __bold__, *italic*, _italic_.
func findDisallowedEmphasis(content string) []hit {
	stripped := linkRe.ReplaceAllString(stripCodeForEmphasisCheck(content), " ")
	var out []hit
	for _, m := range boldAsteriskRe.FindAllString(stripped, -1) {
		out = append(out, hit{kind: "**bold**", match: m})
	}
	for _, m := range boldUnderscoreRe.FindAllString(stripped, -1) {
		out = append(out, hit{kind: "__bold__", match: m})
	}
	for _, m := range singleDelimiterMatches(stripped, '*', true) {
		out = append(out, hit{kind: "*italic*", match: m})
	}
	for _, m := range singleDelimiterMatches(stripped, '_', false) {
		out = append(out, hit{kind: "_italic_", match: m})
	}
	return out
}

// singleDelimiterMatches scans for single-delimiter emphasis spans the way
// the JS patterns /(?<!\*)\*[^*\n]+\*(?!\*)/ and /(?<!_)_[^_]+_(?!_)/ do:
// an opening delimiter not preceded by the same delimiter, a non-empty body
// free of the delimiter, and a closing delimiter not followed by another.
// Go's RE2 has no look-behind or look-ahead, so the exec-loop semantics
// (advance one position on failure, resume after a match) are reproduced by
// hand.
// excludeNewline mirrors the asterisk body class [^*\n]+ — asterisk italic
// must not span lines, which is what exempts * list markers.
func singleDelimiterMatches(s string, delim byte, excludeNewline bool) []string {
	var out []string
	for i := 0; i < len(s); i++ {
		if !opensEmphasis(s, i, delim) {
			continue
		}
		j := emphasisBodyEnd(s, i, delim, excludeNewline)
		if !closesEmphasis(s, i, j, delim) {
			continue
		}
		out = append(out, s[i:j+1])
		i = j // the loop increment resumes scanning just past the match
	}
	return out
}

// opensEmphasis reports whether position i starts a candidate span: the
// delimiter itself, not preceded by the same delimiter (the JS look-behind).
func opensEmphasis(s string, i int, delim byte) bool {
	return s[i] == delim && (i == 0 || s[i-1] != delim)
}

// emphasisBodyEnd scans the span body from i+1 and returns the index of the
// first delimiter — or newline when excludeNewline, the asterisk body class
// [^*\n]+ — or len(s) when the body never closes.
func emphasisBodyEnd(s string, i int, delim byte, excludeNewline bool) int {
	j := i + 1
	for j < len(s) && s[j] != delim && !(excludeNewline && s[j] == '\n') {
		j++
	}
	return j
}

// closesEmphasis reports whether j closes a span opened at i: a non-empty
// body, the closing delimiter reached, and no doubled delimiter after it
// (the JS look-ahead).
func closesEmphasis(s string, i, j int, delim byte) bool {
	if j == i+1 || j == len(s) || s[j] != delim {
		return false
	}
	return !(j+1 < len(s) && s[j+1] == delim)
}

// validateFile validates one markdown file; returns error messages for
// disallowed bold/italic, verbatim from the TS template.
func validateFile(relPath, content string) []string {
	var errors []string
	for _, h := range findDisallowedEmphasis(content) {
		errors = append(errors, fmt.Sprintf(
			"%s: disallowed %s (use only when explicitly required): %s",
			relPath, h.kind, jsonStringify(h.match)))
	}
	return errors
}

// jsonStringify quotes s exactly as JavaScript's JSON.stringify does:
// unlike json.Marshal it leaves <, >, and & unescaped, and unlike
// strconv.Quote it escapes U+2028/U+2029 and nothing else above ASCII.
func jsonStringify(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	q := strings.TrimSuffix(buf.String(), "\n")
	q = strings.ReplaceAll(q, "\u2028", `\u2028`)
	return strings.ReplaceAll(q, "\u2029", `\u2029`)
}
