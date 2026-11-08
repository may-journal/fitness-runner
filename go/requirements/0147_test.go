package requirements

import (
	"os"
	"path/filepath"
	"testing"
)

// timeoutJob is a workflow whose job sets timeout-minutes to minutes, with
// comment on the line above it, line 6.
func timeoutJob(comment, minutes string) string {
	return "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n" + comment +
		"    timeout-minutes: " + minutes + "\n    steps:\n      - run: make\n"
}

// timeoutsCheck runs this repo's workflow-timeouts check, built on this
// checkout's SDK, on happyRepo with workflows written over it.
func timeoutsCheck(t *testing.T, workflows map[string]string) (string, int) {
	t.Helper()
	sdk, err := filepath.Abs("..")
	mustDo(t, err)
	files := map[string]string{"fitness/go.mod": "module example.com/repo/fitness\n\ngo 1.26\n\n" +
		"require github.com/may-journal/fitness-runner/go v0.0.0\n\nreplace github.com/may-journal/fitness-runner/go => " + sdk + "\n"}
	for _, name := range []string{"main.go", "README.md"} {
		body, err := os.ReadFile(filepath.Join(sdk, "..", "fitness", "workflow-timeouts", name))
		mustDo(t, err)
		files["fitness/workflow-timeouts/"+name] = string(body)
	}
	for name, body := range workflows {
		files[name] = body
	}
	return fitness(t, example(t, "happyRepo", files), nil, "check", "workflow-timeouts")
}

func Test0147_1(t *testing.T) {
	t.Parallel()
	out, code := timeoutsCheck(t, map[string]string{".github/workflows/ci.yml": timeoutJob("    # The build is slow.\n", "12")})
	sees(t, out, code, 1, ".github/workflows/ci.yml:7: timeout-minutes 12 cites no observed maximum and date",
		"add a comment above it such as `# Observed max 4m12s on 2026-11-08.`")
}

func Test0147_2(t *testing.T) {
	t.Parallel()
	out, code := timeoutsCheck(t, map[string]string{
		".github/workflows/ci.yml":   timeoutJob("    # Observed max 5m15s on 2026-11-05.\n", "11"),
		".github/workflows/lint.yml": timeoutJob("", "3"),
	})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "workflow-timeouts", "2")
}
