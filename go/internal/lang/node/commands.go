package node

import (
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/runtimes"
)

// runners run the file named after them, past flags and a run
// subcommand, as in "node --watch scripts/seed.mjs".
var runners = strings.Fields("node tsx ts-node vite-node bun deno zx sh bash zsh python python3")

// valueFlags are runner flags whose value is the next word, as in
// "node -r dotenv/config server.js".
var valueFlags = strings.Fields("-r --require --import --loader --experimental-loader --env-file --watch-path -C --conditions --tsconfig")

// launchers start the command after them, as in "npx tsx scripts/a.ts".
var launchers = strings.Fields("npx exec env pnpm yarn dlx")

// scriptRuns lists the files each script of one package.json runs, Start
// for the start script and Dev for the others, and its published entries.
// The containers check reports a package.json it cannot parse.
func scriptRuns(root, file string) runtimes.Runs {
	var runs runtimes.Runs
	pkg, _, err := readPackage(root, file)
	if err != nil {
		return runs
	}
	for name, command := range pkg.Scripts {
		for _, run := range commandRuns(command) {
			if name == "start" {
				runs.Start = append(runs.Start, runtimes.Beside(file, run))
			} else {
				runs.Dev = append(runs.Dev, runtimes.Beside(file, run))
			}
		}
	}
	runs.Published = pkg.entries(file)
	return runs
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
