package comments

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Judge adds what judge finds in each judged block, named `path:line` and read as Markdown, to res.
func Judge(res checkkit.Result, root string, judge func(name, markdown string) []string) checkkit.Result {
	blocks, err := Judged(root)
	return judgeBlocks(res, blocks, err, func(_ Block, name, markdown string) []string { return judge(name, markdown) })
}

// JudgeEach is Judge for a judge that reads each block after Place places it.
func JudgeEach(res checkkit.Result, root string, judge func(b Block, name, markdown string) []string) checkkit.Result {
	blocks, err := Judged(root)
	return judgeBlocks(res, blocks, errors.Join(err, Place(root, blocks)), judge)
}

// judgeBlocks adds to res what judge finds in blocks, and err as info.
func judgeBlocks(res checkkit.Result, blocks []Block, err error, judge func(b Block, name, markdown string) []string) checkkit.Result {
	if err != nil {
		res.Info = append(res.Info, err.Error())
	}
	files := map[string]bool{}
	for _, b := range blocks {
		files[b.Path] = true
		if errs := judge(b, fmt.Sprintf("%s:%d", b.Path, b.Line), Markdown(b.Text)); len(errs) > 0 {
			res.Ok = false
			res.Errors = append(res.Errors, errs...)
		}
	}
	res.FilesChecked += len(files)
	return res
}

// Judged returns the in-scope comment blocks, only those touching a changed line in a commit or pull request run.
func Judged(root string) ([]Block, error) {
	blocks, err := Files(root, walkfs.InScope(walkfs.FilesByExt(root, "")))
	if checkkit.DiffFiles() == nil {
		return blocks, err
	}
	kept, diffErr := onChangedLines(root, blocks)
	return kept, errors.Join(err, diffErr)
}

// onChangedLines keeps the blocks that touch a line changed since the
// run's base, asking git nothing when no block is left. With no base, as in a first commit, every line is new.
func onChangedLines(root string, blocks []Block) ([]Block, error) {
	if len(blocks) == 0 {
		return blocks, nil
	}
	base := gitx.ScopeBase(root)
	if base == "" {
		return blocks, nil
	}
	changed, err := gitx.Repo(root).ChangedLines(base, paths(blocks)...)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(blocks, func(b Block) bool { return !touches(changed[b.Path], b) }), nil
}

// paths lists the files blocks come from, once each.
func paths(blocks []Block) []string {
	var out []string
	for _, b := range blocks {
		if len(out) == 0 || out[len(out)-1] != b.Path {
			out = append(out, b.Path)
		}
	}
	return out
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

// Markdown is a block's text as Markdown, with indented code fenced so no check scores it.
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
	out                []string
	code, list, author bool
}

// line adds one line, joining an indented one to an open list item and fencing other indented code.
func (w *markdown) line(line string) {
	if w.authored(line) {
		w.out = append(w.out, line)
		return
	}
	_, item := mdx.ListItem(strings.TrimSpace(line))
	indented := isIndented(line)
	if indented && !item && w.list {
		w.out[len(w.out)-1] += " " + strings.TrimSpace(line)
		return
	}
	w.fence(w.isCode(item, indented))
	w.list = item
	w.out = append(w.out, line)
}

// isCode reports an indented line that neither starts nor carries on a
// list item.
func (w *markdown) isCode(item, indented bool) bool {
	return indented && !item && !w.list
}

// authored reports a line of a fence the comment writes, markers included, and ends any open list at a marker.
func (w *markdown) authored(line string) bool {
	if strings.HasPrefix(strings.TrimSpace(line), "```") {
		w.fence(false)
		w.author = !w.author
		w.list = false
		return true
	}
	return w.author
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
