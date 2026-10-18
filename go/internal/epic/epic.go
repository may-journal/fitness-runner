// Package epic reads an Epic issue body: the issue references its `## What
// needs to happen` section lists in run order, and the rules each item keeps.
// The epic-structure check judges the list, and `fitness issue-check`
// compares its references with the epic's GitHub sub-issues.
package epic

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/may-journal/fitness-runner/go/internal/mdx"
)

// Section is the heading whose list names the epic's issues.
const Section = "What needs to happen"

// Item is one list item under the section.
type Item struct {
	// Text is the item after its list marker.
	Text string
	// Group is the ### heading above the item, or "" before any.
	Group string
	// Ref is the plain reference the item names, or "" when it is not one.
	Ref string
	// Note is the text after the reference's comma.
	Note string
	// After lists the references the note says the item runs after.
	After []string
}

var (
	itemRe     = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(.*\S)\s*$`)
	refPattern = `(?:[\w.-]+/[\w.-]+)?#\d+`
	plainRe    = regexp.MustCompile(`^(` + refPattern + `)(?:\s*,\s*(.+))?$`)
	refRe      = regexp.MustCompile(refPattern)
	afterRe    = regexp.MustCompile(`(?i)\bafter\b`)
	checkboxRe = regexp.MustCompile(`^\[[ xX]\]`)
	groupRe    = regexp.MustCompile(`^###\s+(.*\S)`)
	statusRe   = regexp.MustCompile(`(?i)\b(done|blocked|wip|todo|merged|closed|shipped|complete|completed|in progress|pending)\b`)
)

// Items returns every list item in the section, in order, with the ###
// heading each sits under. Fenced code never counts.
func Items(body string) []Item {
	group := ""
	var items []Item
	for _, text := range sectionLines(body) {
		group, items = addLine(text, group, items)
	}
	return items
}

// sectionLines returns the section's lines below its heading, with fenced
// code blanked.
func sectionLines(body string) []string {
	start, end := bounds(body)
	if start == 0 {
		return nil
	}
	lines := mdx.NonFencedLines(body)
	if end == 0 {
		end = len(lines) + 1
	}
	var out []string
	for _, ln := range lines[start : end-1] {
		out = append(out, ln.Text)
	}
	return out
}

// addLine reads one section line: a ### heading starts a group, and a list
// item joins the current one.
func addLine(text, group string, items []Item) (string, []Item) {
	if m := groupRe.FindStringSubmatch(strings.TrimSpace(text)); m != nil {
		return m[1], items
	}
	if m := itemRe.FindStringSubmatch(text); m != nil {
		items = append(items, parse(m[1], group))
	}
	return group, items
}

// Refs returns the plain references the section lists, in order.
func Refs(body string) []string {
	var refs []string
	for _, it := range Items(body) {
		if it.Ref != "" {
			refs = append(refs, it.Ref)
		}
	}
	return refs
}

// Looks reports whether body reads as an epic: its first item under the
// section names an issue, with or without a checkbox.
func Looks(body string) bool {
	items := Items(body)
	if len(items) == 0 {
		return false
	}
	text := strings.TrimSpace(checkboxRe.ReplaceAllString(items[0].Text, ""))
	loc := refRe.FindStringIndex(text)
	return loc != nil && loc[0] == 0
}

// parse reads one item's reference and note.
func parse(text, group string) Item {
	it := Item{Text: text, Group: group}
	m := plainRe.FindStringSubmatch(text)
	if m == nil {
		return it
	}
	it.Ref, it.Note = m[1], m[2]
	if loc := afterRe.FindStringIndex(it.Note); loc != nil {
		it.After = refRe.FindAllString(it.Note[loc[1]:], -1)
	}
	return it
}

// bounds returns the section heading's line and the next level-2 heading's
// line, 0 when the section runs to the end; start is 0 with no section.
func bounds(body string) (start, end int) {
	for _, h := range mdx.Headings(body) {
		if h.Level != 2 {
			continue
		}
		if start > 0 {
			return start, h.Line
		}
		if h.Text == Section {
			start = h.Line
		}
	}
	return start, 0
}

// Validate returns the list's findings: each item that is not a plain
// reference, a repeated reference, an item listed before one it runs after,
// a table anywhere, and a list group longer than maxItems.
func Validate(body string, maxItems int) []string {
	start, _ := bounds(body)
	if start == 0 {
		return tableErrors(body)
	}
	items := Items(body)
	if len(items) == 0 {
		return append([]string{"`## " + Section + "` lists no issue; add one #N reference per item"}, tableErrors(body)...)
	}
	var errs []string
	for i, it := range items {
		errs = append(errs, itemErrors(i+1, it)...)
	}
	errs = append(errs, repeatErrors(items)...)
	errs = append(errs, orderErrors(items)...)
	errs = append(errs, tableErrors(body)...)
	return append(errs, sizeErrors(items, maxItems)...)
}

// itemErrors judges one item's form: a plain reference with an optional
// short note, and no checkbox, emoji or status.
func itemErrors(n int, it Item) []string {
	switch {
	case checkboxRe.MatchString(it.Text):
		return []string{fmt.Sprintf("item %d uses a checkbox; list the issue as a plain reference, and GitHub shows whether it is done", n)}
	case hasEmoji(it.Text):
		return []string{fmt.Sprintf("item %d has an emoji; GitHub shows each issue's state, so drop it", n)}
	case it.Ref == "":
		return []string{fmt.Sprintf("item %d (%s) is not a plain #N or org/repo#N reference; put any short note after a comma", n, it.Text)}
	case statusRe.MatchString(it.Note):
		return []string{fmt.Sprintf("item %d (%s) notes a status; GitHub shows each issue's state, so drop it", n, it.Ref)}
	}
	return nil
}

// repeatErrors names each reference listed more than once.
func repeatErrors(items []Item) []string {
	var errs []string
	seen := map[string]bool{}
	for _, it := range items {
		key := strings.ToLower(it.Ref)
		if it.Ref != "" && seen[key] {
			errs = append(errs, fmt.Sprintf("%s is listed more than once; list each issue once", it.Ref))
		}
		seen[key] = true
	}
	return errs
}

// orderErrors names each item listed before an item its note says it runs
// after.
func orderErrors(items []Item) []string {
	at := positions(items)
	var errs []string
	for i, it := range items {
		errs = append(errs, afterErrors(i, it, at)...)
	}
	return errs
}

// positions maps each listed reference, in lower case, to its first index.
func positions(items []Item) map[string]int {
	at := map[string]int{}
	for i, it := range slices.Backward(items) {
		if it.Ref != "" {
			at[strings.ToLower(it.Ref)] = i
		}
	}
	return at
}

// afterErrors names each reference item i runs after that is listed later.
func afterErrors(i int, it Item, at map[string]int) []string {
	var errs []string
	for _, a := range it.After {
		if j, ok := at[strings.ToLower(a)]; ok && j > i {
			errs = append(errs, fmt.Sprintf("item %d (%s) is listed before %s, which it runs after; list them in run order", i+1, it.Ref, a))
		}
	}
	return errs
}

// tableErrors fails any table in the body.
func tableErrors(body string) []string {
	for _, ln := range mdx.NonFencedLines(body) {
		if mdx.IsTableSeparator(ln.Text) {
			return []string{"tables are not allowed; list each issue as a plain reference"}
		}
	}
	return nil
}

// sizeErrors fails each group of items longer than maxItems.
func sizeErrors(items []Item, maxItems int) []string {
	groups, count := groupCounts(items)
	var errs []string
	for _, g := range groups {
		if count[g] > maxItems {
			errs = append(errs, sizeError(g, count[g], maxItems))
		}
	}
	return errs
}

// groupCounts returns the groups in order and how many items each holds.
func groupCounts(items []Item) ([]string, map[string]int) {
	var groups []string
	count := map[string]int{}
	for _, it := range items {
		if count[it.Group] == 0 {
			groups = append(groups, it.Group)
		}
		count[it.Group]++
	}
	return groups, count
}

// sizeError is the finding for group g holding n items over maxItems.
func sizeError(g string, n, maxItems int) string {
	where := "`## " + Section + "`"
	if g != "" {
		where = "`### " + g + "`"
	}
	return fmt.Sprintf("%s lists %d issues (max %d); split them under ### headings", where, n, maxItems)
}

// emoji holds the pictograph and dingbat blocks emoji come from.
var emoji = &unicode.RangeTable{
	R16: []unicode.Range16{{Lo: 0x2600, Hi: 0x27BF, Stride: 1}, {Lo: 0x2B50, Hi: 0x2B50, Stride: 1}, {Lo: 0x2B55, Hi: 0x2B55, Stride: 1}},
	R32: []unicode.Range32{{Lo: 0x1F000, Hi: 0x1FAFF, Stride: 1}},
}

// hasEmoji reports whether text holds an emoji.
func hasEmoji(text string) bool {
	return strings.IndexFunc(text, func(r rune) bool { return unicode.Is(emoji, r) }) >= 0
}
