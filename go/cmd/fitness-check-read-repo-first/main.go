// Command fitness-check-read-repo-first prints the familiarization banner —
// the Go port of the read-repo-first check. The banner (the familiarization
// question, a Check|Src table of the enabled checks, the --no-verify note)
// is display, not judgment: the result is always ok with zero files checked.
// Protocol adaptations from the TypeScript original: the banner goes to
// stderr (stdout is the JSON result channel), enabled names come from
// FITNESS_ENABLED_CHECKS instead of an in-process run context (standalone
// runs with no context print the table-less form), and the folder-override
// map (checkFolderByName) does not cross the env protocol, so Src is always
// go/cmd/fitness-check-<name>/README.md. Table geometry and coloring mirror the
// cli-table3 output: column widths 28 and max(20, max(70, width-8)-34)
// including one space of padding per side, gray borders and red header
// regardless of tty (unless NO_COLOR), chalk-style bold/cyan/yellow only on
// a terminal.
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/render"
)

const noteNoVerify = "NOTE: Do not under any circumstance use `--no-verify`" +
	" as it will cause issues downstream, fixing locally is your best bet."

const questionLine1 = "Did you familiarize yourself with the decisions logged in the repo,"
const questionLine2 = `specifically all "Fitness Checks" that are enabled via @mayjournal/fitness?`

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "read-repo-first"},
		Run:      run,
	})
}

// run writes the banner to stderr; the check itself always passes and
// counts no files.
func run(_ string, _ []string) (checkkit.Result, error) {
	banner := buildContextFeedback(checkkit.EnabledChecks(), render.TermCols(), newPalette())
	os.Stderr.WriteString(banner)
	return checkkit.Pass(0), nil
}

// palette carries the escape codes; zero value = no color. Decoration
// (borders, header) colors even when piped, exactly like cli-table3;
// chalk-style codes appear only on a terminal, exactly like chalk.
type palette struct {
	gray, head, off               string // table decoration: on unless NO_COLOR
	boldOn, boldOff               string // chalk styles: terminal only
	white, cyan, yellow, colorOff string
}

// newPalette builds the banner palette from the shared render color gates.
func newPalette() palette {
	var p palette
	decoration, chalk := render.ColorsEnabled()
	if decoration {
		p.gray, p.head, p.off = "\x1b[90m", "\x1b[31m", "\x1b[39m"
	}
	if chalk {
		p.boldOn, p.boldOff = "\x1b[1m", "\x1b[22m"
		p.white, p.cyan, p.yellow, p.colorOff = "\x1b[37m", "\x1b[36m", "\x1b[33m", "\x1b[39m"
	}
	return p
}

// buildContextFeedback renders the boxed banner: the bold familiarization
// question, the enabled-check table (omitted when no checks are known), and
// the yellow --no-verify note, between full-width cyan horizontal rules.
func buildContextFeedback(names []string, cols int, p palette) string {
	ruleWidth := cols
	if ruleWidth < 20 {
		ruleWidth = 20
	}
	rule := p.cyan + strings.Repeat("─", ruleWidth) + p.colorOff
	question := p.boldOn + questionLine1 + p.boldOff + "\n" +
		p.boldOn + questionLine2 + p.boldOff
	lines := []string{question}
	if len(names) > 0 {
		lines = append(lines, "", checkTable(names, cols, p))
	}
	lines = append(lines, "", p.yellow+noteNoVerify+p.colorOff)
	return rule + "\n" + strings.Join(lines, "\n") + "\n" + rule + "\n"
}

// checkTable draws the two-column Check|Src table, one row per enabled
// check, each Src the check's README path go/cmd/fitness-check-<name>/README.md.
func checkTable(names []string, cols int, p palette) string {
	first, second := tableWidths(cols)
	var b strings.Builder
	borderLine(&b, "┌", "┬", "┐", first, second, p)
	headerLine(&b, first, second, p)
	for _, name := range names {
		borderLine(&b, "├", "┼", "┤", first, second, p)
		src := filepath.Join("go", "cmd", "fitness-check-"+name, "README.md")
		rowLine(&b, name, src, first, second, p)
	}
	borderLine(&b, "└", "┴", "┘", first, second, p)
	return strings.TrimSuffix(b.String(), "\n")
}

// tableWidths mirrors the TS geometry: first column fixed at 28, second
// max(20, max(70, cols-8) - 28 - 6); widths include the padding spaces.
func tableWidths(cols int) (first, second int) {
	first = 28
	maxTotal := cols - 8
	if maxTotal < 70 {
		maxTotal = 70
	}
	second = maxTotal - first - 6
	if second < 20 {
		second = 20
	}
	return first, second
}

// borderLine writes one horizontal border in cli-table3's exact byte form:
// two gray spans, the first covering the left corner and first segment.
func borderLine(b *strings.Builder, left, mid, right string, first, second int, p palette) {
	b.WriteString(p.gray + left + strings.Repeat("─", first) + p.off)
	b.WriteString(p.gray + mid + strings.Repeat("─", second) + right + p.off + "\n")
}

// headerLine writes the Check|Src header row: red cells spanning the
// padding, bold-white labels inside on a terminal.
func headerLine(b *strings.Builder, first, second int, p palette) {
	sep := p.gray + "│" + p.off
	b.WriteString(sep + headCell("Check", first-2, p) + sep + headCell("Src", second-2, p) + sep + "\n")
}

func headCell(h string, width int, p palette) string {
	label := p.boldOn + p.white + h + p.colorOff + p.boldOff
	return p.head + " " + label + strings.Repeat(" ", width-len(h)) + " " + p.off
}

func rowLine(b *strings.Builder, name, src string, first, second int, p palette) {
	sep := p.gray + "│" + p.off
	b.WriteString(sep + " " + padCell(name, first-2) + " " + sep +
		" " + padCell(src, second-2) + " " + sep + "\n")
}

// padCell right-pads s with spaces to width runes; overlong content
// truncates cli-table3 style, '…' as the last rune.
func padCell(s string, width int) string {
	runes := []rune(s)
	if len(runes) > width {
		return string(runes[:width-1]) + "…"
	}
	return s + strings.Repeat(" ", width-len(runes))
}
