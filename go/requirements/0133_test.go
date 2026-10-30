package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// reportStep is a last step that reports every outcome to the job summary.
const reportStep = `      - name: Report job outcome
        if: ${{ always() }}
        run: printf 'Job %s\n' "${{ job.status }}" >> "$GITHUB_STEP_SUMMARY"
`

// reportingJob is a workflow whose job, on line 4, builds and then
// reports with tail as its last steps.
func reportingJob(tail string) string {
	return "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n      - run: make\n" + tail
}

// reportingCheck runs workflow-reporting on happyRepo with files written
// over it.
func reportingCheck(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "workflow-reporting")
}

func Test0133_1(t *testing.T) {
	t.Parallel()
	out, code := reportingCheck(t, map[string]string{".github/workflows/ci.yml": reportingJob(reportStep)})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "workflow-reporting", "1")
}

func Test0133_2(t *testing.T) {
	t.Parallel()
	out, code := reportingCheck(t, map[string]string{".github/workflows/build.yaml": reportingJob("")})
	sees(t, out, code, 1, ".github/workflows/build.yaml:4: job build has no step that runs with if: always() and writes $GITHUB_STEP_SUMMARY",
		"add a last step with `if: ${{ always() }}` whose run writes the job's status to \"$GITHUB_STEP_SUMMARY\"")
}

func Test0133_3(t *testing.T) {
	t.Parallel()
	onSuccess := strings.Replace(reportStep, "        if: ${{ always() }}\n", "", 1)
	out, code := reportingCheck(t, map[string]string{".github/workflows/ci.yml": reportingJob(onSuccess)})
	sees(t, out, code, 1, ".github/workflows/ci.yml:4: job build writes $GITHUB_STEP_SUMMARY only when earlier steps pass, at line 9",
		"add `if: ${{ always() }}` to the step at line 9")
}

func Test0133_4(t *testing.T) {
	t.Parallel()
	out, code := reportingCheck(t, map[string]string{".github/workflows/ci.yml": "name: CI\non: push\njobs:\n" +
		"  fitness:\n    uses: may-journal/fitness-runner/.github/workflows/ci-reusable.yml@v1\n    secrets: inherit\n"})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "workflow-reporting", "1")
}

func Test0133_5(t *testing.T) {
	t.Parallel()
	matrix := "name: CI\non: push\njobs:\n  test:\n    strategy:\n      matrix:\n        os: [ubuntu-latest, macos-latest, windows-latest]\n" +
		"    runs-on: ${{ matrix.os }}\n    steps:\n      - run: make test\n"
	out, code := reportingCheck(t, map[string]string{".github/workflows/ci.yml": matrix})
	sees(t, out, code, 1, ".github/workflows/ci.yml:4: job test has no step")
	if n := strings.Count(out, "job test has no step"); n != 1 {
		t.Errorf("a matrix job must be named once, got %d times:\n%s", n, out)
	}
}

func Test0133_6(t *testing.T) {
	t.Parallel()
	out, code := reportingCheck(t, map[string]string{".github/workflows/ci.yml": "name: CI\non: push\njobs:\n  build:\n\truns-on: ubuntu-latest\n"})
	sees(t, out, code, 1, ".github/workflows/ci.yml:5: this workflow is not valid YAML: a tab indents this line")
}

func Test0133_7(t *testing.T) {
	t.Parallel()
	out, code := reportingCheck(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "workflow-reporting")
}

func Test0133_8(t *testing.T) {
	t.Parallel()
	own, err := filepath.Glob(filepath.Join("..", "..", ".github", "workflows", "*.y*ml"))
	mustDo(t, err)
	files := map[string]string{}
	for _, f := range own {
		body, err := os.ReadFile(f)
		mustDo(t, err)
		files[".github/workflows/"+filepath.Base(f)] = string(body)
	}
	out, code := reportingCheck(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
}
