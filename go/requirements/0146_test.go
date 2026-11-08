package requirements

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func Test0146_1(t *testing.T) {
	t.Parallel()
	out, code := slowCheck(t, "exec /bin/sleep 30", fastTotal(1000))
	sees(t, out, code, 1, "The fast checks ran past their 1s total. Slowest: slow 1")
}

// perfRow matches one row of `fitness perf`: check, stage, time and result.
var perfRow = regexp.MustCompile(`^(\S+)\s+(fast|tests)\s+([\d.]+)(m?s)\s+(passed|failed|timed out)$`)

func Test0146_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", localCheck("pause", passing))
	write(t, repo, map[string]string{"checks/fitness-check-pause": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"pause\"}'; exit 0; }\nsleep 2\necho '" + passing + "'\n"})
	record := filepath.Join(t.TempDir(), "fitness.json")
	out, code := fitness(t, repo, nil, "--policy=external", "--checks=pause,markdown-links", "--json="+record)
	sees(t, out, code, 0)
	out, code = fitness(t, repo, nil, "perf", record)
	sees(t, out, code, 0, "CHECK  STAGE  TIME", "The run took")
	var names []string
	for _, line := range strings.Split(out, "\n") {
		if m := perfRow.FindStringSubmatch(line); m != nil {
			names = append(names, m[1])
		}
	}
	if strings.Join(names, " ") != "pause markdown-links" {
		t.Errorf("fitness perf must list each check slowest first, got %v:\n%s", names, out)
	}
}

func Test0146_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", localCheck("pause", passing))
	write(t, repo, map[string]string{"checks/fitness-check-pause": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"pause\"}'; exit 0; }\nsleep 2\necho '" + passing + "'\n"})
	out, code := fitness(t, repo, nil, "--policy=external", "--checks=pause,markdown-links")
	sees(t, out, code, 0, "Slowest: pause 2")
}
