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

// Judge adds to res what judge finds in each comment block the run
// judges, as a prose check reads a Markdown file. judge gets the block
// named `path:line` and its text as Markdown. Each source file with a
// judged block counts as one more file checked.
func Judge(res checkkit.Result, root string, judge func(name, markdown string) []string) checkkit.Result {
	blocks, err := Judged(root)
	return judgeBlocks(res, blocks, err, func(_ Block, name, markdown string) []string { return judge(name, markdown) })
}

// JudgeEach is Judge for a judge that also reads each block, as Place sets
// it. Place reads only the files of the judged blocks.
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

// Judged returns the comment blocks of the run's in-scope files. A commit
// or pull request run keeps the blocks that touch a line it changes. So it
// judges the comments it writes and no older one.
func Judged(root string) ([]Block, error) {
	blocks, err := Files(root, walkfs.InScope(walkfs.FilesByExt(root, "")))
	if checkkit.DiffFiles() == nil {
		return blocks, err
	}
	kept, diffErr := onChangedLines(root, blocks)
	return kept, errors.Join(err, diffErr)
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
// fence, so no check scores it. A fence the comment writes stays as is.
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

// line adds one line. An indented line carries on an open list, joining
// the item it wraps, else it is code. A blank or unindented line ends the
// list.
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

// authored reports a line inside a fence the comment writes, its opening
// and closing lines included, which passes through unchanged. A fence ends
// any open list, so no later line joins the fence.
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
