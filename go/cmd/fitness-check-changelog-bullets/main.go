// Command fitness-check-changelog-bullets keeps changelog entries tight:
// EVERY section of CHANGELOG.md must have 3 to 5 bullets, every bullet
// must be under 365 characters, and every bullet must start with a
// capitalized semantic type prefix (Feat:, Fix:, …). A repo without a
// CHANGELOG.md has nothing to judge and passes with zero files. Each
// changelog/*.md entry file is judged the same way.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/changelog"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
)

const (
	minBullets = 3
	maxBullets = 5
	maxRunes   = 365
)

var (
	headingRe = regexp.MustCompile(`^### `)
	releaseRe = regexp.MustCompile(`^## \[?\d`)
	datedRe   = regexp.MustCompile(`^### \d{4}\.\d{2}\.\d{2}\.\d{4}`)
	// A bullet opens with a capitalized type (`Feat: `) or a semantic commit
	// subject (`feat(scope)!: `), lowercase with an optional scope and bang.
	typeRe = regexp.MustCompile(`^((Feat|Fix|Docs|Style|Refactor|Perf|Test|Build|Ci|Chore|Revert)|(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([^()\s]+\))?!?): `)
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "changelog-bullets"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	docs, err := changelogDocs(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	var errs []string
	for _, doc := range docs {
		errs = append(errs, judge(doc.Path, doc.Text, conf.MaxListItems(root))...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(docs), errs...), nil
	}
	return checkkit.Pass(len(docs)), nil
}

// changelogDocs are the files this check judges: CHANGELOG.md when present,
// then each entry file.
func changelogDocs(root string) ([]changelog.Entry, error) {
	var docs []changelog.Entry
	if raw, err := os.ReadFile(filepath.Join(root, changelog.File)); err == nil {
		docs = append(docs, changelog.Entry{Path: changelog.File, Text: string(raw)})
	}
	entries, err := changelog.ReadEntries(root)
	return append(docs, entries...), err
}

// bullet is one logical changelog bullet: its 1-based start line and its
// full text with continuation lines joined by single spaces.
type bullet struct {
	line int
	text string
}

// judge validates every section's bullets against all three rules. A
// release section gathers many changes, so it may hold from one bullet up to
// the repo's prose-budget list limit, releaseMax.
func judge(file, content string, releaseMax int) []string {
	var errs []string
	for _, sec := range sections(content) {
		errs = append(errs, judgeSection(file, sec, releaseMax)...)
	}
	return errs
}

// judgeSection applies the bullet-count rule to one section of file and the
// per-bullet rules to each of its bullets.
func judgeSection(file string, sec section, releaseMax int) []string {
	var errs []string
	lo, hi, kind := minBullets, maxBullets, "entries"
	if sec.release {
		lo, hi, kind = 1, releaseMax, "releases"
	}
	if n := len(sec.bullets); n < lo || n > hi {
		errs = append(errs, fmt.Sprintf(
			"%s section %q has %d bullets; keep %s to %d-%d bullets", file, sec.heading, n, kind, lo, hi))
	}
	for _, b := range sec.bullets {
		errs = append(errs, judgeBullet(file, b, sec.release)...)
	}
	return errs
}

// bulletText is a bullet line's text: "- " anywhere, or "* " in a release.
func bulletText(line string, release bool) (string, bool) {
	if text, ok := strings.CutPrefix(line, "- "); ok {
		return text, true
	}
	if text, ok := strings.CutPrefix(line, "* "); ok && release {
		return text, true
	}
	return "", false
}

// judgeBullet applies the length rule to one bullet, and the type-prefix
// rule to an entry's bullet.
func judgeBullet(file string, b bullet, release bool) []string {
	var errs []string
	if n := len([]rune(b.text)); n >= maxRunes {
		errs = append(errs, fmt.Sprintf(
			"%s:%d: bullet is %d characters; keep each under %d", file, b.line, n, maxRunes))
	}
	// A release bullet takes its type from the subheading above it.
	if !release && !typeRe.MatchString(b.text) {
		errs = append(errs, fmt.Sprintf(
			"%s:%d: bullet must start with a semantic type (Feat:, Fix:, ... or feat(scope):, fix(scope):, ...)", file, b.line))
	}
	return errs
}

// section is one ### entry, or a ## release section: its trimmed heading
// text and logical bullets.
type section struct {
	heading string
	bullets []bullet
	release bool
}

// sections splits the document into ### sections, joining indented
// continuation lines into their bullet.
func sections(content string) []section {
	var out []section
	for i, line := range strings.Split(content, "\n") {
		out = fold(out, line, i+1)
	}
	return out
}

// fold accumulates one raw line into the section list: a ### line opens a
// new section, a ## release heading opens a release section, and other
// lines belong to the current one.
func fold(out []section, line string, lineNo int) []section {
	if subheading(out, line) {
		return out
	}
	if headingRe.MatchString(line) {
		return append(out, section{heading: strings.TrimSpace(strings.TrimPrefix(line, "### "))})
	}
	if releaseRe.MatchString(line) {
		return append(out, section{heading: strings.TrimSpace(strings.TrimPrefix(line, "## ")), release: true})
	}
	if len(out) == 0 {
		return out
	}
	cur := &out[len(out)-1]
	cur.bullets = collect(cur.bullets, line, lineNo, cur.release)
	return out
}

// subheading reports whether line is an undated ### heading inside a release
// section, such as Release Please's `### Features`; its bullets stay with
// the release.
func subheading(out []section, line string) bool {
	return len(out) > 0 && out[len(out)-1].release && headingRe.MatchString(line) && !datedRe.MatchString(line)
}

// collect folds one raw line into the bullet list: a "- " line starts a new
// bullet, as does a "* " line in a release section, where Release Please
// writes them; any other non-blank line continues the current one.
func collect(bullets []bullet, line string, lineNo int, release bool) []bullet {
	if text, ok := bulletText(line, release); ok {
		return append(bullets, bullet{line: lineNo, text: text})
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || len(bullets) == 0 {
		return bullets
	}
	last := &bullets[len(bullets)-1]
	last.text += " " + trimmed
	return bullets
}
