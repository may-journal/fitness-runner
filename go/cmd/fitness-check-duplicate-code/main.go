package check

import (
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/clonedetect"
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
	"github.com/may-journal/fitness-runner/go/internal/par"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// The thresholds, modeled on jscpd's --threshold 1 --min-lines 5
// --min-tokens 50.
const (
	threshold = 1.0
	minLines  = 5
	minTokens = 50
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "duplicate-code", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	files := lexTargets(root, scanTargets(root))
	stats := clonedetect.Detect(files, clonedetect.Options{MinLines: minLines, MinTokens: minTokens})
	if pct := stats.Percentage(); pct > threshold {
		errs := []string{fmt.Sprintf(
			"ERROR: duplicate code is %.1f%% of scanned lines, over the %.1f%% threshold",
			pct, threshold)}
		for _, f := range stats.Files {
			errs = append(errs, fmt.Sprintf("%s: %d duplicated lines", f.Path, f.Lines))
		}
		return checkkit.Fail(len(files), errs...), nil
	}
	return checkkit.Pass(len(files)), nil
}

// scanTargets lists the candidates under root, minus those .gitattributes
// marks linguist-generated, the lockfiles and build wrapper files tools
// write, and the templates doc-template makes docs repeat.
func scanTargets(root string) []string {
	var out []string
	toolWritten := lang.IsToolWritten(langs.All)
	for _, p := range walkfs.WithoutGenerated(root, candidates(root)) {
		if !mdtemplate.IsTemplate(p) && !toolWritten(p) {
			out = append(out, p)
		}
	}
	return out
}

// candidates lists the tracked files a run may compare. An unscoped run
// takes every tracked file (the empty suffix matches all). The check
// compares files of one format, so a scoped run takes only the tracked
// files sharing an ending with a changed file, the changed files among
// them.
func candidates(root string) []string {
	changed := checkkit.ChangedFiles()
	if changed == nil {
		return walkfs.FilesByExt(root, "")
	}
	endings := map[string]bool{}
	for _, file := range changed {
		endings[ending(file)] = true
	}
	return slices.DeleteFunc(walkfs.FilesByExt(root, slices.Collect(maps.Keys(endings))...), func(file string) bool {
		return !endings[ending(file)]
	})
}

// ending is the format key the check groups a file by: its extension, or its
// whole name when it has none, such as Makefile.
func ending(file string) string {
	if ext := path.Ext(file); ext != "" {
		return ext
	}
	return path.Base(file)
}

// lexTargets reads and lexes each path, skipping binary files and files
// that vanish mid-scan; what remains is what "scanned" means, so its
// length is the check's filesChecked.
func lexTargets(root string, paths []string) []clonedetect.File {
	lexed := par.Map(len(paths), 0, func(i int) *clonedetect.File {
		return lexOne(root, paths[i])
	})
	var files []clonedetect.File
	for _, f := range lexed {
		if f != nil {
			files = append(files, *f)
		}
	}
	return files
}

// lexOne reads and lexes a single path for the parallel pool; nil marks a
// binary or vanished file (skipped, not scanned).
func lexOne(root, rel string) *clonedetect.File {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil || isBinary(raw) {
		return nil
	}
	content := string(raw)
	return &clonedetect.File{
		Path:   rel,
		Tokens: clonedetect.Lex(content),
		Lines:  clonedetect.CountLines(content),
	}
}

// isBinary applies git's heuristic: a NUL byte in the first 8000 bytes.
func isBinary(raw []byte) bool {
	probe := raw
	if len(probe) > 8000 {
		probe = probe[:8000]
	}
	for _, b := range probe {
		if b == 0 {
			return true
		}
	}
	return false
}
