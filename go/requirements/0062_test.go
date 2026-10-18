package requirements

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
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

// releaseToolBundles returns bundle archives for platforms and the GoReleaser
// checksums that list them, as the bundle build leaves them in out/bundles.
func releaseToolBundles(platforms []string) map[string]string {
	files, sums := map[string]string{}, ""
	for _, platform := range platforms {
		name := "fitness-1.0.1-" + platform + ".tar.gz"
		files["out/bundles/"+name] = "bundle for " + platform + "\n"
		sums += fmt.Sprintf("%x  %s\n", sha256.Sum256([]byte(files["out/bundles/"+name])), name)
	}
	files["out/bundles/checksums.txt"] = sums
	return files
}

// releaseToolOutputs returns both GoReleaser passes for platforms: bundle
// archives in out/bundles, and installers, verifiers, and the checksums that
// list installers and archives in out/tools.
func releaseToolOutputs(platforms []string) map[string]string {
	files, sums := releaseToolBundles(platforms), ""
	for _, platform := range platforms {
		installer := "fitness-install-1.0.1-" + platform
		files["out/tools/"+installer] = "installer for " + platform + "\n"
		files["out/tools/fitness-release-"+platform] = "verifier for " + platform + "\n"
		for name, path := range map[string]string{installer: "out/tools/" + installer, "fitness-1.0.1-" + platform + ".tar.gz": "out/bundles/fitness-1.0.1-" + platform + ".tar.gz"} {
			sums += fmt.Sprintf("%x  %s\n", sha256.Sum256([]byte(files[path])), name)
		}
	}
	files["out/tools/checksums.txt"] = sums
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

func Test0062_9(t *testing.T) {
	t.Parallel()
	bundles := releaseToolBundles(releaseToolPlatforms)
	out, code := releaseToolRun(t, releaseToolRepo(t, bundles), nil, "bundle-hashes")
	for _, platform := range releaseToolPlatforms {
		hash := sha256.Sum256([]byte(bundles["out/bundles/fitness-1.0.1-"+platform+".tar.gz"]))
		sees(t, out, code, 0, fmt.Sprintf("%s=%x", platform, hash))
	}
	bundles["out/bundles/fitness-1.0.1-linux-arm64.tar.gz"] = "changed after the build\n"
	out, code = releaseToolRun(t, releaseToolRepo(t, bundles), nil, "bundle-hashes")
	sees(t, out, code, 1, "GoReleaser checksum mismatch for fitness-1.0.1-linux-arm64.tar.gz")
	missing := map[string]string{"assemble": "tools/checksums.txt", "smoke": "release.json", "verify-download": "release.json", "bundle-hashes": "bundles/checksums.txt"}
	repo := releaseToolRepo(t, nil)
	for step, file := range missing {
		out, code := releaseToolRun(t, repo, nil, step)
		sees(t, out, code, 1, filepath.Join("out", filepath.FromSlash(file))+": no such file or directory")
	}
}

func Test0062_10(t *testing.T) {
	t.Parallel()
	host := runtime.GOOS + "-" + runtime.GOARCH
	full := releaseToolOutputs(releaseToolPlatforms)
	hash := fmt.Sprintf("%s=%x", host, sha256.Sum256([]byte(full["out/bundles/fitness-1.0.1-"+host+".tar.gz"])))
	out, code := releaseToolRun(t, releaseToolRepo(t, full), nil, "bundle-hashes", "--host")
	if strings.TrimSpace(out) != hash || code != 0 {
		t.Errorf("bundle-hashes --host printed %q and exited %d, want only %s", out, code, hash)
	}
	files := releaseToolOutputs([]string{host})
	for _, platform := range releaseToolPlatforms {
		files["out/fitness-1.0.0-"+platform+".tar.gz"] = "full build archive\n"
		files["out/fitness-install-1.0.0-"+platform] = "full build installer\n"
		files["out/fitness-release-"+platform] = "full build verifier\n"
	}
	repo := releaseToolRepo(t, files)
	out, code = releaseToolRun(t, repo, nil, "assemble", "--host")
	sees(t, out, code, 0)
	want := []string{"bundles", "checksums.txt", "fitness-1.0.1-" + host + ".tar.gz", "fitness-install-1.0.1-" + host, "fitness-release-" + host, "release.json", "tools"}
	if got := releaseToolEntries(t, filepath.Join(repo, "out")); !slices.Equal(got, want) {
		t.Errorf("assemble --host left %v in out, want %v", got, want)
	}
	out, code = releaseToolRun(t, repo, nil, "tag", "--host")
	sees(t, out, code, 0, "v1.0.1", "fitness-release tag: warning: --host not handled, ignoring it")
}

// releaseToolEntries returns the sorted names in dir.
func releaseToolEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}
