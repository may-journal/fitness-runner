package runtimes

import (
	"path"
	"slices"
	"strings"
)

// runners run the file named after them, past flags and a run
// subcommand, as in "node --watch scripts/seed.mjs".
var runners = strings.Fields("node tsx ts-node vite-node bun deno zx sh bash zsh python python3")

// valueFlags are runner flags whose value is the next word, as in
// "node -r dotenv/config server.js".
var valueFlags = strings.Fields("-r --require --import --loader --experimental-loader --env-file --watch-path -C --conditions --tsconfig")

// launchers start the command after them, as in "npx tsx scripts/a.ts".
var launchers = strings.Fields("npx exec env pnpm yarn dlx")

// devRun collects what the package.json files of a repository run and
// publish while DevScripts reads them.
type devRun struct {
	tracked, dev, start map[string]bool
	published           []string
}

// DevScripts lists the tracked files that are tools by how a repository
// runs them: files under the top-level scripts folder, and files only
// package.json scripts run, such as tools/gen.mjs in "node tools/gen.mjs".
// A file the start script runs, or one the package publishes, stays
// product code.
func DevScripts(root string, files []string) map[string]bool {
	r := newDevRun(files)
	for _, file := range files {
		if path.Base(file) == "package.json" {
			r.scripts(root, file)
		}
	}
	return r.tools()
}

// newDevRun starts with every file under the top-level scripts folder.
func newDevRun(files []string) *devRun {
	r := &devRun{tracked: map[string]bool{}, dev: map[string]bool{}, start: map[string]bool{}}
	for _, file := range files {
		r.tracked[file] = true
		r.dev[file] = strings.HasPrefix(file, "scripts/")
	}
	return r
}

// tools lists the dev files the start script never runs and the package
// never publishes.
func (r *devRun) tools() map[string]bool {
	tools := map[string]bool{}
	for file, dev := range r.dev {
		tools[file] = dev && !r.start[file] && !published(file, r.published)
	}
	return tools
}

// scripts adds the files each script of one package.json runs, to start
// for the start script and to dev for the others, and keeps its
// published entries. The containers check reports a package.json it
// cannot parse.
func (r *devRun) scripts(root, file string) {
	pkg, _, err := readPackage(root, file)
	if err != nil {
		return
	}
	for name, command := range pkg.Scripts {
		into := r.dev
		if name == "start" {
			into = r.start
		}
		r.mark(file, command, into)
	}
	r.published = append(r.published, pkg.entries(file)...)
}

// mark adds each tracked file a script command runs to into.
func (r *devRun) mark(file, command string, into map[string]bool) {
	for _, run := range commandRuns(command) {
		if p := beside(file, run); r.tracked[p] {
			into[p] = true
		}
	}
}

// published reports a file an entry names: the entry itself, a file in
// a published folder such as lib, or one under an exports pattern such
// as ./lib/* before its star.
func published(file string, entries []string) bool {
	for _, entry := range entries {
		if star := strings.Index(entry, "*"); star >= 0 {
			entry = path.Dir(entry[:star] + "x")
		}
		if file == entry || strings.HasPrefix(file, entry+"/") {
			return true
		}
	}
	return false
}

// commandRuns lists the program each command in a script line runs:
// the first word past launchers and variables, or the file a runner
// such as node runs.
func commandRuns(line string) []string {
	for _, op := range []string{"&&", "||", "|", ";", "&", "(", ")"} {
		line = strings.ReplaceAll(line, op, "\n")
	}
	var runs []string
	for _, command := range strings.Split(line, "\n") {
		if run := program(strings.Fields(command)); run != "" {
			runs = append(runs, run)
		}
	}
	return runs
}

// program is the file a command's words run, or "".
func program(words []string) string {
	var s scan
	for _, word := range words {
		if run, ok := s.next(strings.Trim(word, `'"`)); ok {
			return run
		}
	}
	return ""
}

// scan reads a command word by word: runner is set past node or another
// runner, and value while a runner flag waits for its value.
type scan struct{ runner, value bool }

// next returns word when it is the program.
func (s *scan) next(word string) (string, bool) {
	switch {
	case s.skip(word):
	case !s.runner && slices.Contains(runners, word):
		s.runner = true
	default:
		return word, true
	}
	return "", false
}

// skip reports a word before the program, such as a flag or its value.
func (s *scan) skip(word string) bool {
	skip := s.value || skipped(word, s.runner)
	s.value = skip && s.runner && slices.Contains(valueFlags, word)
	return skip
}

// skipped reports a word before the program: a launcher, a variable
// assignment, a flag, or a runner's run subcommand.
func skipped(word string, runner bool) bool {
	if runner {
		return strings.HasPrefix(word, "-") || word == "run"
	}
	return slices.Contains(launchers, word) || strings.Contains(word, "=") && !strings.HasPrefix(word, "-")
}
