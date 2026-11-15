package requirements

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// jobLine0167 starts a job under a workflow's jobs key.
var jobLine0167 = regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`)

// jobs0167 splits a workflow's text into its jobs' texts by name.
func jobs0167(text string) map[string]string {
	jobs := map[string]string{}
	_, body, _ := strings.Cut(text, "\njobs:\n")
	name := ""
	for _, line := range strings.Split(body, "\n") {
		if m := jobLine0167.FindStringSubmatch(line); m != nil {
			name = m[1]
		}
		jobs[name] += line + "\n"
	}
	return jobs
}

// checksOutFirst reports whether a job's text checks out the repo before it
// runs the fitness-command action, or runs no fitness command at all.
func checksOutFirst(job string) bool {
	command := strings.Index(job, "actions/fitness-command")
	checkout := strings.Index(job, "actions/checkout@")
	return command < 0 || (checkout >= 0 && checkout < command)
}

func Test0167_1(t *testing.T) {
	t.Parallel()
	paths, err := filepath.Glob(filepath.Join("..", "..", ".github", "workflows", "*-reusable.yml"))
	mustDo(t, err)
	commands := 0
	for _, path := range paths {
		text, err := os.ReadFile(path)
		mustDo(t, err)
		for name, job := range jobs0167(string(text)) {
			commands += strings.Count(job, "actions/fitness-command")
			if !checksOutFirst(job) {
				t.Errorf("%s job %s runs a fitness command without a checkout first, so it installs no pinned check", filepath.Base(path), name)
			}
		}
	}
	if commands == 0 {
		t.Fatal("no reusable workflow runs the fitness-command action, so the test proves nothing")
	}
}
