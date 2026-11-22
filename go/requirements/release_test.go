package requirements

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
// archives in out/bundles, and installers, verifiers and checksums in out/tools.
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

// workflowReleasePlease is the reusable Release Please workflow opted-in repos call.
const workflowReleasePlease = "../../.github/workflows/release-please-reusable.yml"

// outputsReleasePlease is what Release Please sets for one root release v1.2.3 at abc123.
const outputsReleasePlease = `{"releases_created":"true","paths_released":"[\".\"]","tag_name":"v1.2.3","major":"1","sha":"abc123"}`

// firstReleaseReleasePlease is what Release Please writes for a first release, which
// has no compare link: a `##` version heading and bold scopes.
const firstReleaseReleasePlease = "> Release the changes merged since the last version.\n\n## Background\n\nRelease Please groups merged changes.\n\n## Changelog\n---\n\n\n" +
	"## 1.0.0 (2026-10-10)\n\n\n### Features\n\n* **library:** one API over UIKit and AppKit\n\n---\nAgent: Release Please."

// sectionReleasePlease returns the workflow text of the step named name, up to the
// next step.
func sectionReleasePlease(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(workflowReleasePlease)
	mustDo(t, err)
	_, rest, ok := strings.Cut(string(raw), "- name: "+name+"\n")
	if !ok {
		t.Fatalf("%s has no step %q", workflowReleasePlease, name)
	}
	section, _, _ := strings.Cut(rest, "      - name: ")
	return section
}

// runReleasePlease runs the named step's script as Actions does for release PR 23 at
// commit abc123, with gh answering from saved responses.
func runReleasePlease(t *testing.T, name string, gh []response) (replays, string, int, string) {
	t.Helper()
	return runWithReleasePlease(t, name, gh, "OUTPUTS="+outputsReleasePlease)
}

// runWithReleasePlease is runReleasePlease with the step's env also holding env.
func runWithReleasePlease(t *testing.T, name string, gh []response, env ...string) (replays, string, int, string) {
	t.Helper()
	script, ok := stepScript(t, sectionReleasePlease(t, name))
	if !ok {
		t.Fatalf("step %q has no run script", name)
	}
	rp := standIn(t, map[string][]response{"gh": gh})
	output := filepath.Join(t.TempDir(), "output")
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-c", blockAutoMerge(script))
		cmd.Dir = t.TempDir()
		cmd.Env = append(append(userEnv(), rp.env...), "PR=23", "GH_TOKEN=app-token", "GH_REPO="+repoPR, "GITHUB_OUTPUT="+output,
			"GITHUB_REPOSITORY="+repoPR)
		cmd.Env = append(cmd.Env, env...)
		return cmd
	})
	written, _ := os.ReadFile(output)
	return rp, out, code, string(written)
}

// viewReleasePlease is the CLI's answer to reading release PR 23's body.
func viewReleasePlease(body string) response {
	return response{Match: []string{"pr view 23 --json body"}, Stdout: body + "\n"}
}

// headReleasePlease is the CLI's answer to reading release PR 23's head commit.
var headReleasePlease = response{Match: []string{"pr view 23 --json headRefOid"}, Stdout: "def456\n"}

// editsReleasePlease is the CLI's answer to any other pull request call.
var editsReleasePlease = response{Match: []string{"pr "}}

// loggedReleasePlease is text as the stand-in logs it, newlines escaped.
func loggedReleasePlease(text string) string {
	return strings.ReplaceAll(text, "\n", `\n`)
}

// linkedAgainReleasePlease runs the link step again once the tracking issue exists
// as issue 41 and the release PR body already closes it.
func linkedAgainReleasePlease(t *testing.T, title, body string) {
	t.Helper()
	tracked, err := json.Marshal([]map[string]any{{"number": 41, "title": title}})
	mustDo(t, err)
	rp, out, code, written := runReleasePlease(t, "Link the release PR to its tracking issue", []response{
		{Match: []string{"issue list"}, Stdout: string(tracked) + "\n"},
		viewReleasePlease(body), editsReleasePlease,
	})
	sees(t, out, code, 0)
	if rp.called("gh", "issue create") || rp.called("gh", "pr edit") || written != "" {
		t.Errorf("a linked release PR was changed again; gh calls: %q, output %q", rp.calls("gh"), written)
	}
}

var (
	outputRefReleasePlease = regexp.MustCompile(`steps\.\w+\.outputs\.\w+`)
	fromJSONReleasePlease  = regexp.MustCompile(`fromJSON\(([^)]*)\)`)
)

// runsReleasePlease evaluates the step's `if:` as Actions does, given the outputs of
// the earlier steps keyed step.output; a step without `if:` runs.
func runsReleasePlease(t *testing.T, name string, outputs map[string]string) bool {
	t.Helper()
	_, cond, ok := strings.Cut(sectionReleasePlease(t, name), "if: ${{ ")
	if !ok {
		return true
	}
	cond, _, _ = strings.Cut(cond, " }}\n")
	cond = outputRefReleasePlease.ReplaceAllStringFunc(cond, func(ref string) string {
		return `"` + outputs[strings.TrimPrefix(strings.ReplaceAll(ref, ".outputs.", "."), "steps.")] + `"`
	})
	return exec.Command("bash", "-c", "[[ "+cond+" ]]").Run() == nil
}

// readsEmptyPR reports whether every fromJSON in the step's env gets valid
// JSON when Release Please sets no `pr` output. Actions evaluates env even
// for a skipped step, and fromJSON of an empty string fails the job.
func readsEmptyPR(t *testing.T, name string) bool {
	t.Helper()
	for _, m := range fromJSONReleasePlease.FindAllStringSubmatch(sectionReleasePlease(t, name), -1) {
		arg := "" // the value when steps.release.outputs.pr is empty
		if _, fallback, ok := strings.Cut(m[1], " || '"); ok {
			arg, _, _ = strings.Cut(fallback, "'")
		}
		if !json.Valid([]byte(arg)) {
			return false
		}
	}
	return true
}

// tagReleasePlease runs the major tag step with GitHub answering the tag lookup with
// lookup, and accepting any change to a ref.
func tagReleasePlease(t *testing.T, lookup response) (replays, string, int) {
	t.Helper()
	lookup.Match = []string{"git/ref/tags/v1"}
	rp, out, code, _ := runReleasePlease(t, "Move the major version tag", []response{lookup, {Match: []string{"git/refs"}}})
	return rp, out, code
}

// hasReleasePlease fails the test for each want that text lacks.
func hasReleasePlease(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("want %q in %q", want, text)
		}
	}
}
