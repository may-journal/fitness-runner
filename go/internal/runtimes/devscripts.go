package runtimes

import (
	"path"
	"slices"
	"strings"
)

// runners run the file named after them, past flags and a run
// subcommand, as in "node --env-file=.env scripts/seed.mjs".
var runners = strings.Fields("node tsx ts-node vite-node bun deno zx sh bash zsh python python3")

// launchers start the command after them, as in "npx tsx scripts/a.ts".
var launchers = strings.Fields("npx exec env pnpm yarn dlx")

// DevScripts lists the tracked files that only package.json scripts run,
// such as scripts/analyze.mjs in "node scripts/analyze.mjs". A file the
// start script runs, or one the package publishes, is product code.
func DevScripts(root string, files []string) map[string]bool {
	tracked := map[string]bool{}
	for _, file := range files {
		tracked[file] = true
	}
	dev, product := map[string]bool{}, map[string]bool{}
	for _, file := range files {
		if path.Base(file) == "package.json" {
			scriptRuns(root, file, tracked, dev, product)
		}
	}
	for file := range product {
		delete(dev, file)
	}
	return dev
}

// scriptRuns adds the files each script of one package.json runs to dev,
// or to product for the start script and the published entries. The
// containers check reports a package.json it cannot parse.
func scriptRuns(root, file string, tracked, dev, product map[string]bool) {
	pkg, _, err := readPackage(root, file)
	if err != nil {
		return
	}
	for name, command := range pkg.Scripts {
		into := dev
		if name == "start" {
			into = product
		}
		mark(file, command, tracked, into)
	}
	for _, entry := range pkg.entries(file) {
		product[entry] = true
	}
}

// mark adds each tracked file a script command runs to into.
func mark(file, command string, tracked, into map[string]bool) {
	for _, run := range commandRuns(command) {
		if p := beside(file, run); tracked[p] {
			into[p] = true
		}
	}
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
	runner := false
	for _, word := range words {
		word = strings.Trim(word, `'"`)
		switch {
		case skipped(word, runner):
			continue
		case !runner && slices.Contains(runners, word):
			runner = true
			continue
		}
		return word
	}
	return ""
}

// skipped reports a word before the program: a launcher, a variable
// assignment, a flag, or a runner's run subcommand.
func skipped(word string, runner bool) bool {
	if runner {
		return strings.HasPrefix(word, "-") || word == "run"
	}
	return slices.Contains(launchers, word) || strings.Contains(word, "=") && !strings.HasPrefix(word, "-")
}
