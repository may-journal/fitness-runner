package requirements

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// releaseToolPlatforms are the platforms a release bundles, in build order.
var releaseToolPlatforms = []string{"darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64"}

// releaseTool returns the candidate fitness-release built next to the
// candidate installer, as the release workflow downloads them together.
func releaseTool(t *testing.T) string {
	t.Helper()
	path := filepath.Join(filepath.Dir(installer(t)), "fitness-release-"+runtime.GOOS+"-"+runtime.GOARCH)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("candidate release tool: %v", err)
	}
	return path
}

// releaseToolRepo copies happyRepo with version.txt at 1.0.1 and files over it.
func releaseToolRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	all := map[string]string{"version.txt": "1.0.1\n"}
	for name, body := range files {
		all[name] = body
	}
	return example(t, "happyRepo", all)
}

// releaseToolRun runs `fitness-release <args> --root .` in repo, as a
// maintainer does, and returns what it printed and its exit code. With no
// args it runs the tool bare.
func releaseToolRun(t *testing.T, repo string, env []string, args ...string) (string, int) {
	t.Helper()
	if len(args) > 0 {
		args = append(args, "--root", ".")
	}
	cmd := exec.Command(releaseTool(t), args...)
	cmd.Dir, cmd.Env = repo, append(userEnv(), env...)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		t.Fatalf("running fitness-release: %v", err)
	}
	return string(out), cmd.ProcessState.ExitCode()
}

// releaseToolBundles returns four bundle archives and the GoReleaser
// checksums that list them, as the bundle build leaves them in out/bundles.
func releaseToolBundles() map[string]string {
	files, sums := map[string]string{}, ""
	for _, platform := range releaseToolPlatforms {
		name := "fitness-1.0.1-" + platform + ".tar.gz"
		files["out/bundles/"+name] = "bundle for " + platform + "\n"
		sums += fmt.Sprintf("%x  %s\n", sha256.Sum256([]byte(files["out/bundles/"+name])), name)
	}
	files["out/bundles/checksums.txt"] = sums
	return files
}

// releaseToolActions is a GitHub Actions job's environment with its summary
// and env files in dir.
func releaseToolActions(dir, ref string) []string {
	return []string{"GITHUB_ACTIONS=true", "GITHUB_STEP_SUMMARY=" + filepath.Join(dir, "summary"), "GITHUB_ENV=" + filepath.Join(dir, "env"), "GITHUB_REF_NAME=" + ref}
}

// releaseToolSummary returns the job summary written into dir.
func releaseToolSummary(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "summary"))
	if err != nil {
		t.Fatalf("job summary: %v", err)
	}
	return string(data)
}

func Test0062_1(t *testing.T) {
	t.Parallel()
	repo := releaseToolRepo(t, nil)
	out, code := releaseToolRun(t, repo, nil, "version")
	sees(t, out, code, 0, "1.0.1")
	out, code = releaseToolRun(t, repo, nil, "tag")
	sees(t, out, code, 0, "v1.0.1")
}

func Test0062_2(t *testing.T) {
	t.Parallel()
	out, code := releaseToolRun(t, releaseToolRepo(t, nil), []string{"GITHUB_REF_NAME=v1.0.2"}, "verify-tag")
	sees(t, out, code, 1, "tag v1.0.2 does not match version.txt v1.0.1")
}

func Test0062_3(t *testing.T) {
	t.Parallel()
	bundles := releaseToolBundles()
	out, code := releaseToolRun(t, releaseToolRepo(t, bundles), nil, "bundle-hashes")
	for _, platform := range releaseToolPlatforms {
		hash := sha256.Sum256([]byte(bundles["out/bundles/fitness-1.0.1-"+platform+".tar.gz"]))
		sees(t, out, code, 0, fmt.Sprintf("%s=%x", platform, hash))
	}
}

func Test0062_4(t *testing.T) {
	t.Parallel()
	changed := releaseToolBundles()
	changed["out/bundles/fitness-1.0.1-linux-arm64.tar.gz"] = "changed after the build\n"
	out, code := releaseToolRun(t, releaseToolRepo(t, changed), nil, "bundle-hashes")
	sees(t, out, code, 1, "GoReleaser checksum mismatch for fitness-1.0.1-linux-arm64.tar.gz")
	missing := map[string]string{"assemble": "tools/checksums.txt", "smoke": "release.json", "verify-download": "release.json", "bundle-hashes": "bundles/checksums.txt"}
	repo := releaseToolRepo(t, nil)
	for step, file := range missing {
		out, code := releaseToolRun(t, repo, nil, step)
		sees(t, out, code, 1, filepath.Join("out", filepath.FromSlash(file))+": no such file or directory")
	}
}

func Test0062_5(t *testing.T) {
	t.Parallel()
	repo := releaseToolRepo(t, nil)
	out, code := releaseToolRun(t, repo, nil)
	sees(t, out, code, 1, "use bundle-hashes, assemble, smoke, verify-download, verify-tag")
	out, code = releaseToolRun(t, repo, nil, "publish-everything")
	sees(t, out, code, 1, `unknown release command "publish-everything"`)
	out, code = releaseToolRun(t, repo, nil, "tag", "--invalid")
	sees(t, out, code, 1, "flag provided but not defined: -invalid", "-root string")
	out, code = releaseToolRun(t, releaseToolRepo(t, map[string]string{"version.txt": "one\n"}), nil, "version")
	sees(t, out, code, 1, `invalid version.txt version "one"`)
}

func Test0062_6(t *testing.T) {
	t.Parallel()
	repo, job := releaseToolRepo(t, nil), t.TempDir()
	out, code := releaseToolRun(t, repo, releaseToolActions(job, "v1.0.1"), "verify-tag")
	sees(t, out, code, 0)
	sees(t, releaseToolSummary(t, job), 0, 0, "## ✅ fitness-release verify-tag", "Completed successfully.")
	out, code = releaseToolRun(t, repo, releaseToolActions(job, "v9.9.9"), "verify-tag")
	sees(t, out, code, 1, "::error::fitness-release verify-tag: tag v9.9.9 does not match version.txt v1.0.1")
	sees(t, releaseToolSummary(t, job), 0, 0, "## ❌ fitness-release verify-tag", "tag v9.9.9 does not match")
}

func Test0062_7(t *testing.T) {
	t.Parallel()
	job := t.TempDir()
	env := append(releaseToolActions(job, "v9.9.9"), "GITHUB_ACTIONS=false")
	out, code := releaseToolRun(t, releaseToolRepo(t, nil), env, "verify-tag")
	sees(t, out, code, 1, "tag v9.9.9 does not match version.txt v1.0.1")
	if strings.Contains(out, "::error") {
		t.Errorf("a local run printed a workflow annotation:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(job, "summary")); !os.IsNotExist(err) {
		t.Errorf("a local run wrote a job summary: %v", err)
	}
}

func Test0062_8(t *testing.T) {
	t.Parallel()
	job := t.TempDir()
	mustDo(t, os.Mkdir(filepath.Join(job, "summary"), 0o755))
	out, code := releaseToolRun(t, releaseToolRepo(t, nil), releaseToolActions(job, "v9.9.9"), "verify-tag")
	sees(t, out, code, 1, "Could not write GitHub job summary", "Fitness: fitness-release verify-tag: tag v9.9.9 does not match")
}
