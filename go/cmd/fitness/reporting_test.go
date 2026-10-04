package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/render"
)

func captureReporting(t *testing.T, run func() int) (int, string, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(root, "summary.md"))
	t.Setenv("GITHUB_ENV", filepath.Join(root, "env"))
	t.Setenv("GITHUB_SERVER_URL", "https://github.example.com")
	t.Setenv("GITHUB_REPOSITORY", "owner/consumer")
	out, err := os.Create(filepath.Join(root, "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	prior := os.Stdout
	os.Stdout = out
	defer func() { os.Stdout = prior; out.Close() }()
	code := run()
	content, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	summary, err := os.ReadFile(os.Getenv("GITHUB_STEP_SUMMARY"))
	if err != nil {
		t.Fatal(err)
	}
	return code, string(content), string(summary)
}

func TestBodyChecksReportGlobalDiagnostics(t *testing.T) {
	for _, kind := range []string{"pr", "plan"} {
		t.Run(kind, func(t *testing.T) {
			var calls []call
			checker := fakeChecker(map[string]string{"cspell": "docs/fake.md:9: bad word"}, nil, &calls)
			subject := &target{Number: 7, Body: "body", Title: "feat(scope): sample"}
			code, output, summary := captureReporting(t, func() int {
				if kind == "pr" {
					return prCheck(ghEvent{name: "pull_request", PullRequest: subject}, &fakeGH{}, checker)
				}
				return planCheck(ghEvent{name: "issues", Issue: subject}, &fakeGH{}, checker)
			})
			if code != 1 {
				t.Fatalf("code = %d", code)
			}
			assertContainsAll(t, "output", output, "::error::", "#7", "bad word")
			assertContainsNone(t, "output", output, "::error file=")
			assertContainsAll(t, "summary", summary, "https://github.example.com/owner/consumer/", "bad word")
		})
	}
}

func TestCloseRemediationReportsSuccess(t *testing.T) {
	gh := &fakeGH{}
	code, output, summary := captureReporting(t, func() int {
		return closeCheck(closedEvent(openChecklist, "completed", "alice"), gh)
	})
	if code != 0 || len(gh.reopened) != 1 {
		t.Fatalf("code %d reopened %v", code, gh.reopened)
	}
	assertContainsNone(t, "output", output, "::error")
	assertContainsAll(t, "summary", summary, "Reopened", "https://github.example.com/owner/consumer/issues/8")
}

func TestEarlyFailuresReportWithoutSourceLocations(t *testing.T) {
	for _, command := range []string{"runner", "pr-check", "plan-check", "close-check"} {
		t.Run(command, func(t *testing.T) {
			t.Setenv("GITHUB_EVENT_PATH", filepath.Join(t.TempDir(), "missing-event"))
			code, output, summary := captureReporting(t, func() int {
				if command == "runner" {
					return run([]string{"nonexistent-reporting-check"})
				}
				return dispatch([]string{command})
			})
			if code != 1 {
				t.Fatalf("code = %d", code)
			}
			assertContainsAll(t, "output", output, "::error::fitness")
			assertContainsNone(t, "output", output, "::error file=")
			if len(summary) == 0 {
				t.Fatal("missing failure summary")
			}
		})
	}
}

func TestFailedCheckWithoutDiagnosticsStillReports(t *testing.T) {
	code, output, summary := captureReporting(t, func() int {
		return finish([]render.Row{{Name: "broken", Ok: false}}, 0, 1, 0, 0)
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	assertContainsAll(t, "output", output, "::error::broken", "without a diagnostic")
	assertContainsAll(t, "summary", summary, "1 failed")
}

func TestSummaryFailureDoesNotClearFailure(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_STEP_SUMMARY", t.TempDir())
	if code := finish([]render.Row{{Name: "broken", Ok: false}}, 0, 1, 0, 0); code != 1 {
		t.Fatalf("code = %d", code)
	}
}

func TestBodySweepCapsAnnotationsAcrossTargets(t *testing.T) {
	var calls []call
	gh := &fakeGH{}
	for i := 1; i <= 15; i++ {
		gh.prs = append(gh.prs, target{Number: i, Body: "body"})
	}
	code, output, summary := captureReporting(t, func() int {
		return prCheck(ghEvent{name: "workflow_dispatch"}, gh, fakeChecker(map[string]string{"cspell": "bad word"}, nil, &calls))
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if count := strings.Count(output, "::error::"); count != 10 {
		t.Fatalf("annotations = %d", count)
	}
	assertContainsAll(t, "output", output, "5 more issue(s)")
	assertContainsAll(t, "summary", summary, "#15")
}
