// Command fitness-check-changelog-bullets keeps changelog entries tight:
// EVERY section of CHANGELOG.md must have 3 to 5 bullets, every bullet
// must be under 365 characters, and every bullet must start with a
// capitalized semantic type prefix (Feat:, Fix:, …). A repo without a
// CHANGELOG.md has nothing to judge and passes with zero files.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

const (
	minBullets = 3
	maxBullets = 5
	maxRunes   = 365
)

var (
	headingRe = regexp.MustCompile(`^### `)
	typeRe    = regexp.MustCompile(`^(Feat|Fix|Docs|Style|Refactor|Perf|Test|Build|Ci|Chore|Revert): `)
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "changelog-bullets"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	raw, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		return checkkit.Pass(0), nil
	}
	errs := judge(string(raw))
	if len(errs) > 0 {
		return checkkit.Fail(1, errs...), nil
	}
	return checkkit.Pass(1), nil
}

// bullet is one logical changelog bullet: its 1-based start line and its
// full text with continuation lines joined by single spaces.
type bullet struct {
	line int
	text string
}

// judge validates every section's bullets against all three rules.
func judge(content string) []string {
	var errs []string
	for _, sec := range sections(content) {
		errs = append(errs, judgeSection(sec)...)
	}
	return errs
}

// judgeSection applies the bullet-count rule to one section and the
// per-bullet rules to each of its bullets.
func judgeSection(sec section) []string {
	var errs []string
	if n := len(sec.bullets); n < minBullets || n > maxBullets {
		errs = append(errs, fmt.Sprintf(
			"CHANGELOG.md section %q has %d bullets; keep entries to %d-%d bullets", sec.heading, n, minBullets, maxBullets))
	}
	for _, b := range sec.bullets {
		errs = append(errs, judgeBullet(b)...)
	}
	return errs
}

// judgeBullet applies the length and type-prefix rules to one bullet.
func judgeBullet(b bullet) []string {
	var errs []string
	if n := len([]rune(b.text)); n >= maxRunes {
		errs = append(errs, fmt.Sprintf(
			"CHANGELOG.md:%d: bullet is %d characters; keep each under %d", b.line, n, maxRunes))
	}
	if !typeRe.MatchString(b.text) {
		errs = append(errs, fmt.Sprintf(
			"CHANGELOG.md:%d: bullet must start with a semantic type (Feat:, Fix:, Docs:, Style:, Refactor:, Perf:, Test:, Build:, Ci:, Chore:, Revert:)", b.line))
	}
	return errs
}

// section is one ### entry: its trimmed heading text and logical bullets.
type section struct {
	heading string
	bullets []bullet
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
// new section; other lines belong to the current one.
func fold(out []section, line string, lineNo int) []section {
	if headingRe.MatchString(line) {
		return append(out, section{heading: strings.TrimSpace(strings.TrimPrefix(line, "### "))})
	}
	if len(out) == 0 {
		return out
	}
	cur := &out[len(out)-1]
	cur.bullets = collect(cur.bullets, line, lineNo)
	return out
}

// collect folds one raw line into the bullet list: a "- " line starts a new
// bullet; any other non-blank line continues the current one.
func collect(bullets []bullet, line string, lineNo int) []bullet {
	if strings.HasPrefix(line, "- ") {
		return append(bullets, bullet{line: lineNo, text: strings.TrimPrefix(line, "- ")})
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || len(bullets) == 0 {
		return bullets
	}
	last := &bullets[len(bullets)-1]
	last.text += " " + trimmed
	return bullets
}
