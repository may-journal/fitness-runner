package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/changelog"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

const (
	minBullets = 3
	maxBullets = 5
	maxRunes   = 365
)

var (
	headingRe = regexp.MustCompile(`^### `)
	// A bullet opens with a capitalized type (`Feat: `) or a semantic commit
	// subject (`feat(scope)!: `), lowercase with an optional scope and bang.
	typeRe = regexp.MustCompile(`^((Feat|Fix|Docs|Style|Refactor|Perf|Test|Build|Ci|Chore|Revert)|(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([^()\s]+\))?!?): `)
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "changelog-bullets", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsNotApplicable},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	entries, err := changelog.ReadEntries(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	var errs []string
	for _, e := range entries {
		errs = append(errs, judge(e.Path, e.Text)...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(entries), errs...), nil
	}
	return checkkit.Pass(len(entries)), nil
}

// bullet is one logical changelog bullet: its 1-based start line and its
// full text with continuation lines joined by single spaces.
type bullet struct {
	line int
	text string
}

// judge validates every section's bullets against all three rules.
func judge(file, content string) []string {
	var errs []string
	for _, sec := range sections(content) {
		errs = append(errs, judgeSection(file, sec)...)
	}
	return errs
}

// judgeSection applies the bullet-count rule to one section of file and the
// per-bullet rules to each of its bullets.
func judgeSection(file string, sec section) []string {
	var errs []string
	if n := len(sec.bullets); n < minBullets || n > maxBullets {
		errs = append(errs, fmt.Sprintf(
			"%s section %q has %d bullets; keep entries to %d-%d bullets", file, sec.heading, n, minBullets, maxBullets))
	}
	for _, b := range sec.bullets {
		errs = append(errs, judgeBullet(file, b)...)
	}
	return errs
}

// judgeBullet applies the length and type-prefix rules to one bullet.
func judgeBullet(file string, b bullet) []string {
	var errs []string
	if n := len([]rune(b.text)); n >= maxRunes {
		errs = append(errs, fmt.Sprintf(
			"%s:%d: bullet is %d characters; keep each under %d", file, b.line, n, maxRunes))
	}
	if !typeRe.MatchString(b.text) {
		errs = append(errs, fmt.Sprintf(
			"%s:%d: bullet must start with a semantic type (Feat:, Fix:, ... or feat(scope):, fix(scope):, ...)", file, b.line))
	}
	return errs
}

// section is one ### entry: its trimmed heading text and logical bullets.
type section struct {
	heading string
	bullets []bullet
}

// sections splits the document into ### sections, joining indented
// continuation lines into their bullet. Lines above the first heading,
// such as front matter, belong to none.
func sections(content string) []section {
	var out []section
	for i, line := range strings.Split(content, "\n") {
		if headingRe.MatchString(line) {
			out = append(out, section{heading: strings.TrimSpace(strings.TrimPrefix(line, "### "))})
		} else if len(out) > 0 {
			cur := &out[len(out)-1]
			cur.bullets = collect(cur.bullets, line, i+1)
		}
	}
	return out
}

// collect folds one raw line into the bullet list: a "- " line starts a new
// bullet; any other non-blank line continues the current one.
func collect(bullets []bullet, line string, lineNo int) []bullet {
	if text, ok := strings.CutPrefix(line, "- "); ok {
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
