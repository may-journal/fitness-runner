// cspell:ignore autonumber Descr gantt mindmap sankey treemap xychart zenuml

package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "mermaid-syntax", Scope: checkkit.ScopeFiles, Blob: true},
		Run:      run,
	})
}

func run(root string, args []string) (checkkit.Result, error) {
	if res, handled, err := bodycheck.RunDoc(root, args, func(_, content string) []string {
		return validateDoc("(description)", content)
	}); err != nil || handled {
		return res, err
	}
	errors, filesChecked, err := mermaid.RunDocCheck(root, validateDoc)
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errors) > 0 {
		return checkkit.Fail(filesChecked, errors...), nil
	}
	return checkkit.Pass(filesChecked), nil
}

// validateDoc parses every mermaid diagram in content and names each
// problem with its document line.
func validateDoc(file, content string) []string {
	var errors []string
	for _, block := range mermaid.ParseDoc(content) {
		if d, ok := block.(*mermaid.DiagramBlock); ok {
			for _, p := range parseDiagram(d.Body) {
				errors = append(errors, fmt.Sprintf("%s: line %d: %s (diagram at line %d)", file, d.Line+p.line, p.msg, d.Line))
			}
		}
	}
	return errors
}

// problem is one syntax error at a 1-based line of the diagram body.
type problem struct {
	line int
	msg  string
}

// srcLine is one statement-bearing line of a diagram body.
type srcLine struct {
	n    int
	text string
}

// grammar parses a diagram's statements after its header line.
type grammar func(header srcLine, lines []srcLine) []problem

// grammars are the diagram types parsed statement by statement.
var grammars = map[string]grammar{
	"flowchart":       flowchart,
	"graph":           flowchart,
	"flowchart-elk":   flowchart,
	"sequenceDiagram": sequence,
}

// balancedTypes are the other mermaid diagram types; only their brackets
// and quotes are checked.
// A full grammar for each is out of reach.
var balancedTypes = map[string]bool{
	"classDiagram": true, "classDiagram-v2": true, "stateDiagram": true, "stateDiagram-v2": true,
	"journey": true, "gantt": true, "pie": true, "quadrantChart": true, "requirementDiagram": true,
	"gitGraph": true, "C4Context": true, "C4Container": true, "C4Component": true, "C4Dynamic": true,
	"C4Deployment": true, "timeline": true, "zenuml": true, "sankey-beta": true,
	"xychart-beta": true, "xychart": true, "sankey": true, "packet": true, "packet-beta": true, "kanban": true,
	"architecture-beta": true, "radar-beta": true, "treemap-beta": true, "wardley-beta": true,
}

// headerOnlyTypes are known types whose syntax leaves brackets unpaired,
// such as erDiagram's `||--o{` cardinality, mindmap's `)cloud(` shape, or
// block-beta's `>flag]` shape.
// Only the header is checked.
var headerOnlyTypes = map[string]bool{"erDiagram": true, "mindmap": true, "block-beta": true, "block": true}

// parseDiagram judges one diagram body.
func parseDiagram(body string) []problem {
	lines := statements(body)
	if len(lines) == 0 {
		return []problem{{1, "empty diagram: start with a diagram type such as flowchart TD"}}
	}
	header := lines[0]
	kind := strings.Fields(header.text)[0]
	if g, ok := grammars[kind]; ok {
		return g(header, lines[1:])
	}
	return otherType(kind, header, lines)
}

// otherType checks a diagram mermaid knows but this check does not parse.
func otherType(kind string, header srcLine, lines []srcLine) []problem {
	if balancedTypes[kind] {
		return balance(lines)
	}
	if headerOnlyTypes[kind] {
		return nil
	}
	return []problem{{header.n, fmt.Sprintf("unknown diagram type %q", kind)}}
}

// statements drops front matter, %% comments and directives, and blank
// lines, keeping each remaining line's number.
func statements(body string) []srcLine {
	raw := strings.Split(body, "\n")
	start := frontMatterEnd(raw)
	var out []srcLine
	for i := start; i < len(raw); i++ {
		text := strings.TrimSpace(raw[i])
		if accBlock.MatchString(text) {
			i = blockEnd(raw, i)
			continue
		}
		if text != "" && !strings.HasPrefix(text, "%%") {
			out = append(out, srcLine{i + 1, text})
		}
	}
	return out
}

// accBlock opens a multi-line accessible description, which holds free text.
var accBlock = regexp.MustCompile(`^accDescr\s*\{[^}]*$`)

// blockEnd is the index of the line that closes the block opened at i, or
// the last line when none does.
func blockEnd(raw []string, i int) int {
	for j := i + 1; j < len(raw); j++ {
		if strings.Contains(raw[j], "}") {
			return j
		}
	}
	return len(raw) - 1
}

// frontMatterEnd is the index after a leading `---` front matter block, or 0.
func frontMatterEnd(raw []string) int {
	if len(raw) == 0 || strings.TrimSpace(raw[0]) != "---" {
		return 0
	}
	for i := 1; i < len(raw); i++ {
		if strings.TrimSpace(raw[i]) == "---" {
			return i + 1
		}
	}
	return 0
}

// pairs maps each closing bracket to its opener.
var pairs = map[rune]rune{')': '(', ']': '[', '}': '{'}

// opener is an unclosed bracket and where it opened.
type opener struct {
	r    rune
	line int
}

// balance reports the first unmatched bracket or unclosed quote, outside
// double-quoted strings.
func balance(lines []srcLine) []problem {
	var stack []opener
	for _, l := range lines {
		var p *problem
		if stack, p = balanceLine(stack, l); p != nil {
			return []problem{*p}
		}
	}
	if len(stack) > 0 {
		o := stack[len(stack)-1]
		return []problem{{o.line, fmt.Sprintf("%q is never closed", o.r)}}
	}
	return nil
}

// balanceLine folds one line's brackets into stack.
func balanceLine(stack []opener, l srcLine) ([]opener, *problem) {
	text, ok := unquoted(l.text)
	if !ok {
		return stack, &problem{l.n, "a double quote is never closed"}
	}
	for _, r := range text {
		var bad bool
		if stack, bad = push(stack, r, l.n); bad {
			return stack, &problem{l.n, fmt.Sprintf("%q closes nothing", r)}
		}
	}
	return stack, nil
}

// push applies one rune to the bracket stack and reports a stray closer.
func push(stack []opener, r rune, line int) ([]opener, bool) {
	if strings.ContainsRune("([{", r) {
		return append(stack, opener{r, line}), false
	}
	want, closer := pairs[r]
	if !closer {
		return stack, false
	}
	if len(stack) == 0 || stack[len(stack)-1].r != want {
		return stack, true
	}
	return stack[:len(stack)-1], false
}

// unquoted blanks every double-quoted string in text, reporting false when
// a quote never closes.
func unquoted(text string) (string, bool) {
	parts := strings.Split(text, `"`)
	if len(parts)%2 == 0 {
		return "", false
	}
	var b strings.Builder
	for i := 0; i < len(parts); i += 2 {
		b.WriteString(parts[i])
	}
	return b.String(), true
}
