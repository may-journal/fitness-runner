package requirements

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
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

func Test0146_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", localCheck("suite", passing))
	write(t, repo, map[string]string{"checks/fitness-check-suite": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"suite\", \"stage\": \"tests\"}'; exit 0; }\nexec /bin/sleep 30\n"})
	out, code := fitness(t, repo, []string{testsTotal(1000)}, "--policy=external", "--checks=suite,markdown-links")
	sees(t, out, code, 1, "Stopped at the 1s tests total", "The test runners ran past their 1s total. Slowest: suite 1")
}

// moduleSleep is how long each module's one test sleeps in Test0146_5.
const moduleSleep = 6 * time.Second

func Test0146_5(t *testing.T) {
	t.Parallel()
	// A fresh constant in each module keeps go test from replaying a cached
	// result, so every module's test sleeps.
	fresh := strconv.FormatInt(time.Now().UnixNano(), 10)
	files := map[string]string{}
	for _, dir := range []string{"a/", "b/", "c/", "d/"} {
		files[dir+"go.mod"] = "module example.com/m\n\ngo 1.22\n"
		files[dir+"m_test.go"] = "package m\n\nimport (\n\t\"testing\"\n\t\"time\"\n)\n\nconst fresh = " + fresh +
			"\n\nfunc TestSlow(t *testing.T) { time.Sleep(" + strconv.FormatInt(int64(moduleSleep), 10) + ") }\n"
	}
	start := time.Now()
	out, code := goTest(t, files)
	took := time.Since(start)
	sees(t, out, code, 0, "All 1 checks passed · 4 files scanned")
	if oneByOne := 4 * moduleSleep; took >= oneByOne {
		t.Errorf("go-test took %s on four modules whose tests sleep %s each; one module at a time takes %s", took, moduleSleep, oneByOne)
	}
}

func Test0146_6(t *testing.T) {
	t.Parallel()
	pause := "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"pause\"}'; exit 0; }\nsleep 5\necho '" + passing + "'\n"
	repo := prBase(t, with(localCheck("pause", passing), map[string]string{"checks/fitness-check-pause": pause}))
	commit(t, repo, noteGuide(t), "docs(changelog): note the guide")
	out, code := fitness(t, repo, append([]string{fastTotal(8000)}, prEnv...), "--policy=external", "--checks=pause")
	sees(t, out, code, 1, "Stage 1 of 2:", "Stage 2 of 2:", "Stopped at the 8s fast total", "The fast checks ran past their 8s total. Slowest: pause 8s")
}
