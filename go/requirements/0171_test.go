package requirements

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// scriptRef0171 finds a script a step names, under scripts or beside an action.
var scriptRef0171 = regexp.MustCompile(`(?:run: ["']?(?:\.\./)?(scripts/[\w-]+\.sh)|script: ([\w-]+\.sh)|action_path \}\}/((?:\.\./)*[\w./-]+\.sh))`)

// yamlFiles0171 lists the workflow and action files of the repo at root.
func yamlFiles0171(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	for _, pattern := range []string{".github/workflows/*.yml", ".github/actions/*/action.yml", "action.yml"} {
		found, err := filepath.Glob(filepath.Join(root, pattern))
		mustDo(t, err)
		files = append(files, found...)
	}
	return files
}

// scriptPath0171 is the file a reference names, from the folder of its file.
func scriptPath0171(root, file string, m []string) string {
	switch {
	case m[1] != "":
		return filepath.Join(root, m[1])
	case m[2] != "":
		return filepath.Join(root, "scripts", m[2])
	}
	return filepath.Join(filepath.Dir(file), m[3])
}

// executable0171 reports whether path is a file someone can run.
func executable0171(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode()&0o111 != 0
}

// missing0171 lists the scripts file names that are missing or not executable.
func missing0171(t *testing.T, root, file string) (named int, bad []string) {
	t.Helper()
	text, err := os.ReadFile(file)
	mustDo(t, err)
	for _, m := range scriptRef0171.FindAllStringSubmatch(string(text), -1) {
		named++
		if !executable0171(scriptPath0171(root, file, m)) {
			bad = append(bad, m[0])
		}
	}
	return named, bad
}

func Test0171_1(t *testing.T) {
	t.Parallel()
	root, named := "../..", 0
	for _, file := range yamlFiles0171(t, root) {
		n, bad := missing0171(t, root, file)
		named += n
		for _, ref := range bad {
			t.Errorf("%s names %s, which is missing or not executable", file, ref)
		}
	}
	if named < 20 {
		t.Errorf("want the test to read the repo's script references, found %d", named)
	}
}

// outputKey0171 finds a key a script writes to the step output.
var outputKey0171 = regexp.MustCompile(`([a-z]+)=`)

// outputKeys0171 lists the step output keys a script writes.
func outputKeys0171(text string) []string {
	var keys []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "GITHUB_OUTPUT") {
			continue
		}
		for _, m := range outputKey0171.FindAllStringSubmatch(strings.ReplaceAll(strings.SplitN(line, "GITHUB_OUTPUT", 2)[0], `\n`, " "), -1) {
			keys = append(keys, m[1])
		}
	}
	return keys
}

func Test0171_2(t *testing.T) {
	t.Parallel()
	action, err := os.ReadFile("../../.github/actions/script/action.yml")
	mustDo(t, err)
	for _, file := range yamlFiles0171(t, "../..") {
		text, err := os.ReadFile(file)
		mustDo(t, err)
		for _, m := range scriptInput0171.FindAllStringSubmatch(string(text), -1) {
			script, err := os.ReadFile(filepath.Join("..", "..", "scripts", m[1]))
			mustDo(t, err)
			for _, key := range outputKeys0171(string(script)) {
				if !strings.Contains(string(action), "  "+key+":\n    description") {
					t.Errorf("%s writes the output %s, which the script action does not declare", m[1], key)
				}
			}
		}
	}
}
