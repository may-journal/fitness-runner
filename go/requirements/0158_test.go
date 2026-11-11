// cspell:ignore xlarge
package requirements

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// suite0158 is the workflow the org ruleset runs on every pull request, and
// reusable0158 the build it calls.
const (
	suite0158    = "../../.github/workflows/fitness-suite.yml"
	reusable0158 = "../../.github/workflows/ci-reusable.yml"
)

// pick0158 is the tail of the suite's runs-on input that hands the variable
// on when the condition before it holds, and nothing otherwise.
const pick0158 = " && vars.FITNESS_RUNS_ON || ''"

// call0158 matches startsWith or endsWith on the variable with a literal.
var call0158 = regexp.MustCompile(`(startsWith|endsWith)\(vars\.FITNESS_RUNS_ON, '([^']*)'\)`)

// runsOn0158 returns the expression on the file's first runs-on line.
func runsOn0158(t *testing.T, file string) string {
	t.Helper()
	raw, err := os.ReadFile(file)
	mustDo(t, err)
	_, line, ok := strings.Cut(string(raw), "runs-on: ${{ ")
	expr, _, closed := strings.Cut(line, " }}\n")
	if !ok || !closed {
		t.Fatalf("%s has no runs-on expression", file)
	}
	return expr
}

// label0158 is the runner label the suite passes to ci-reusable when the
// repo's FITNESS_RUNS_ON holds value. Bash evaluates the condition as
// Actions does, ignoring case; an empty label keeps ci-reusable's choice.
func label0158(t *testing.T, value string) string {
	t.Helper()
	expr := runsOn0158(t, suite0158)
	cond, ok := strings.CutSuffix(expr, pick0158)
	if !ok {
		t.Fatalf("the suite's runs-on %q does not end with %q", expr, pick0158)
	}
	cond = call0158.ReplaceAllStringFunc(cond, func(call string) string {
		m := call0158.FindStringSubmatch(call)
		if m[1] == "startsWith" {
			return ` "$V" == '` + m[2] + `'*`
		}
		return ` "$V" == *'` + m[2] + `'`
	})
	if strings.Contains(cond, "vars.") {
		t.Fatalf("the suite's runs-on condition %q reads the variable some other way", cond)
	}
	cmd := exec.Command("bash", "-c", "shopt -s nocasematch; [[ "+cond+" ]]")
	cmd.Env = append(os.Environ(), "V="+value)
	if cmd.Run() == nil {
		return value
	}
	return ""
}

func Test0158_1(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"xcode-27", "macos-26", "macos-15-intel"} {
		if got := label0158(t, value); got != value {
			t.Errorf("FITNESS_RUNS_ON=%s runs the suite on %q, want %q", value, got, value)
		}
	}
}

func Test0158_2(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", "fitness", "self-hosted", "ubuntu-24.04-arm", "xcode-27-xlarge", "macos-latest-large", "MACOS-26-LARGE", "my-macos-26"} {
		if got := label0158(t, value); got != "" {
			t.Errorf("FITNESS_RUNS_ON=%q runs the suite on %q, want the usual runner", value, got)
		}
	}
	if expr := runsOn0158(t, reusable0158); !strings.HasPrefix(expr, "inputs.runs-on || github.event.repository.private && ") {
		t.Errorf("ci-reusable does not fall back to its usual choice for an empty label: %q", expr)
	}
}
