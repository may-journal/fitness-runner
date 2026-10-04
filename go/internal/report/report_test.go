package report

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnnotationEscaping(t *testing.T) {
	t.Setenv("GITHUB_WORKSPACE", "")
	got := Annotation("check%", "dir/a,b.go:12: bad\r\n::warning::injected", true)
	want := "::error file=dir/a%2Cb.go,line=12::check%25: dir/a,b.go:12: bad%0D%0A::warning::injected"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLocations(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("GITHUB_WORKSPACE", workspace)
	mustWrite(t, filepath.Join(workspace, "Dockerfile"), "FROM scratch")
	mustWrite(t, filepath.Join(workspace, "a file.go"), "package main")
	nested := filepath.Join(workspace, "go")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(workspace, "collision.go"), "package root")
	mustWrite(t, filepath.Join(nested, "collision.go"), "package nested")
	t.Chdir(nested)
	cases := []struct{ message, property string }{
		{"collision.go:5: bad", " file=go/collision.go,line=5"},
		{"Dockerfile:7: bad", " file=Dockerfile,line=7"},
		{"a file.go:2: bad", " file=a file.go,line=2"},
		{"internal/check.go:3: bad", " file=go/internal/check.go,line=3"},
		{filepath.Join(workspace, "a file.go") + ":4: bad", " file=a file.go,line=4"},
		{"../../outside.go:1: bad", ""},
		{"timeout: exceeded", ""},
		{"exit status: 1", ""},
	}
	for _, tt := range cases {
		t.Run(tt.message, func(t *testing.T) {
			got := Annotation("check", tt.message, true)
			if !strings.HasPrefix(got, "::error"+tt.property+"::") {
				t.Fatal(got)
			}
		})
	}
}

func TestLocalDoesNotPublish(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "false")
	summary := filepath.Join(t.TempDir(), "summary")
	marker := filepath.Join(t.TempDir(), "env")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	t.Setenv("GITHUB_ENV", marker)
	Error("fitness", errors.New("broken"))
	Outcome("install", "ready")
	EmitAnnotations("check", []string{"bad"}, true)
	MarkFailure()
	if _, err := os.Stat(summary); !os.IsNotExist(err) {
		t.Fatal("local summary written", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("local marker written", err)
	}
}

func TestSummaryOverflowPreservesCompleteReport(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	root := t.TempDir()
	t.Setenv("RUNNER_TEMP", root)
	path := filepath.Join(root, "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", path)
	first := strings.Repeat("é", summaryLimit/2-10)
	WriteSummary(first)
	WriteSummary("second append overflows the summary\n")
	WriteSummary("third append\n")
	summary := mustRead(t, path)
	if len(summary) > summaryLimit || !strings.Contains(string(summary), "Summary truncated") {
		t.Fatal("missing bounded summary")
	}
	complete := mustRead(t, filepath.Join(root, "fitness-reports", "summary.md"))
	if string(complete) != first+"second append overflows the summary\nthird append\n" {
		t.Fatal("complete report lost content")
	}
}

func TestFailureMarker(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	path := filepath.Join(t.TempDir(), "env")
	t.Setenv("GITHUB_ENV", path)
	MarkFailure()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "FITNESS_FAILURE_REPORTED=true\n" {
		t.Fatalf("marker %q: %v", data, err)
	}
}

func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestErrorRejectsWorkflowInjection(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "summary"))
	t.Setenv("GITHUB_ENV", "")
	out, diagnostic := captureError(t)
	if strings.Contains(strings.ReplaceAll(diagnostic, "\r", "\n"), "\n::") {
		t.Fatal("unescaped console command", diagnostic)
	}
	if strings.Count(out, "\n") != 1 || !strings.Contains(out, "%0A::warning::") {
		t.Fatal("unescaped annotation", out)
	}
	if strings.Contains(out, "file=") {
		t.Fatal("operational error given source location", out)
	}
}

func captureError(t *testing.T) (string, string) {
	t.Helper()
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errReader, errWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalOut, originalErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outWriter, errWriter
	defer func() { os.Stdout, os.Stderr = originalOut, originalErr }()
	Error("install", errors.New("a.go:2: bad\n::warning::injected"))
	outWriter.Close()
	errWriter.Close()
	out, _ := io.ReadAll(outReader)
	diagnostic, _ := io.ReadAll(errReader)
	outReader.Close()
	errReader.Close()
	return string(out), string(diagnostic)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSummaryPrefixReplacesInvalidUTF8(t *testing.T) {
	input := []byte("valid\xff" + strings.Repeat("é", summaryLimit))
	prefix := summaryPrefix(input, 31)
	if string(prefix) != "valid�"+strings.Repeat("é", 11) {
		t.Fatalf("unexpected prefix %q", prefix)
	}
	if input[5] != 0xff {
		t.Fatal("original report changed")
	}
}
