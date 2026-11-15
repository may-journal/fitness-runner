package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// inner0162 is a Go test that runs the fitness named by REQ_FITNESS, as a
// repo's own requirement tests do. It fails unless that run used a check.
const inner0162 = `package m

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

const fresh = FRESH

func TestInnerFitness(t *testing.T) {
	out, err := exec.Command(os.Getenv("REQ_FITNESS"), "--policy=external", "--checks=markdown-links").CombinedOutput()
	if err != nil || strings.Contains(string(out), "no result from the check") || !strings.Contains(string(out), "All 1 checks passed") {
		t.Fatalf("fitness inside the test found no installed check:\n%s", out)
	}
}
`

// checksSource0162 is the fitness-checks commit whose go-test the test builds.
// That repo tags no module versions, so a commit keeps the build the same.
const checksSource0162 = "d09ff3049e7c8cd44cd8c1d2982a4c45535cb94c"

// goTestOnThisEngine builds fitness-check-go-test from fitness-checks on this
// checkout's engine code. It returns a checks folder that holds that build
// in place of the released one.
func goTestOnThisEngine(t *testing.T) string {
	t.Helper()
	here, err := os.Getwd()
	mustDo(t, err)
	module := filepath.Dir(here)
	build := t.TempDir()
	write(t, build, map[string]string{
		"go.mod": "module example.com/build\n\ngo 1.26\n\n" +
			"replace github.com/may-journal/fitness-runner/go => " + module + "\n",
		"tools.go": "//go:build tools\n\npackage build\n\nimport _ \"github.com/may-journal/fitness-checks/cmd/fitness-check-go-test\"\n",
	})
	checks := t.TempDir()
	for _, args := range [][]string{
		{"get", "github.com/may-journal/fitness-checks@" + checksSource0162},
		{"build", "-o", filepath.Join(checks, "fitness-check-go-test"), "github.com/may-journal/fitness-checks/cmd/fitness-check-go-test"},
	} {
		cmd := exec.Command("go", args...)
		cmd.Dir, cmd.Env = build, append(os.Environ(), "GOFLAGS=-mod=mod")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	linkReleased0162(t, checks)
	return checks
}

// linkReleased0162 links every released check but go-test into checks.
func linkReleased0162(t *testing.T, checks string) {
	t.Helper()
	released, err := os.ReadDir(os.Getenv("FITNESS_CHECKS_DIR"))
	if err != nil {
		t.Fatalf("the suite names the released checks in FITNESS_CHECKS_DIR, as tests/requirements.sh sets it: %v", err)
	}
	for _, e := range released {
		if e.Name() != "fitness-check-go-test" {
			mustDo(t, os.Symlink(filepath.Join(os.Getenv("FITNESS_CHECKS_DIR"), e.Name()), filepath.Join(checks, e.Name())))
		}
	}
}

func Test0162_5(t *testing.T) {
	t.Parallel()
	// The runner stands alone, so only FITNESS_CHECKS_DIR leads it to the checks.
	runner := filepath.Join(bundle0162(t), "fitness")
	checks := goTestOnThisEngine(t)
	files := map[string]string{
		"go.mod":    "module example.com/m\n\ngo 1.22\n",
		"m_test.go": strings.Replace(inner0162, "FRESH", strconv.FormatInt(time.Now().UnixNano(), 10), 1),
	}
	out, code := goTest(t, files, "FITNESS_CHECKS_DIR="+checks, "REQ_FITNESS="+runner)
	sees(t, out, code, 0, "All 1 checks passed")
}
