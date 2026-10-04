package report

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkflowReportingCoverage(t *testing.T) {
	paths, err := filepath.Glob("../../../.github/workflows/*.yml")
	if err != nil || len(paths) == 0 {
		t.Fatalf("find workflows: %v", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) { checkWorkflowReporting(t, path) })
	}
}

func checkWorkflowReporting(t *testing.T, path string) {
	t.Helper()
	body := readWorkflow(t, path)
	jobs := strings.Count(body, "    steps:\n")
	if got := strings.Count(body, "      - name: Report job outcome\n        if: ${{ always() }}"); got != jobs {
		t.Fatalf("%d executable jobs but %d unconditional final reporters", jobs, got)
	}
	if strings.Contains(body, "id: report-step-") && !strings.Contains(path, "auto-merge") {
		checkReportArtifacts(t, body, jobs)
	}
}

func checkReportArtifacts(t *testing.T, body string, jobs int) {
	t.Helper()
	if strings.Count(body, "env.FITNESS_REPORT_ID") != jobs {
		t.Fatal("reusable workflow calls must have distinct report artifacts")
	}
	if strings.Count(body, "name: Retain complete Fitness reports") != jobs {
		t.Fatal("reporting jobs must retain full overflow reports")
	}
	if strings.Count(body, "steps.fitness-reports.outcome != 'failure'") != jobs {
		t.Fatal("artifact failures must not be hidden by a previous diagnostic")
	}
}

func readWorkflow(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestBootstrapReportingWithoutTools(t *testing.T) {
	paths, _ := filepath.Glob("../../../.github/workflows/*.yml")
	paths = append(paths, "../../../action.yml")
	for _, path := range paths {
		body := readWorkflow(t, path)
		if !strings.Contains(body, "REPORT_STATUS:") {
			continue
		}
		t.Run(filepath.Base(path), func(t *testing.T) { checkBootstrapStates(t, body) })
	}
}

func checkBootstrapStates(t *testing.T, body string) {
	t.Helper()
	for _, state := range []struct{ status, annotated, artifact string }{
		{"success", "false", ""}, {"failure", "false", ""}, {"failure", "true", ""}, {"cancelled", "false", ""},
		{"success", "false", "https://github.com/example/repo/actions/runs/123/artifacts/456"},
	} {
		t.Run(state.status+state.annotated, func(t *testing.T) { runBootstrap(t, body, state.status, state.annotated, state.artifact) })
	}
}

func runBootstrap(t *testing.T, body, status, annotated, artifact string) {
	t.Helper()
	dir := t.TempDir()
	script := body[strings.LastIndex(body, "run: |\n")+len("run: |\n"):]
	script = strings.SplitN(script, "\n\n", 2)[0]
	cmd := exec.Command("/bin/bash", "-e", "-c", script)
	cmd.Env = []string{"PATH=/no-tools", "REPORT_ARTIFACT_URL=" + artifact, "REPORT_STATUS=" + status, "REPORT_ALREADY_ANNOTATED=" + annotated,
		"GITHUB_STEP_SUMMARY=" + filepath.Join(dir, "summary"), "GITHUB_ENV=" + filepath.Join(dir, "env"),
		"GITHUB_JOB=test-job", "GITHUB_SERVER_URL=https://github.com", "GITHUB_REPOSITORY=example/repo", "GITHUB_RUN_ID=123"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bootstrap reporter requires no installed tools: %v\n%s", err, out)
	}
	wantError := status == "failure" && annotated != "true"
	if strings.Contains(string(out), "::error::") != wantError {
		t.Fatalf("unexpected annotation for %s/%s: %s", status, annotated, out)
	}
	checkBootstrapSummary(t, filepath.Join(dir, "summary"), status)
	checkArtifactLink(t, body, filepath.Join(dir, "summary"), artifact)
}

func checkBootstrapSummary(t *testing.T, path, status string) {
	t.Helper()
	body := readWorkflow(t, path)
	if !strings.Contains(body, status) || !strings.Contains(body, "https://github.com/example/repo/actions/runs/123") {
		t.Fatalf("summary missing outcome or run link: %s", body)
	}
}

func checkArtifactLink(t *testing.T, workflow, summaryPath, artifact string) {
	t.Helper()
	summary := readWorkflow(t, summaryPath)
	want := artifact != "" && strings.Contains(workflow, "REPORT_ARTIFACT_URL:")
	if strings.Contains(summary, "[Download complete Fitness reports]("+artifact+")") != want {
		t.Fatalf("unexpected artifact link for %q: %s", artifact, summary)
	}
}

func TestActionResetsPriorFailureMarker(t *testing.T) {
	body := readWorkflow(t, "../../../action.yml")
	start := strings.Index(body, "run: |\n") + len("run: |\n")
	bootstrap := strings.SplitN(body[start:], "\n        curl -fsSL", 2)[0]
	path := filepath.Join(t.TempDir(), "env")
	cmd := exec.Command("/bin/bash", "-e", "-c", bootstrap)
	cmd.Env = []string{"GITHUB_ENV=" + path, "FITNESS_FAILURE_REPORTED=true"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap: %v %s", err, out)
	}
	if got := readWorkflow(t, path); got != "FITNESS_FAILURE_REPORTED=false\n" {
		t.Fatalf("stale failure marker: %q", got)
	}
}
