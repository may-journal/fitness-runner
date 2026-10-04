// cspell:ignore goreleaser
package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func workflowSource(t *testing.T, path string) string {
	t.Helper()
	_, source, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(source), "../../..", path))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func requireWorkflowParts(t *testing.T, text string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(text, part) {
			t.Fatalf("missing release gate %q", part)
		}
	}
}

func installerSelection(t *testing.T, version string) (string, error) {
	t.Helper()
	return installerSelectionAt(t, version, "")
}

func installerSelectionAt(t *testing.T, version, actionRef string) (string, error) {
	t.Helper()
	return installerSelectionWithLatest(t, version, actionRef, "https://github.com/may-journal/fitness-runner/releases/tag/go/v0.20261004.1140")
}

func installerSelectionWithLatest(t *testing.T, version, actionRef, redirect string) (string, error) {
	t.Helper()
	text := workflowSource(t, "action.yml")
	start := strings.Index(text, "        if [ \"$REQUESTED_VERSION\" = latest ]")
	end := strings.Index(text, "        curl -fsSL")
	script := text[start:end] + "\nprintf '%s\\n%s\\n' \"$RELEASE_URL\" \"$ASSET\"\n"
	command := exec.Command("bash", "-eu", "-c", script)
	command.Env = append(os.Environ(), "REQUESTED_VERSION="+version, "ACTION_REF="+actionRef, "LATEST_REDIRECT="+redirect, "PATH="+selectionCurl(t)+string(os.PathListSeparator)+os.Getenv("PATH"), "PLATFORM=linux-amd64", "RELEASE_URL=https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1140")
	data, err := command.CombinedOutput()
	return string(data), err
}

func TestActionSelectsExplicitInstaller(t *testing.T) {
	for _, requested := range []string{"", "latest", "0.20261005.1", "v0.20261005.1", "go/v0.20261005.1", "1.0.0", "v1.2.3", "go/v2.0.0", "0.1.0"} {
		assertInstallerSelection(t, requested)
	}
	text := workflowSource(t, "action.yml")
	requireWorkflowParts(t, text, "ACTION_REF: ${{ github.action_ref }}", "REQUESTED_VERSION: ${{ inputs.version }}", "FITNESS_EXECUTABLE: ${{ steps.download.outputs.executable }}", "FITNESS_VERSION: ${{ inputs.version }}", "shasum -a 256 -c -")
}

func assertInstallerSelection(t *testing.T, requested string) {
	t.Helper()
	version := expectedInstallerVersion(requested)
	output, err := installerSelection(t, requested)
	tag := "v" + version
	if strings.HasPrefix(version, "0.2026") {
		tag = "go/" + tag
	}
	want := "https://github.com/may-journal/fitness-runner/releases/download/" + tag + "\nfitness-install-" + version + "-linux-amd64\n"
	if err != nil || output != want {
		t.Fatalf("installer %q: %q, %v", requested, output, err)
	}
}

func TestActionRejectsInvalidVersions(t *testing.T) {
	for _, version := range []string{"1.2.3; exit 0", "$(exit 0)", "01.2.3", "1.2.3\n", "../../other"} {
		output, err := installerSelection(t, version)
		if err == nil || !strings.Contains(output, "::error::Invalid Fitness version") {
			t.Fatalf("invalid version %q accepted: %q %v", version, output, err)
		}
	}
}

func expectedInstallerVersion(requested string) string {
	if requested == "" || requested == "latest" {
		return "0.20261004.1140"
	}
	return strings.TrimPrefix(strings.TrimPrefix(requested, "go/"), "v")
}

func TestExternalReleaseGates(t *testing.T) {
	ci := workflowSource(t, ".github/workflows/ci.yml")
	job := ci[strings.Index(ci, "  release-please:"):]
	requireWorkflowParts(t, job, "needs: fitness", "github.ref == 'refs/heads/main'", "googleapis/release-please-action@", "token: ${{ steps.app-token.outputs.token }}")
	workflow := workflowSource(t, ".github/workflows/release.yml")
	requireWorkflowParts(t, workflow, "tags: ['v*']", "needs: binaries", "steps.publication.outputs.draft == 'true'", "goreleaser/goreleaser-action@", "needs: [verify-download, verify]", "version: ${{ needs.verify.outputs.version }}", "peter-evans/create-pull-request@", "if [ \"$latest\" = \"$RELEASE_TAG\" ]; then")
	for _, platform := range platforms {
		requireWorkflowParts(t, workflow, "platform: "+platform)
	}
}

func TestActionUsesPinnedReleaseInstaller(t *testing.T) {
	cases := []struct{ requested, actionRef, tag, version string }{
		{"", "v1.0.0", "v1.0.0", "1.0.0"},
		{"latest", "v1.1.0", "go/v0.20261004.1140", "0.20261004.1140"},
		{"", "go/v0.20261005.1", "go/v0.20261005.1", "0.20261005.1"},
		{"2.0.0", "v1.0.0", "v2.0.0", "2.0.0"},
		{"", "main", "go/v0.20261004.1140", "0.20261004.1140"},
		{"latest", "0123456789abcdef", "go/v0.20261004.1140", "0.20261004.1140"},
		{"", "v01.0.0", "go/v0.20261004.1140", "0.20261004.1140"},
	}
	for _, test := range cases {
		output, err := installerSelectionAt(t, test.requested, test.actionRef)
		want := "https://github.com/may-journal/fitness-runner/releases/download/" + test.tag + "\nfitness-install-" + test.version + "-linux-amd64\n"
		if err != nil || output != want {
			t.Fatalf("requested %q action %q: %q, %v", test.requested, test.actionRef, output, err)
		}
	}
}

func selectionCurl(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	script := "#!/bin/sh\nprintf '%s' \"$LATEST_REDIRECT\"\n"
	if err := os.WriteFile(filepath.Join(directory, "curl"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestActionLatestResolvesModernInstaller(t *testing.T) {
	for _, ref := range []string{"main", "0123456789abcdef", "v1.0.0"} {
		output, err := installerSelectionWithLatest(t, "latest", ref, "https://github.com/may-journal/fitness-runner/releases/tag/v1.2.0")
		want := "https://github.com/may-journal/fitness-runner/releases/download/v1.2.0\nfitness-install-1.2.0-linux-amd64\n"
		if err != nil || output != want {
			t.Fatalf("latest at %s: %s, %v", ref, output, err)
		}
	}
}

func TestActionLatestRejectsInvalidRedirect(t *testing.T) {
	for _, redirect := range []string{"", "https://example.com/releases/tag/v1.0.0", "https://github.com/may-journal/fitness-runner/releases/tag/v", "https://github.com/may-journal/fitness-runner/releases/tag/"} {
		_, err := installerSelectionWithLatest(t, "latest", "main", redirect)
		if err == nil {
			t.Fatalf("accepted latest redirect %q", redirect)
		}
	}
}
