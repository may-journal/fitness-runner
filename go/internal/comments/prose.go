package comments

import (
	"fmt"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Judge adds to res what judge finds in each comment block the run
// judges, as a prose check reads a Markdown file. judge gets the block
// named `path:line` and its text as Markdown. Each source file with a
// judged block counts as one more file checked.
func Judge(res checkkit.Result, root string, judge func(name, markdown string) []string) checkkit.Result {
	blocks, err := Judged(root)
	if err != nil {
		res.Info = append(res.Info, err.Error())
	}
	files := map[string]bool{}
	for _, b := range blocks {
		files[b.Path] = true
		if errs := judge(fmt.Sprintf("%s:%d", b.Path, b.Line), Markdown(b.Text)); len(errs) > 0 {
			res.Ok = false
			res.Errors = append(res.Errors, errs...)
		}
	}
	res.FilesChecked += len(files)
	return res
}

// Judged returns the comment blocks of the run's in-scope files. A commit
// or pull request run keeps the blocks that touch a line it changes. So it
// judges the comments it writes and no older one.
func Judged(root string) ([]Block, error) {
	blocks, err := Files(root, walkfs.InScope(walkfs.FilesByExt(root, "")))
	if err != nil || checkkit.DiffFiles() == nil {
		return blocks, err
	}
	return onChangedLines(root, blocks)
}

// onChangedLines keeps the blocks that touch a line changed since the
// run's base. With no base, as in a first commit, every line is new.
func onChangedLines(root string, blocks []Block) ([]Block, error) {
	base := gitx.ScopeBase(root)
	if base == "" {
		return blocks, nil
	}
	changed, err := gitx.Repo(root).ChangedLines(base)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(blocks, func(b Block) bool { return !touches(changed[b.Path], b) }), nil
}

// touches reports a run of changed lines inside block b.
func touches(changed []gitx.Lines, b Block) bool {
	for _, c := range changed {
		if c.From <= b.End && c.To >= b.Line {
			return true
		}
	}
	return false
}

// Markdown is a block's text as the Markdown a prose check reads. An
// indented line that carries on no list item is code, which goes in a
// fence, so no check scores it.
func Markdown(text string) string {
	w := &markdown{}
	for _, line := range strings.Split(text, "\n") {
		w.line(line)
	}
	w.fence(false)
	return strings.Join(w.out, "\n")
}

// markdown collects a comment's lines, tracking an open code fence and an
// open list.
type markdown struct {
	out        []string
	code, list bool
}

// line adds one line. An indented line carries on an open list, else it
// is code; a blank or unindented line ends the list.
func (w *markdown) line(line string) {
	_, item := mdx.ListItem(strings.TrimSpace(line))
	indented := isIndented(line)
	w.fence(indented && !item && !w.list)
	w.list = item || indented && w.list
	w.out = append(w.out, line)
}

// fence opens or closes the code fence so it is open exactly when code is.
func (w *markdown) fence(code bool) {
	if code != w.code {
		w.out, w.code = append(w.out, "```"), code
	}
}

// isIndented reports a line with text that starts with a space or tab.
func isIndented(line string) bool {
	return strings.TrimSpace(line) != "" && (line[0] == ' ' || line[0] == '\t')
}
