// The italic scan is hand-rolled because the TS original relied on JavaScript look-around assertions.
// The flagged snippet is quoted byte-for-byte like JSON.stringify.

package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "markdown-no-bold-italic", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsNotApplicable, Blob: true},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	if res, handled, err := bodycheck.RunDoc(root, args, func(_, content string) []string {
		return checkkit.Lines(errorLine, validateFile("(description)", content))
	}); handled || err != nil {
		return res, err
	}
	return walkFiles(root)
}

// walkFiles flags bold/italic in every in-scope .md file under root.
func walkFiles(root string) (checkkit.Result, error) {
	var found []checkkit.Finding
	filesChecked := 0
	for _, file := range walkfs.InScope(walkfs.FilesByExt(root, ".md")) {
		filesChecked++
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return checkkit.Result{}, err
		}
		found = append(found, validateFile(file, string(raw))...)
	}
	return checkkit.Found(filesChecked, errorLine, found...), nil
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

// singleDelimiterMatches scans single-delimiter emphasis spans like the JS look-around patterns, by hand because RE2 lacks look-around; excludeNewline keeps asterisk italic on one line.
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

// emphasisBodyEnd returns the index of the first delimiter after i, or of a newline when excludeNewline is set, else len(s).
func emphasisBodyEnd(s string, i int, delim byte, excludeNewline bool) int {
	j := i + 1
	for j < len(s) && s[j] != delim && !(excludeNewline && s[j] == '\n') {
		j++
	}
	return j
}

// closesEmphasis reports whether j closes a span opened at i with a non-empty body and no doubled delimiter.
func closesEmphasis(s string, i, j int, delim byte) bool {
	if j == i+1 || j == len(s) || s[j] != delim {
		return false
	}
	return !(j+1 < len(s) && s[j+1] == delim)
}

// validateFile validates one markdown file; returns error messages for
// disallowed bold/italic, verbatim from the TS template.
func validateFile(relPath, content string) []checkkit.Finding {
	var found []checkkit.Finding
	if path.Base(relPath) == "CHANGELOG.md" {
		content = plainReleaseScopes(content)
	}
	for _, h := range findDisallowedEmphasis(content) {
		found = append(found, checkkit.Finding{
			File: relPath, Rule: rule, Fix: fix,
			Message: fmt.Sprintf("disallowed %s (use only when explicitly required): %s", h.kind, jsonStringify(h.match)),
		})
	}
	return found
}

// rule names what the check enforces: markdown uses no bold or italic.
const rule = "no-bold-italic"

// fix says how to pass: drop the emphasis markers and keep the words.
const fix = "remove the emphasis markers and keep the words; let structure such as headings and lists carry the weight"

// errorLine renders a finding as "<path>: <message>".
func errorLine(f checkkit.Finding) string {
	return f.File + ": " + f.Message
}

// Release Please writes each release bullet as `* **scope:** text`. Inside
// a CHANGELOG release section, from a ## version heading to the next dated
// ### entry, that bold scope is its format, so it is allowed.
var (
	releaseHeadingRe = regexp.MustCompile(`^## \[?\d`)
	datedEntryRe     = regexp.MustCompile(`^### \d{4}\.\d{2}\.\d{2}\.\d{4}`)
	boldScopeRe      = regexp.MustCompile(`^\* \*\*([^*]+):\*\* `)
)

// plainReleaseScopes drops the bold from release bullets' scopes, leaving
// every other line, and any other bold, for the check to judge.
func plainReleaseScopes(content string) string {
	lines := strings.Split(content, "\n")
	inRelease := false
	for i, line := range lines {
		inRelease = (inRelease || releaseHeadingRe.MatchString(line)) && !datedEntryRe.MatchString(line)
		if inRelease {
			lines[i] = boldScopeRe.ReplaceAllString(line, "* $1: ")
		}
	}
	return strings.Join(lines, "\n")
}

// jsonStringify quotes s as JavaScript's JSON.stringify does, leaving <, >, and & unescaped and escaping only U+2028/U+2029 above ASCII.
func jsonStringify(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	q := strings.TrimSuffix(buf.String(), "\n")
	q = strings.ReplaceAll(q, "\u2028", `\u2028`)
	return strings.ReplaceAll(q, "\u2029", `\u2029`)
}
