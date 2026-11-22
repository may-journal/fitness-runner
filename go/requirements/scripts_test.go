package requirements

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// scriptInput finds the script a step hands to the script action.
var scriptInput = regexp.MustCompile(`script: ([\w-]+\.sh)\n`)

// stepScript is the script a workflow step runs, from its script input or run block.
func stepScript(step string) (string, bool) {
	runAt := strings.Index(step, "run: |\n")
	if m := scriptInput.FindStringSubmatchIndex(step); m != nil && (runAt < 0 || m[0] < runAt) {
		raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", step[m[2]:m[3]]))
		return string(raw), err == nil
	}
	if runAt < 0 {
		return "", false
	}
	return step[runAt+len("run: |\n"):], true
}

// exportsChecksDir reports whether the job runs the script that installs every pinned check.
func exportsChecksDir(t *testing.T, job string) bool {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "export-checks-dir.sh"))
	mustDo(t, err)
	return strings.Contains(job, "scripts/export-checks-dir.sh") && strings.Contains(string(script), `checks="$(githooks/checks-dir)"`)
}

// scriptRef finds a script a step names, under scripts or beside an action.
var scriptRef = regexp.MustCompile(`(?:run: ["']?(?:\.\./)?(scripts/[\w-]+\.sh)|script: ([\w-]+\.sh)|action_path \}\}/((?:\.\./)*[\w./-]+\.sh))`)

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

// scriptPath is the file a reference names, from the folder of its file.
func scriptPath(root, file string, m []string) string {
	switch {
	case m[1] != "":
		return filepath.Join(root, m[1])
	case m[2] != "":
		return filepath.Join(root, "scripts", m[2])
	}
	return filepath.Join(filepath.Dir(file), m[3])
}

// executable reports whether path is a file someone can run.
func executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode()&0o111 != 0
}

// missingScripts lists the scripts file names that are missing or not executable.
func missingScripts(t *testing.T, root, file string) (named int, bad []string) {
	t.Helper()
	text, err := os.ReadFile(file)
	mustDo(t, err)
	for _, m := range scriptRef.FindAllStringSubmatch(string(text), -1) {
		named++
		if !executable(scriptPath(root, file, m)) {
			bad = append(bad, m[0])
		}
	}
	return named, bad
}

// outputKey finds a key a script writes to the step output.
var outputKey = regexp.MustCompile(`([a-z]+)=`)

// outputKeys lists the step output keys a script writes.
func outputKeys(text string) []string {
	var keys []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "GITHUB_OUTPUT") {
			continue
		}
		for _, m := range outputKey.FindAllStringSubmatch(strings.ReplaceAll(strings.SplitN(line, "GITHUB_OUTPUT", 2)[0], `\n`, " "), -1) {
			keys = append(keys, m[1])
		}
	}
	return keys
}
