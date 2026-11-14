package check

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/questions"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "open-questions", Scope: checkkit.ScopeFiles, Blob: true},
		Run:      run,
	})
}

// htmlExts are the endings of the pages the check reads by their tags.
var htmlExts = []string{".html", ".htm"}

func run(root string, args []string) (checkkit.Result, error) {
	if res, handled, err := bodycheck.RunDoc(root, args, func(_, content string) []string {
		return checkkit.Lines(errorLine, judge("(description)", content, false))
	}); handled || err != nil {
		return res, err
	}
	return walkFiles(root)
}

// walkFiles judges every in-scope Markdown and HTML file under root.
func walkFiles(root string) (checkkit.Result, error) {
	files := walkfs.InScope(walkfs.FilesByExt(root, slices.Concat(mdx.Exts, htmlExts)...))
	var found []checkkit.Finding
	for _, file := range files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return checkkit.Result{}, err
		}
		found = append(found, judge(file, string(raw), mdx.HasExt(file, htmlExts))...)
	}
	return checkkit.Found(len(files), errorLine, found...), nil
}

// judge returns a finding for each line of content that opens a list of
// open questions.
func judge(file, content string, html bool) []checkkit.Finding {
	var found []checkkit.Finding
	for _, hit := range questions.Find(content, html) {
		found = append(found, checkkit.Finding{
			File: file, Line: hit.Line, Rule: rule, Fix: fix,
			Message: fmt.Sprintf("a %s opens a list of open questions: %q", hit.Kind, hit.Text),
		})
	}
	return found
}

// rule names what the check enforces: open questions go to chat.
const rule = "questions-in-chat"

// fix says how to pass.
const fix = "ask the owner in chat, or turn each unknown into a checklist step"

// errorLine renders a finding as "<path>:<line>: <message> — <fix>".
func errorLine(f checkkit.Finding) string {
	return fmt.Sprintf("%s:%d: %s — %s", f.File, f.Line, f.Message, f.Fix)
}
