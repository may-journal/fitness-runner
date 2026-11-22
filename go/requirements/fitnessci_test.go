package requirements

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
)

// fitnessCIDir holds the fitness-ci this test binary builds once; TestMain
// removes it.
var fitnessCIDir = sync.OnceValues(func() (string, error) {
	return os.MkdirTemp("", "fitness-ci-")
})

// builtFitnessCI builds fitness-ci from the checkout once, as a workflow job
// does, and returns its path.
var builtFitnessCI = sync.OnceValues(func() (string, error) {
	dir, err := fitnessCIDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "fitness-ci")
	build := exec.Command("go", "-C", "..", "build", "-o", path, "./cmd/fitness-ci")
	if out, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go build fitness-ci: %v\n%s", err, out)
	}
	return path, nil
})

// fitnessCI returns the fitness-ci built from the checkout.
func fitnessCI(t *testing.T) string {
	t.Helper()
	path, err := builtFitnessCI()
	mustDo(t, err)
	return path
}

// removeFitnessCI removes the fitness-ci the tests built, if any.
func removeFitnessCI() {
	if dir, err := fitnessCIDir(); err == nil {
		_ = os.RemoveAll(dir)
	}
}

var (
	// commandInput finds the subcommand a step hands to the fitness-ci action,
	// reading on from the action's name to the end of the step.
	commandInput = regexp.MustCompile(`actions/fitness-ci\n(?: +[^-\s][^\n]*\n)*? +command: ([a-z-]+(?: [^\n]+)?)\n`)
	// commandRun finds the subcommand a step runs from the job's bin folder.
	commandRun = regexp.MustCompile(`run: (?:bin/)?fitness-ci ([a-z-]+(?: [a-z-]+)*)\n`)
)

// quoted is s quoted for bash.
func quoted(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// stepScript is the bash a workflow step runs: its fitness-ci subcommand,
// called on the program built from the checkout, or its run block.
func stepScript(t *testing.T, step string) (string, bool) {
	t.Helper()
	runAt := strings.Index(step, "run: |\n")
	if sub, at := stepCommand(step); at >= 0 && (runAt < 0 || at < runAt) {
		return quoted(fitnessCI(t)) + " " + sub + "\n", true
	}
	if runAt < 0 {
		return "", false
	}
	return step[runAt+len("run: |\n"):], true
}

// stepCommand is the first fitness-ci subcommand step names and where, or
// -1 for none.
func stepCommand(step string) (string, int) {
	for _, re := range []*regexp.Regexp{commandInput, commandRun} {
		if m := re.FindStringSubmatchIndex(step); m != nil {
			return step[m[2]:m[3]], m[0]
		}
	}
	return "", -1
}

// exportsChecksDir reports whether the job installs every pinned check, by
// running the subcommand it names as the step does.
func exportsChecksDir(t *testing.T, job string) bool {
	t.Helper()
	if !strings.Contains(job, "run: bin/fitness-ci export-checks-dir\n") {
		return false
	}
	run := runFitnessCI(t, ciStep{
		file: "ci.yml", command: "export-checks-dir",
		files: map[string]string{"githooks/checks-dir": "#!/bin/sh\necho /cache/checks\n"},
	})
	return run.code == 0 && strings.Contains(run.env, "FITNESS_CHECKS_DIR=/cache/checks\n")
}

// yamlFiles lists the workflow and action files of the repo at root.
func yamlFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	for _, pattern := range []string{".github/workflows/*.yml", ".github/actions/*/action.yml", "action.yml"} {
		found, err := filepath.Glob(filepath.Join(root, pattern))
		mustDo(t, err)
		files = append(files, found...)
	}
	return files
}

// scriptRef finds a step that runs a shell script, which no step may.
var scriptRef = regexp.MustCompile(`run: [^\n]*\.sh\b[^\n]*`)

// subcommandOutputs is every fitness-ci subcommand and the step outputs it
// writes, as its help lists them.
func subcommandOutputs(t *testing.T) map[string][]string {
	t.Helper()
	out, err := exec.Command(fitnessCI(t), "help").Output()
	mustDo(t, err)
	subs := map[string][]string{}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
			continue
		}
		name, outputs, _ := strings.Cut(strings.TrimSpace(line), " (outputs: ")
		subs[name] = nil
		if outputs != "" {
			subs[name] = strings.Split(strings.TrimSuffix(outputs, ")"), ", ")
		}
	}
	return subs
}

// badReferences counts file's subcommand references, listing each unknown
// subcommand and each step that runs a shell script.
func badReferences(t *testing.T, file string, subs map[string][]string) (named int, bad []string) {
	t.Helper()
	text, err := os.ReadFile(file)
	mustDo(t, err)
	refs := subcommandRefs(string(text))
	for _, m := range refs {
		if _, ok := subs[strings.Fields(m[1])[0]]; !ok {
			bad = append(bad, strings.TrimSpace(m[0]))
		}
	}
	return len(refs), append(bad, scriptRef.FindAllString(string(text), -1)...)
}

// subcommandRefs is every step of text that names a fitness-ci subcommand,
// with the subcommand as the first group.
func subcommandRefs(text string) [][]string {
	return append(commandInput.FindAllStringSubmatch(text, -1), commandRun.FindAllStringSubmatch(text, -1)...)
}

// undeclaredOutputs counts the outputs of each subcommand file runs through
// the fitness-ci action. It fails the test for each one action lacks.
func undeclaredOutputs(t *testing.T, file, action string, subs map[string][]string) int {
	t.Helper()
	text, err := os.ReadFile(file)
	mustDo(t, err)
	checked := 0
	for _, m := range commandInput.FindAllStringSubmatch(string(text), -1) {
		for _, key := range subs[strings.Fields(m[1])[0]] {
			checked++
			if !strings.Contains(action, "  "+key+":\n    description") {
				t.Errorf("fitness-ci %s writes the output %s, which the fitness-ci action does not declare", m[1], key)
			}
		}
	}
	return checked
}

// ciStep is one step that runs a fitness-ci subcommand, and the job around it.
type ciStep struct {
	// file is the workflow under .github/workflows, or the shared action
	// under .github/actions, that calls the subcommand.
	file, command string
	// env is the step's environment.
	env []string
	// files are written into the job's checkout first.
	files map[string]string
	// saved are the stand-ins' answers, by program.
	saved map[string][]response
	// dir is the checkout, a new one when empty.
	dir string
}

// ciStepLine returns how file calls command. That is the job's own run line,
// or the fitness-ci action's run line and the input it passes.
func ciStepLine(t *testing.T, file, command string) (line string, env []string) {
	t.Helper()
	raw, err := os.ReadFile(ciFile(file))
	mustDo(t, err)
	if callsCommand(commandRun, string(raw), command) {
		return "bin/fitness-ci " + command, nil
	}
	if callsCommand(commandInput, string(raw), command) {
		return actionRunLine(t), []string{"COMMAND=" + command}
	}
	t.Fatalf("%s runs no fitness-ci %s", ciFile(file), command)
	return "", nil
}

// ciFile is the workflow file, or the shared action's file, file names.
func ciFile(file string) string {
	if file == "action.yml" {
		return filepath.Join("..", "..", file)
	}
	if strings.HasSuffix(file, ".yml") {
		return filepath.Join("..", "..", ".github", "workflows", file)
	}
	return filepath.Join("..", "..", ".github", "actions", file, "action.yml")
}

// callsCommand reports whether some match of re in text names command.
func callsCommand(re *regexp.Regexp, text, command string) bool {
	return slices.ContainsFunc(re.FindAllStringSubmatch(text, -1), func(m []string) bool { return m[1] == command })
}

// actionRunLine is the line the fitness-ci action runs its subcommand with.
func actionRunLine(t *testing.T) string {
	t.Helper()
	action, err := os.ReadFile(filepath.Join("..", "..", ".github", "actions", "fitness-ci", "action.yml"))
	mustDo(t, err)
	_, step, _ := strings.Cut(string(action), "- name: Run the subcommand\n")
	_, run, _ := strings.Cut(step, "\n      run: ")
	run, _, _ = strings.Cut(run, "\n")
	return run
}

// ciRun is what one step run left. It holds the output, exit code, files for
// later steps, RUNNER_TEMP and stand-in calls.
type ciRun struct {
	out                               string
	code                              int
	outputs, env, path, summary, temp string
	rp                                replays
}

// runFitnessCI runs the step as Actions does, in bash in the job's checkout
// with fitness-ci built into bin/.
func runFitnessCI(t *testing.T, s ciStep) ciRun {
	t.Helper()
	line, actionEnv := ciStepLine(t, s.file, s.command)
	dir := s.dir
	if dir == "" {
		dir = t.TempDir()
	}
	write(t, dir, s.files)
	mustDo(t, os.MkdirAll(filepath.Join(dir, "bin"), 0o755))
	if _, err := os.Lstat(filepath.Join(dir, "bin", "fitness-ci")); err != nil {
		mustDo(t, os.Symlink(fitnessCI(t), filepath.Join(dir, "bin", "fitness-ci")))
	}
	job := t.TempDir()
	files := map[string]string{}
	for _, name := range []string{"GITHUB_OUTPUT", "GITHUB_ENV", "GITHUB_PATH", "GITHUB_STEP_SUMMARY"} {
		files[name] = filepath.Join(job, strings.ToLower(name))
	}
	saved := s.saved
	if saved == nil {
		saved = map[string][]response{}
	}
	rp := standIn(t, saved)
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", line)
		cmd.Dir = dir
		cmd.Env = append(append(userEnv(), rp.env...), "RUNNER_TEMP="+job, "GITHUB_REPOSITORY="+repoPR)
		for name, path := range files {
			cmd.Env = append(cmd.Env, name+"="+path)
		}
		// The job's bin/ comes first on PATH, as a job finds fitness-ci.
		cmd.Env = append(cmd.Env, "PATH="+filepath.Join(dir, "bin")+string(os.PathListSeparator)+strings.TrimPrefix(rp.env[0], "PATH="))
		cmd.Env = append(append(cmd.Env, actionEnv...), s.env...)
		return cmd
	})
	read := func(name string) string {
		raw, _ := os.ReadFile(files[name])
		return string(raw)
	}
	return ciRun{out, code, read("GITHUB_OUTPUT"), read("GITHUB_ENV"), read("GITHUB_PATH"), read("GITHUB_STEP_SUMMARY"), job, rp}
}

// tool is a program file that prints its name and arguments, left without
// run bits, as an artifact download leaves a program.
func tool(name string) string {
	return "#!/bin/sh\necho \"" + name + " $*\"\n"
}

// writeTools writes each program in files without run bits.
func writeTools(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	write(t, dir, files)
	for name := range files {
		mustDo(t, os.Chmod(filepath.Join(dir, name), 0o644))
	}
}

// taggedRepo is a repo with one commit tagged tag, pushed to a bare origin,
// and the origin's path.
func taggedRepo(t *testing.T, tag string) (repo, origin string) {
	t.Helper()
	repo, origin = t.TempDir(), t.TempDir()
	git(t, origin, "init", "-q", "--bare")
	commit(t, repo, map[string]string{"README.md": "# app\n"}, "docs(readme): add readme")
	git(t, repo, "tag", tag)
	git(t, repo, "remote", "add", "origin", origin)
	return repo, origin
}

// tagCommit is the commit tag names in repo, or "" when there is none.
func tagCommit(repo, tag string) string {
	out, err := exec.Command("git", "-C", repo, "rev-list", "-n", "1", tag).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// hasAll fails the test for each want text lacks.
func hasAll(t *testing.T, what, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("%s lacks %q:\n%s", what, want, text)
		}
	}
}
