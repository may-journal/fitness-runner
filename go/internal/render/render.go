// Package render draws the results table the fitness suite has always
// printed: box-drawn, columns Check/Status/Files/Time, a full-width
// word-wrapped error block per failed check, and a bold totals line.
// Geometry and coloring mirror the TypeScript cli-table3 output: column
// widths 28/10/8/max(10, terminal-51) including one space of padding per
// side, gray borders and red header regardless of tty (unless NO_COLOR),
// chalk-style status colors only on a terminal.
package render

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Row is one check's rendered result.
type Row struct {
	Name string
	Ok   bool
	// FilesChecked below zero renders as "-".
	FilesChecked int
	Ms           int64
	// Errors renders as a full-width block when the row failed.
	Errors []string
}

// Palette carries the escape codes; zero value = no color.
type Palette struct {
	gray, head, off         string
	red, green, boldOn, rst string
	boldGreenOff            string
}

// ColorsEnabled reports the two color gates every fitness surface shares:
// decoration (borders/header; off only under NO_COLOR, even when piped,
// exactly like cli-table3) and chalk-style colors (a terminal on stderr or
// FORCE_COLOR).
func ColorsEnabled() (decoration, chalk bool) {
	if os.Getenv("NO_COLOR") != "" {
		return false, false
	}
	return true, os.Getenv("FORCE_COLOR") != "" || isTerminal(os.Stderr)
}

// NewPalette builds the palette from the shared color gates.
func NewPalette() Palette {
	var p Palette
	decoration, chalk := ColorsEnabled()
	if decoration {
		p.gray, p.head, p.off = "\x1b[90m", "\x1b[31m", "\x1b[39m"
	}
	if chalk {
		p.red, p.green = "\x1b[31m", "\x1b[32m"
		p.rst = "\x1b[39m"
		p.boldOn = "\x1b[1m"
		p.boldGreenOff = "\x1b[22m"
	}
	return p
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// TermCols returns the terminal width when attached to one, else 80.
func TermCols() int {
	if !isTerminal(os.Stderr) {
		return 80
	}
	out, err := exec.Command("stty", "size").Output()
	if err == nil {
		fields := strings.Fields(string(out))
		if len(fields) == 2 {
			if n, convErr := strconv.Atoi(fields[1]); convErr == nil && n > 0 {
				return n
			}
		}
	}
	if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n > 0 {
		return n
	}
	return 80
}

// Table renders the rows into the bordered results table.
func Table(rows []Row, cols int, p Palette) string {
	timeW := cols - 51
	if timeW < 10 {
		timeW = 10
	}
	widths := []int{28, 10, 8, timeW}
	var b strings.Builder
	border(&b, widths, "┌", "┬", "┐", p)
	head := []string{"Check", "Status", "Files", "Time"}
	headCells := make([]string, len(head))
	for i, h := range head {
		headCells[i] = p.head + pad(truncate(h, widths[i]-2), widths[i]-2) + p.off
	}
	rowLine(&b, headCells, p)
	for _, r := range rows {
		border(&b, widths, "├", "┼", "┤", p)
		files := "-"
		if r.FilesChecked >= 0 {
			files = strconv.Itoa(r.FilesChecked)
		}
		status := p.green + pad("passed", widths[1]-2) + p.rst
		if !r.Ok {
			status = p.red + pad("failed", widths[1]-2) + p.rst
		}
		cells := []string{
			pad(truncate(r.Name, widths[0]-2), widths[0]-2),
			status,
			pad(truncate(files, widths[2]-2), widths[2]-2),
			pad(truncate(fmt.Sprintf("%dms", r.Ms), widths[3]-2), widths[3]-2),
		}
		rowLine(&b, cells, p)
		if !r.Ok && len(r.Errors) > 0 {
			border(&b, widths, "├", "┼", "┤", p)
			spanW := widths[0] + widths[1] + widths[2] + widths[3] + 3 - 2
			lines := []string{fmt.Sprintf("[%s] Please fix these items:", r.Name)}
			for _, e := range r.Errors {
				lines = append(lines, "  ✖ "+e)
			}
			for _, logical := range lines {
				for _, wrapped := range wrap(logical, spanW) {
					b.WriteString(p.gray + "│" + p.off + " " +
						p.red + pad(wrapped, spanW) + p.rst + " " + p.gray + "│" + p.off + "\n")
				}
			}
		}
	}
	border(&b, widths, "└", "┴", "┘", p)
	return strings.TrimSuffix(b.String(), "\n")
}

// TotalLine formats the bold summary line.
func TotalLine(success, failure, files int, ms int64, p Palette) string {
	line := fmt.Sprintf("Total: %d succeeded, %d failed, %d files in %dms", success, failure, files, ms)
	color := p.green
	if failure > 0 {
		color = p.red
	}
	if color == "" {
		return line
	}
	return p.boldOn + color + line + p.rst + p.boldGreenOff
}

func border(b *strings.Builder, widths []int, left, mid, right string, p Palette) {
	b.WriteString(p.gray + left)
	for i, w := range widths {
		if i > 0 {
			b.WriteString(mid)
		}
		b.WriteString(strings.Repeat("─", w))
	}
	b.WriteString(right + p.off + "\n")
}

func rowLine(b *strings.Builder, cells []string, p Palette) {
	sep := p.gray + "│" + p.off
	b.WriteString(sep)
	for _, c := range cells {
		b.WriteString(" " + c + " " + sep)
	}
	b.WriteString("\n")
}

// truncate shortens s to width runes, cli-table3 style: '…' as the last rune.
func truncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width < 1 {
		return ""
	}
	return string(runes[:width-1]) + "…"
}

// pad right-pads s with spaces to width runes.
func pad(s string, width int) string {
	n := width - len([]rune(s))
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}

// wrap word-wraps s to width runes, breaking long words when unavoidable.
// Leading whitespace (the "  ✖ " bullet indent) stays on the first line.
func wrap(s string, width int) []string {
	if len([]rune(s)) <= width {
		return []string{s}
	}
	trimmed := strings.TrimLeft(s, " ")
	words := strings.Split(trimmed, " ")
	if prefix := s[:len(s)-len(trimmed)]; prefix != "" && len(words) > 0 {
		words[0] = prefix + words[0]
	}
	var out []string
	line := ""
	for _, word := range words {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if len([]rune(candidate)) <= width {
			line = candidate
			continue
		}
		if line != "" {
			out = append(out, line)
		}
		for len([]rune(word)) > width {
			r := []rune(word)
			out = append(out, string(r[:width]))
			word = string(r[width:])
		}
		line = word
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
