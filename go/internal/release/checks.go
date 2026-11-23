package release

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/docs"
)

// WriteChecks rewrites every file Generated returns, so no one keeps the
// generated Go file, the marked regions and llms.txt by hand.
func (c Config) WriteChecks() error {
	files, err := c.Generated()
	if err != nil {
		return err
	}
	for path, body := range files {
		if err := os.WriteFile(filepath.Join(c.Root, filepath.FromSlash(path)), body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Stale lists each generated file differing from what WriteChecks would
// write, with the total; an error means the generator cannot read something.
func (c Config) Stale() (stale []string, total int, err error) {
	files, err := c.Generated()
	if err != nil {
		return nil, 0, err
	}
	for _, path := range slices.Sorted(maps.Keys(files)) {
		differs, err := c.differs(path, files[path])
		if err != nil {
			return nil, 0, err
		}
		if differs {
			stale = append(stale, path)
		}
	}
	return stale, len(files), nil
}

// differs reports whether the file at path, relative to the root, holds
// other than want. A missing file differs; any other read error is returned.
func (c Config) differs(path string, want []byte) (bool, error) {
	got, err := os.ReadFile(filepath.Join(c.Root, filepath.FromSlash(path)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return !bytes.Equal(got, want), nil
}

// Generated returns each file the generator writes, by repository path. It
// puts the code's own facts in marked regions of Markdown and in usage.go.
func (c Config) Generated() (map[string][]byte, error) {
	in, err := c.readSources()
	if err != nil {
		return nil, err
	}
	return c.render(in)
}

// sources is what the generator reads: the code's facts, and the Markdown
// whose summaries the lists repeat.
type sources struct {
	topics, runner []docs.Topic
	usage          usageDoc
	configKeys     []string
}

// readSources reads the topics and runner pages in go/docs, the documented
// command line, and the config file's top-level keys.
func (c Config) readSources() (s sources, err error) {
	folder := os.DirFS(filepath.Join(c.Root, filepath.FromSlash(docsFolder)))
	linked := c.onlyLocalBranch()
	s.topics, err = docs.Read(folder, ".")
	runner, runnerErr := docs.Read(folder, docs.RunnerFolder)
	s.runner, runnerErr = runner, errors.Join(runnerErr, hasPages(runner))
	var usageErr, keysErr error
	s.usage, usageErr = c.usage()
	s.configKeys, keysErr = c.configKeys()
	return s, errors.Join(linked, err, runnerErr, usageErr, keysErr)
}

// render lays out every generated file: usage.go, the marked regions, and
// llms.txt, a copy of the llms page.
func (c Config) render(s sources) (map[string][]byte, error) {
	usage, err := usageSource(s.usage)
	checks := checkList(s.runner)
	page, pageErr := c.withRegions(llmsPage, []region{
		{runMarker, usageSourceNote, entryList(s.usage.Runner)},
		{commandsMarker, usageSourceNote, entryList(s.usage.Commands)},
		{checksMarker, checksSourceNote, checks},
		{topicsMarker, "the Markdown files in go/docs", topicList(s.topics)},
	})
	readme, readmeErr := c.withRegions(readmeFile, []region{
		{usageMarker, usageSourceNote, usageBlock(s.usage)},
		{configMarker, "the json tags of Config in go/pkg/conf/conf.go", keyList(s.configKeys)},
	})
	checksDoc, checksErr := c.withRegions(checksDocFile, []region{{checksMarker, checksSourceNote, checks}})
	if err := errors.Join(err, pageErr, readmeErr, checksErr); err != nil {
		return nil, err
	}
	return map[string][]byte{usageFile: usage, llmsPage: page, llmsFile: page, readmeFile: readme, checksDocFile: checksDoc}, nil
}

// docsFolder holds the runner's Markdown, relative to the repository root.
const docsFolder = "go/docs"

// llmsPage is the page `fitness llms` prints, which llms.txt copies,
// relative to the repository root.
var llmsPage = path.Join(docsFolder, docs.RunnerFolder, "fitness.md")

// checksDocFile lists the checks the engine runs itself, among other notes
// on checks, relative to the repository root.
const checksDocFile = "docs/checks.md"

// The markers of the regions the generator writes.
const (
	runMarker      = "fitness run"
	commandsMarker = "fitness commands"
	checksMarker   = "fitness checks"
	topicsMarker   = "fitness topics"
	configMarker   = "fitness config"
)

// usageSourceNote and checksSourceNote name where a region's facts come from.
const (
	usageSourceNote  = "the usage comments in go/cmd/fitness and go/internal/selection"
	checksSourceNote = "the check folders under go/cmd and go/docs/runner"
)

// localBranchFolder is the package of local-branch, the one check the
// engine runs itself, relative to the repository root.
const localBranchFolder = "go/cmd/fitness-check-local-branch"

// onlyLocalBranch fails any check package under go/cmd beside local-branch,
// which the engine would never link: each other check lives in
// fitness-checks.
func (c Config) onlyLocalBranch() error {
	dirs, err := filepath.Glob(filepath.Join(c.Root, "go", "cmd", "fitness-check-*"))
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if rel := "go/cmd/" + filepath.Base(dir); rel != localBranchFolder {
			return fmt.Errorf("%s: the engine runs only local-branch; move this check to fitness-checks", rel)
		}
	}
	return nil
}

// checkList lists each check the engine runs itself with the summary its
// page in go/docs/runner opens with.
func checkList(runner []docs.Topic) string {
	var b strings.Builder
	for _, t := range runner {
		if "go/cmd/fitness-check-"+t.Name == localBranchFolder {
			fmt.Fprintf(&b, "- `%s`: %s\n", t.Name, t.Summary)
		}
	}
	return b.String()
}

// topicList lists each `fitness docs` topic with its summary.
func topicList(topics []docs.Topic) string {
	var b strings.Builder
	for _, t := range topics {
		fmt.Fprintf(&b, "- `fitness docs %s`: %s\n", t.Name, t.Summary)
	}
	return b.String()
}

// Input reports whether a changed path can change the generator's output:
// a Go source, Markdown in go/docs, or a file it writes.
func Input(path string) bool {
	written := []string{readmeFile, llmsFile, checksDocFile}
	return strings.HasSuffix(path, ".go") || strings.HasPrefix(path, docsFolder+"/") || slices.Contains(written, path)
}

// runnerPages are the pages the binary reads from go/docs/runner by name.
var runnerPages = []string{"fitness", "local-branch"}

// hasPages fails when a page the binary reads is missing from runner.
func hasPages(runner []docs.Topic) error {
	var errs []error
	for _, name := range runnerPages {
		if !slices.ContainsFunc(runner, func(t docs.Topic) bool { return t.Name == name }) {
			errs = append(errs, fmt.Errorf("go/docs/runner/%s.md is missing; the fitness binary reads it", name))
		}
	}
	return errors.Join(errs...)
}
