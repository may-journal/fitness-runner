package requirements

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// scriptInput0171 finds the script a step hands to the script action.
var scriptInput0171 = regexp.MustCompile(`script: ([\w-]+\.sh)\n`)

// stepScript0171 is the script a workflow step runs, from its script input or run block.
func stepScript0171(step string) (string, bool) {
	runAt := strings.Index(step, "run: |\n")
	if m := scriptInput0171.FindStringSubmatchIndex(step); m != nil && (runAt < 0 || m[0] < runAt) {
		raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", step[m[2]:m[3]]))
		return string(raw), err == nil
	}
	if runAt < 0 {
		return "", false
	}
	return step[runAt+len("run: |\n"):], true
}
