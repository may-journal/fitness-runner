// cspell:ignore antml Handback

// Command fitness-check-comment-structure judges one GitHub issue or PR
// comment: a word budget outside tables, code, and diagrams, at most one
// plain table, numbered diagrams, and no change-log notes, agent frames, or
// model identifiers.
//
// It reads its target from stdin, a `--body-file` path (`-` meaning stdin),
// or the runner's context-inline `--body` value, and passes inert with no
// input. Under `fitness --blob` it skips Plan, Epic, and PR bodies, which have
// their own structure checks.
package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "comment-structure", Blob: true, ContextInlineArg: "--body"},
		Run:      run,
	})
}

// Default limits, which .fitnessrc.json's comment key may replace.
const (
	defaultMaxWords  = 100
	defaultMaxTables = 1
)

// limits is one repo's comment budget.
type limits struct{ words, tables int }

func run(root string, args []string) (checkkit.Result, error) {
	lim, err := loadLimits(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	return bodycheck.RunIf(args, isComment, func(body string) []string {
		return validate(body, lim)
	})
}

// isComment reports whether a draft is a comment rather than a Plan, Epic,
// or PR body.
func isComment(body string) bool {
	return !bodycheck.PlanBody(body) && !bodycheck.EpicBody(body) && !bodycheck.PRBody(body)
}

// loadLimits reads the repo's comment budget over the defaults.
func loadLimits(root string) (limits, error) {
	lim := limits{defaultMaxWords, defaultMaxTables}
	cfg, err := conf.Load(root)
	if err != nil || cfg == nil {
		return lim, err
	}
	lim.words = positiveOr(cfg.Comment.MaxWords, lim.words)
	lim.tables = positiveOr(cfg.Comment.MaxTables, lim.tables)
	return lim, nil
}

func positiveOr(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

// validate runs every comment rule on body.
func validate(body string, lim limits) []string {
	errs := budget(body, lim)
	errs = append(errs, unnumbered(body)...)
	errs = append(errs, lineRules(body)...)
	return errs
}

// budget checks the word count outside tables, code, and diagrams, and the
// number of plain tables; callout tables belong to their diagrams.
func budget(body string, lim limits) []string {
	var errs []string
	if n := mdx.WordCount(mdx.Prose(body)); n > lim.words {
		errs = append(errs, fmt.Sprintf("(comment): %d words outside tables, code, and diagrams; the budget is %d, so cut it or move detail into a table", n, lim.words))
	}
	if n := plainTables(body); n > lim.tables {
		errs = append(errs, fmt.Sprintf("(comment): %d tables besides callout tables; the budget is %d", n, lim.tables))
	}
	return errs
}

// plainTables counts the tables outside code whose header is not a callout
// header.
func plainTables(body string) int {
	lines := mdx.NonFencedLines(body)
	n := 0
	for i := 1; i < len(lines); i++ {
		if isPlainTable(lines[i-1].Text, lines[i].Text) {
			n++
		}
	}
	return n
}

// isPlainTable reports whether header and separator open a non-callout
// table.
func isPlainTable(header, separator string) bool {
	return strings.Contains(header, "|") && mdx.IsTableSeparator(separator) && !mermaid.IsCalloutHeader(mdx.SplitTableRow(header))
}

// unnumbered fails a diagram with no numbered callouts, unless it is the
// legend right after a numbered diagram.
func unnumbered(body string) []string {
	var errs []string
	var prev mermaid.DocBlock
	for _, b := range mermaid.ParseDoc(body) {
		if d, ok := b.(*mermaid.DiagramBlock); ok && len(d.Numbers) == 0 && !numberedDiagram(prev) {
			errs = append(errs, fmt.Sprintf("(comment): line %d: mermaid diagram has no numbered callouts; number its nodes, as in A[1 Step], and add a # | Element | Description | Why table", d.Line))
		}
		prev = b
	}
	return errs
}

// numberedDiagram reports whether b is a diagram with callout numbers.
func numberedDiagram(b mermaid.DocBlock) bool {
	d, ok := b.(*mermaid.DiagramBlock)
	return ok && len(d.Numbers) > 0
}

// lineRule is one pattern a comment line must not match, and why.
type lineRule struct {
	re *regexp.Regexp
	// fenced also judges lines inside code fences.
	fenced bool
	why    string
}

// rules are the patterns a comment may not hold.
var rules = []lineRule{
	{regexp.MustCompile(`(?i)^\W*(?:edit|edited|update|updated|revised|revision|correction|addendum)(?:\s*\([^)]*\)|\s+\d[\w/.-]*)?\s*:`), false,
		"change-log note; edit the text in place and drop the note"},
	{regexp.MustCompile(`(?i)\((?:edited|updated|revised)\b[^)]*\)|\b(?:replaces|supersedes) (?:the|my) (?:earlier|previous)\b`), false,
		"change-log note; edit the text in place and drop the note"},
	{regexp.MustCompile(`</?(?:system-reminder|function_calls|function_results|invoke|parameter|task-notification|thinking)\b|\bantml:|\bSubagentHandback\b|^(?:Human|Assistant): `), true,
		"agent frame; post only the finished text"},
	{regexp.MustCompile(`(?i)\bclaude-(?:opus|sonnet|haiku|fable|instant|\d)[\w.-]*|\bgpt-\d[\w.-]*|\bgemini-\d[\w.-]*`), false,
		"model identifier; name the agent, not the model"},
	{regexp.MustCompile(regexp.QuoteMeta(aftereffect.MarkerPrefix)), false,
		"carries a fitness marker; only fitness writes those, so remove it"},
}

// lineRules checks every line against rules, naming each line once per
// rule.
func lineRules(body string) []string {
	raw := strings.Split(body, "\n")
	prose := mdx.NonFencedLines(body)
	var errs []string
	for _, r := range rules {
		errs = append(errs, r.find(raw, prose)...)
	}
	return errs
}

// inlineCode is a code span or a URL, which may name a model or quote a note.
var inlineCode = regexp.MustCompile("`[^`]*`|https?://\\S+")

// outsideCode blanks the code spans and URLs in a prose line.
func outsideCode(line string) string {
	return inlineCode.ReplaceAllString(line, " ")
}

// find names each line r matches.
func (r lineRule) find(raw []string, prose []mdx.TextLine) []string {
	var errs []string
	for i, l := range prose {
		text := outsideCode(l.Text)
		if r.fenced {
			text = raw[i]
		}
		if m := r.re.FindString(text); m != "" {
			errs = append(errs, fmt.Sprintf("(comment): line %d: %q: %s", i+1, strings.TrimSpace(m), r.why))
		}
	}
	return errs
}
