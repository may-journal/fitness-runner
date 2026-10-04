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

func TestReleaseOnlyTagsTestedMainCommit(t *testing.T) {
	text := workflowSource(t, ".github/workflows/ci.yml")
	job := text[strings.Index(text, "  release-tag:"):]
	requireWorkflowParts(t, job, "needs: fitness", "github.event_name == 'push'", "github.event_name == 'workflow_dispatch'", "github.ref == 'refs/heads/main'", "ref: ${{ github.sha }}", "fetch-depth: 0", "token: ${{ steps.app-token.outputs.token }}", "ensure-tag --root ..")
	requireWorkflowParts(t, job, "if: ${{ always() }}", "::error::", "GITHUB_STEP_SUMMARY", "actions/upload-artifact@v4")
	if strings.Index(job, "actions/create-github-app-token@v2") > strings.Index(job, "actions/checkout@v4") {
		t.Fatal("checkout cannot use a token created later")
	}
}

func TestPromotionWaitsForPublicDownloads(t *testing.T) {
	text := workflowSource(t, ".github/workflows/release.yml")
	requireWorkflowParts(t, text, "group: fitness-release", "cancel-in-progress: false", "queue: max", "needs: [release, verify]", "version: ${{ needs.verify.outputs.version }}")
	for _, platform := range platforms {
		requireWorkflowParts(t, text, "platform: "+platform)
	}
	job := text[strings.Index(text, "  promote-pins:"):]
	requireWorkflowParts(t, job, "needs: verify-download", "ref: main", "fetch-depth: 0", "GH_TOKEN: ${{ steps.app-token.outputs.token }}", "out/fitness-release-linux-amd64 promote\n          out/fitness-release-linux-amd64 pin-release")
	requireWorkflowParts(t, job, "if: ${{ always() }}", "::error::", "GITHUB_STEP_SUMMARY", "actions/upload-artifact@v4")
}

func installerSelection(t *testing.T, version string) (string, error) {
	t.Helper()
	text := workflowSource(t, "action.yml")
	start := strings.Index(text, "        if [ -n \"$REQUESTED_VERSION\" ]")
	end := strings.Index(text, "        curl -fsSL")
	script := text[start:end] + "\nprintf '%s\\n%s\\n' \"$RELEASE_URL\" \"$ASSET\"\n"
	command := exec.Command("bash", "-eu", "-c", script)
	command.Env = append(os.Environ(), "REQUESTED_VERSION="+version, "PLATFORM=linux-amd64", "RELEASE_URL=https://github.com/may-journal/fitness-runner/releases/download/go/v0.20261004.1140")
	data, err := command.CombinedOutput()
	return string(data), err
}

func TestActionSelectsExplicitInstaller(t *testing.T) {
	for _, requested := range []string{"", "latest", "0.20261005.1", "v0.20261005.1", "go/v0.20261005.1"} {
		assertInstallerSelection(t, requested)
	}
	text := workflowSource(t, "action.yml")
	requireWorkflowParts(t, text, "REQUESTED_VERSION: ${{ inputs.version }}", "FITNESS_EXECUTABLE: ${{ steps.download.outputs.executable }}", "FITNESS_VERSION: ${{ inputs.version }}", "shasum -a 256 -c -")
}

func assertInstallerSelection(t *testing.T, requested string) {
	t.Helper()
	version := strings.TrimPrefix(strings.TrimPrefix(requested, "go/"), "v")
	if requested == "" || requested == "latest" {
		version = "0.20261004.1140"
	}
	output, err := installerSelection(t, requested)
	want := "https://github.com/may-journal/fitness-runner/releases/download/go/v" + version + "\nfitness-install-" + version + "-linux-amd64\n"
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
