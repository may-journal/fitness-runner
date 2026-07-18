// Command fitness-check-changelog-bullets keeps changelog entries tight:
// the FIRST (newest) section of CHANGELOG.md must have 3 to 5 bullets,
// every bullet must be under 365 characters, and every bullet must start
// with a capitalized semantic type prefix (Feat:, Fix:, …). Older sections
// are history and are not judged. A repo without a CHANGELOG.md has
// nothing to judge and passes with zero files.
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

// judge validates the first section's bullets against all three rules.
func judge(content string) []string {
	bullets := firstSectionBullets(content)
	var errs []string
	if n := len(bullets); n < minBullets || n > maxBullets {
		errs = append(errs, fmt.Sprintf(
			"CHANGELOG.md first section has %d bullets; keep entries to %d-%d bullets", n, minBullets, maxBullets))
	}
	for _, b := range bullets {
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

// firstSectionBullets returns the logical bullets of the first ### section,
// joining indented continuation lines into their bullet.
func firstSectionBullets(content string) []bullet {
	lines := strings.Split(content, "\n")
	start := sectionStart(lines)
	if start < 0 {
		return nil
	}
	var bullets []bullet
	for i := start + 1; i < len(lines); i++ {
		line := lines[i]
		if headingRe.MatchString(line) || strings.HasPrefix(line, "## ") {
			break
		}
		bullets = collect(bullets, line, i+1)
	}
	return bullets
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

// sectionStart returns the index of the first ### heading, or -1.
func sectionStart(lines []string) int {
	for i, line := range lines {
		if headingRe.MatchString(line) {
			return i
		}
	}
	return -1
}
