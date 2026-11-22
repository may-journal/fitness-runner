package requirements

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func Test0008_1(t *testing.T) {
	cases{
		{"a repo whose `version.txt` says 1.0.1", func(t *testing.T) {
			t.Parallel()
			repo := releaseToolRepo(t, nil)
			out, code := releaseToolRun(t, repo, nil, "version")
			sees(t, out, code, 0, "1.0.1")
			out, code = releaseToolRun(t, repo, nil, "tag")
			sees(t, out, code, 0, "v1.0.1")
		}},
		{"a release tag that differs from `version.txt`", func(t *testing.T) {
			t.Parallel()
			out, code := releaseToolRun(t, releaseToolRepo(t, nil), []string{"GITHUB_REF_NAME=v1.0.2"}, "verify-tag")
			sees(t, out, code, 1, "tag v1.0.2 does not match version.txt v1.0.1")
		}},
		{"no command, an unknown one, a bad flag, or an invalid version", func(t *testing.T) {
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
		}},
	}.run(t)
}

func Test0008_2(t *testing.T) {
	cases{
		{"a release step inside a GitHub Actions job", func(t *testing.T) {
			t.Parallel()
			repo, job := releaseToolRepo(t, nil), t.TempDir()
			out, code := releaseToolRun(t, repo, releaseToolActions(job, "v1.0.1"), "verify-tag")
			sees(t, out, code, 0)
			sees(t, releaseToolSummary(t, job), 0, 0, "## ✅ fitness-release verify-tag", "Completed successfully.")
			out, code = releaseToolRun(t, repo, releaseToolActions(job, "v9.9.9"), "verify-tag")
			sees(t, out, code, 1, "::error::fitness-release verify-tag: tag v9.9.9 does not match version.txt v1.0.1")
			sees(t, releaseToolSummary(t, job), 0, 0, "## ❌ fitness-release verify-tag", "tag v9.9.9 does not match")
		}},
		{"a release step outside GitHub Actions", func(t *testing.T) {
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
		}},
		{"a job summary path it cannot write", func(t *testing.T) {
			t.Parallel()
			job := t.TempDir()
			mustDo(t, os.Mkdir(filepath.Join(job, "summary"), 0o755))
			out, code := releaseToolRun(t, releaseToolRepo(t, nil), releaseToolActions(job, "v9.9.9"), "verify-tag")
			sees(t, out, code, 1, "Could not write GitHub job summary", "Fitness: fitness-release verify-tag: tag v9.9.9 does not match")
		}},
	}.run(t)
}

func Test0008_3(t *testing.T) {
	cases{
		{"GoReleaser build outputs and their checksums", func(t *testing.T) {
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
		}},
		{"built only this host's platform after a full build", func(t *testing.T) {
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
		}},
		{"built release assets", func(t *testing.T) {
			t.Parallel()
			built := filepath.Dir(installer(t))
			repo := t.TempDir()
			mustDo(t, os.Symlink(built, filepath.Join(repo, "out")))
			record := recorded(t, repo)
			describesHost(t, record)
			if len(record.Assets) == 0 {
				t.Fatal("record lists no assets")
			}
			for _, a := range record.Assets {
				realSize(t, filepath.Join(built, a.Name), a.Bytes)
			}
		}},
	}.run(t)
}

func Test0008_4(t *testing.T) {
	cases{
		{"a release PR, behind main, that copies its notes", func(t *testing.T) {
			t.Parallel()
			patch := response{Match: []string{"--method PATCH", "repos/" + repoPR + "/pulls/7"}, Stdout: "{}\n"}
			repo, head := entryPending(t)
			rp := standIn(t, map[string][]response{"gh": {patch, entryReleasePR(t, entryCopiedNotes, head)}})
			env := append(rp.env, "RELEASE_PR={\"number\":7}", "GITHUB_REPOSITORY="+repoPR)
			out, code := releaseToolRun(t, repo, env, "release-pr")
			sees(t, out, code, 0)
			sent := strings.Join(rp.calls("gh"), "\n")
			want, err := json.Marshal(entryShips(head))
			mustDo(t, err)
			if !rp.called("gh", "--method PATCH", strings.Trim(string(want), `"`)) {
				t.Errorf("release PR body does not link each new entry file and the compare view at its head:\n%s", sent)
			}
			if rp.called("gh", "--method PATCH", "add one file") || strings.Contains(sent, entryOldFile) || rp.called("gh", "graphql") {
				t.Errorf("release PR body still copies its notes, links a shipped entry, or the tool committed:\n%s", sent)
			}
			entryAllResolve(t, repo, sent)
		}},
		{"entry files, some without front matter, from before the previous tag and since, or no earlier tag", func(t *testing.T) {
			t.Parallel()
			newest := []string{"### " + entryLater, "Fix: tidy the file.", "### " + entryFirst, "Feat: add one file."}
			notes := entryNotes(t, entryTagged(t, true), "v1.1.0")
			if !entryOrdered(notes, append(newest, "[Compare view]("+entryCompare+")")...) {
				t.Errorf("the notes do not list the new entries newest first, then the compare view:\n%s", notes)
			}
			if strings.Contains(notes, "ship the old file") || strings.Contains(notes, "relatedConfigurations") {
				t.Errorf("the notes list an entry from an earlier release, or front matter:\n%s", notes)
			}
			first := entryNotes(t, entryTagged(t, false), "v1.1.0")
			if !entryOrdered(first, append(newest, "[Commits](https://github.com/"+repoPR+"/commits/v1.1.0)")...) {
				t.Errorf("a first release does not list every entry, then its commits:\n%s", first)
			}
		}},
		{"a release tag the release tool cannot place, from a shallow clone or a missing tag", func(t *testing.T) {
			t.Parallel()
			repo := entryTagged(t, true)
			shallow := t.TempDir()
			git(t, shallow, "clone", "-q", "--depth", "1", "--branch", "v1.1.0", "file://"+repo, ".")
			for _, c := range []struct{ repo, tag, want string }{
				{shallow, "v1.1.0", "the clone is shallow, so the previous release tag is unknown"},
				{repo, "v9.9.9", "v9.9.9"},
			} {
				rp := standIn(t, map[string][]response{"gh": {{Stdout: "{}\n"}}})
				env := append(rp.env, "GITHUB_REF_NAME="+c.tag, "GITHUB_REPOSITORY="+repoPR)
				out, code := releaseToolRun(t, c.repo, env, "release-notes")
				sees(t, out, code, 1, c.want)
				if len(rp.calls("gh")) > 0 {
					t.Errorf("the release tool reached GitHub after failing:\n%s", strings.Join(rp.calls("gh"), "\n"))
				}
			}
		}},
		{"a first-release pull request body with a `## 1.0.0` heading and bold scopes", func(t *testing.T) {
			t.Parallel()
			rp, out, code, written := runReleasePlease(t, "Shape the release PR notes", []response{viewReleasePlease(firstReleaseReleasePlease), headReleasePlease, editsReleasePlease})
			sees(t, out, code, 0)
			shaped := "> Release the changes merged since the last version.\n\n## Background\n\nRelease Please groups merged changes.\n\n## Changelog\n\n---\n\n\n" +
				"### 1.0.0 (2026-10-10)\n\n\n### Features\n\n* library: one API over UIKit and AppKit\n\n---\nAgent: Release Please."
			if !rp.called("gh", "pr edit 23 --body "+loggedReleasePlease(shaped)) {
				t.Errorf("the release PR body was not shaped; gh calls: %q", rp.calls("gh"))
			}
			if written != "edited=true\n" {
				t.Errorf("shape step output = %q, want edited=true", written)
			}
			rp, out, code, _ = runReleasePlease(t, "Rerun pr-check on the shaped body", []response{editsReleasePlease})
			sees(t, out, code, 0)
			if calls := rp.calls("gh"); strings.Join(calls, "\n") != "pr close 23\npr reopen 23" {
				t.Errorf("gh calls after the rewrite = %q, want the PR closed then reopened", calls)
			}
		}},
		{"a release pull request whose heading links `compare/v1.2.0...v1.3.0`", func(t *testing.T) {
			t.Parallel()
			compare := "https://github.com/" + repoPR + "/compare/v1.2.0..."
			body := "> Release the changes.\n\n## Changelog\n\n" +
				"## [1.3.0](" + compare + "v1.3.0) (2026-11-09)\n\n### Bug Fixes\n\n* one fix\n\nAgent: Release Please."
			rp, out, code, _ := runReleasePlease(t, "Shape the release PR notes", []response{
				viewReleasePlease(body), headReleasePlease, editsReleasePlease,
			})
			sees(t, out, code, 0)
			shaped := "> Release the changes.\n\n## Changelog\n\n" +
				"### [1.3.0](" + compare + "def456) (2026-11-09)\n\n### Bug Fixes\n\n* one fix\n\nAgent: Release Please."
			if !rp.called("gh", "pr edit 23 --body "+loggedReleasePlease(shaped)) {
				t.Errorf("the compare link does not end at the release PR's head commit; gh calls: %q", rp.calls("gh"))
			}
		}},
	}.run(t)
}

func Test0008_5(t *testing.T) {
	cases{
		{"a release pull request with no tracking issue", func(t *testing.T) {
			t.Parallel()
			const title = "Publish the release from PR #23"
			body := "> Release the changes.\n\nAgent: Release Please."
			rp, out, code, written := runReleasePlease(t, "Link the release PR to its tracking issue", []response{
				{Match: []string{"issue list"}, Stdout: "[]\n"},
				{Match: []string{"issue create --title " + title}, Stdout: "https://github.com/" + repoPR + "/issues/41\n"},
				viewReleasePlease(body), editsReleasePlease,
			})
			sees(t, out, code, 0)
			if !rp.called("gh", "pr edit 23 --body "+loggedReleasePlease(body+"\n\nCloses #41.")) {
				t.Errorf("the release PR does not close its new tracking issue; gh calls: %q", rp.calls("gh"))
			}
			if written != "edited=true\n" {
				t.Errorf("link step output = %q, want edited=true", written)
			}
			if !rp.called("gh", "issue list --state all --search \""+title+"\" in:title") {
				t.Errorf("the tracking issue lookup does not search by title; gh calls: %q", rp.calls("gh"))
			}
			linkedAgainReleasePlease(t, title, body+"\n\nCloses #41.")
		}},
		{"a release created with no pull request", func(t *testing.T) {
			t.Parallel()
			noPR := map[string]string{"release.releases_created": "true"}
			for _, step := range []string{"Shape the release PR notes", "Link the release PR to its tracking issue", "Rerun pr-check on the shaped body"} {
				if runsReleasePlease(t, step, noPR) {
					t.Errorf("with no release PR, step %q runs, want it skipped", step)
				}
				if !readsEmptyPR(t, step) {
					t.Errorf("with no release PR, step %q cannot read its env: fromJSON gets an empty string", step)
				}
			}
			if !runsReleasePlease(t, "Move the major version tag", noPR) {
				t.Error("with a release created, the major version tag step is skipped, want it run")
			}
		}},
	}.run(t)
}

func Test0008_6(t *testing.T) {
	cases{
		{"a created release whose major tag exists, then is missing, then cannot be read", func(t *testing.T) {
			t.Parallel()
			refs := "repos/" + repoPR + "/git/refs"
			rp, out, code := tagReleasePlease(t, response{})
			sees(t, out, code, 0)
			if !rp.called("gh", "-X PATCH "+refs+"/tags/v1 -f sha=abc123") {
				t.Errorf("an existing major tag was not moved; gh calls: %q", rp.calls("gh"))
			}
			rp, out, code = tagReleasePlease(t, response{Stderr: "gh: Not Found (HTTP 404)\n", Exit: 1})
			sees(t, out, code, 0)
			if !rp.called("gh", "-X POST "+refs+" -f ref=refs/tags/v1 -f sha=abc123") {
				t.Errorf("a missing major tag was not created; gh calls: %q", rp.calls("gh"))
			}
			rp, out, code = tagReleasePlease(t, response{Stderr: "gh: Resource not accessible by integration (HTTP 403)\n", Exit: 1})
			sees(t, out, code, 1, "::error::Could not read tag v1: gh: Resource not accessible by integration (HTTP 403)")
			if rp.called("gh", "-X ") {
				t.Errorf("a tag lookup error still changed a ref; gh calls: %q", rp.calls("gh"))
			}
		}},
		{"a monorepo whose release created the root and two components, each component tagged `<component>-vX.Y.Z`", func(t *testing.T) {
			t.Parallel()
			// The root and two components release together.
			outputs := `{"releases_created":"true","paths_released":"[\".\",\"cmd/fitness-check-foo\",\"cmd/fitness-check-bar\"]","tag_name":"v3.0.0","major":"3","sha":"ccc333",` +
				`"cmd/fitness-check-foo--tag_name":"fitness-check-foo-v1.2.0","cmd/fitness-check-foo--major":"1","cmd/fitness-check-foo--sha":"aaa111",` +
				`"cmd/fitness-check-bar--tag_name":"fitness-check-bar-v2.0.0","cmd/fitness-check-bar--major":"2","cmd/fitness-check-bar--sha":"bbb222"}`
			refs := "repos/" + repoPR + "/git/refs"
			rp, out, code, written := runWithReleasePlease(t, "Move the major version tag", []response{
				{Match: []string{"git/ref/tags/v3"}},
				{Match: []string{"git/ref/tags/fitness-check-foo-v1"}},
				{Match: []string{"git/ref/tags/fitness-check-bar-v2"}, Stderr: "gh: Not Found (HTTP 404)\n", Exit: 1},
				{Match: []string{"git/refs"}},
			}, "OUTPUTS="+outputs)
			sees(t, out, code, 0)
			calls := strings.Join(rp.calls("gh"), "\n")
			hasReleasePlease(t, calls, "-X PATCH "+refs+"/tags/v3 -f sha=ccc333", "-X PATCH "+refs+"/tags/fitness-check-foo-v1 -f sha=aaa111",
				"-X POST "+refs+" -f ref=refs/tags/fitness-check-bar-v2 -f sha=bbb222")
			hasReleasePlease(t, written, `"path":"cmd/fitness-check-foo","tag":"fitness-check-foo-v1.2.0","major_tag":"fitness-check-foo-v1","sha":"aaa111"`,
				`"path":"cmd/fitness-check-bar","tag":"fitness-check-bar-v2.0.0","major_tag":"fitness-check-bar-v2","sha":"bbb222"`)
			if rp.called("gh", "tags/v1") || rp.called("gh", "tags/v2") {
				t.Errorf("a component moved a root major tag; gh calls: %q", calls)
			}
		}},
		{"a monorepo release whose component tag omits its name, or whose output has no commit", func(t *testing.T) {
			t.Parallel()
			// A component tagged without its name would move another component's tag.
			for _, outputs := range []string{
				`{"releases_created":"true","paths_released":"[\"cmd/a\"]","cmd/a--tag_name":"v1.0.0","cmd/a--major":"1","cmd/a--sha":"aaa111"}`,
				`{"releases_created":"true","paths_released":"[\"cmd/a\"]","cmd/a--tag_name":"a-v1.0.0","cmd/a--major":"1"}`,
			} {
				rp, out, code, _ := runWithReleasePlease(t, "Move the major version tag", nil, "OUTPUTS="+outputs)
				sees(t, out, code, 1, "::error::")
				if rp.called("gh", "-X ") {
					t.Errorf("a bad release still changed a ref; gh calls: %q", rp.calls("gh"))
				}
			}
		}},
		{"a release of the root and a component whose tag omits its name", func(t *testing.T) {
			t.Parallel()
			// The root is valid and a later component is not, so no tag may move.
			outputs := `{"releases_created":"true","paths_released":"[\".\",\"cmd/a\"]","tag_name":"v1.0.0","major":"1","sha":"aaa111",` +
				`"cmd/a--tag_name":"v2.0.0","cmd/a--major":"2","cmd/a--sha":"bbb222"}`
			rp, out, code, _ := runWithReleasePlease(t, "Move the major version tag", nil, "OUTPUTS="+outputs)
			sees(t, out, code, 1, "::error::")
			if rp.called("gh", "-X ") {
				t.Errorf("a bad later release still changed a ref; gh calls: %q", rp.calls("gh"))
			}
		}},
	}.run(t)
}

func Test0008_7(t *testing.T) {
	refs := cases{{"every step's reference to a subcommand or script", func(t *testing.T) {
		t.Parallel()
		subs, named := subcommandOutputs(t), 0
		for _, file := range yamlFiles(t, "../..") {
			n, bad := badReferences(t, file, subs)
			named += n
			for _, ref := range bad {
				t.Errorf("%s names %q, which fitness-ci lacks or which is missing or not executable", file, ref)
			}
		}
		if named < 40 {
			t.Errorf("want the test to read the repo's subcommand and script references, found %d", named)
		}
	}}}
	append(append(refs, ciStepCases()...), releaseStepCases()...).run(t)
}

func Test0008_8(t *testing.T) {
	t.Parallel()
	action, err := os.ReadFile("../../.github/actions/fitness-ci/action.yml")
	mustDo(t, err)
	subs, checked := subcommandOutputs(t), 0
	for _, file := range yamlFiles(t, "../..") {
		checked += undeclaredOutputs(t, file, string(action), subs)
	}
	if checked < 8 {
		t.Errorf("want the test to check the outputs of the action's subcommands, checked %d", checked)
	}
}

// releaseStepCases run each subcommand a release or distribution step calls,
// the way the step calls it, for 0008.7.
func releaseStepCases() cases {
	return cases{
		{"the release workflow's verify step", func(t *testing.T) {
			t.Parallel()
			repo, job := releaseToolRepo(t, nil), t.TempDir()
			output := filepath.Join(job, "output")
			out, code := releaseToolRun(t, repo, append(releaseToolActions(job, "v1.0.1"), "GITHUB_OUTPUT="+output), "verify-tag")
			sees(t, out, code, 0)
			written, err := os.ReadFile(output)
			mustDo(t, err)
			if string(written) != "version=1.0.1\n" {
				t.Errorf("verify-tag must write the version for later jobs, wrote %q", written)
			}
		}},
		{"a bundle build on a branch", func(t *testing.T) {
			t.Parallel()
			run := runFitnessCI(t, ciStep{file: "distribution.yml", command: "build-release-bundle", env: []string{"SNAPSHOT=true"},
				saved: map[string][]response{
					"go":         {{Match: []string{"fitness-release version"}, Stdout: "1.2.3\n"}, {Match: []string{"fitness-release bundle-hashes"}, Stdout: "hashes\n"}, {Match: []string{"assemble"}}},
					"goreleaser": {{Match: []string{"release"}}},
				}})
			sees(t, run.out, run.code, 0)
			hasAll(t, "goreleaser calls", strings.Join(run.rp.calls("goreleaser"), "\n"),
				"release --clean --skip=publish --snapshot --config .goreleaser-bundles.yml",
				"release --clean --skip=publish --snapshot --config .goreleaser-tools.yml")
			hasAll(t, "go calls", strings.Join(run.rp.calls("go"), "\n"), "-C go run ./cmd/fitness-release assemble --root ..")
		}},
		{"a release still a draft, and an answer that is neither", func(t *testing.T) {
			t.Parallel()
			view := func(answer string) map[string][]response {
				return map[string][]response{"gh": {{Match: []string{"release view v1.2.3 --json isDraft"}, Stdout: answer + "\n"}}}
			}
			run := runFitnessCI(t, ciStep{file: "release.yml", command: "inspect-publication", env: []string{"RELEASE_TAG=v1.2.3"}, saved: view("true")})
			sees(t, run.out, run.code, 0)
			if run.outputs != "draft=true\n" {
				t.Errorf("want draft=true written, got %q", run.outputs)
			}
			run = runFitnessCI(t, ciStep{file: "release.yml", command: "inspect-publication", env: []string{"RELEASE_TAG=v1.2.3"}, saved: view("maybe")})
			sees(t, run.out, run.code, 1, "maybe")
		}},
		{"a downloaded release tool that lost its run bits", func(t *testing.T) {
			t.Parallel()
			for command, want := range map[string]string{"write-release-notes": "release-notes", "promote-release": "promote"} {
				dir := t.TempDir()
				writeTools(t, dir, map[string]string{"out/fitness-release-linux-amd64": tool("fitness-release")})
				run := runFitnessCI(t, ciStep{file: "release.yml", command: command, dir: dir})
				sees(t, run.out, run.code, 0, "fitness-release "+want)
			}
		}},
		{"a release commit the Go module tag names, then another", func(t *testing.T) {
			t.Parallel()
			repo, origin := taggedRepo(t, "v1.2.3")
			step := ciStep{file: "release.yml", command: "publish-go-module-tag", env: []string{"RELEASE_TAG=v1.2.3"}, dir: repo}
			run := runFitnessCI(t, step)
			sees(t, run.out, run.code, 0)
			if got := tagCommit(origin, "go/v1.2.3"); got == "" || got != tagCommit(repo, "v1.2.3") {
				t.Fatalf("origin must hold go/v1.2.3 at the release commit, got %q", got)
			}
			commit(t, repo, map[string]string{"CHANGELOG.md": "# Changelog\n"}, "docs(changelog): add changelog")
			git(t, repo, "tag", "-f", "v1.2.3")
			run = runFitnessCI(t, step)
			sees(t, run.out, run.code, 1, "::error::Tag go/v1.2.3 names")
		}},
		{"the newest release, and an older one", func(t *testing.T) {
			t.Parallel()
			for latest, moves := range map[string]bool{"v1.2.3": true, "v1.3.0": false} {
				repo, origin := taggedRepo(t, "v1.2.3")
				run := runFitnessCI(t, ciStep{file: "release.yml", command: "move-engine-major-tag", env: []string{"RELEASE_TAG=v1.2.3"}, dir: repo,
					saved: map[string][]response{"gh": {{Match: []string{"release view --json tagName"}, Stdout: latest + "\n"}}}})
				sees(t, run.out, run.code, 0)
				if moved := tagCommit(origin, "v1") == tagCommit(repo, "v1.2.3"); moved != moves {
					t.Errorf("with %s newest, v1 moved = %v, want %v", latest, moved, moves)
				}
			}
		}},
		{"a downloaded verifier that lost its run bits", func(t *testing.T) {
			t.Parallel()
			for file, command := range map[string]string{"distribution.yml": "run-verifier smoke", "release.yml": "run-verifier verify-download"} {
				dir := t.TempDir()
				writeTools(t, dir, map[string]string{"out/fitness-release-linux-amd64": tool("verifier")})
				run := runFitnessCI(t, ciStep{file: file, command: command, dir: dir, env: []string{"VERIFIER=out/fitness-release-linux-amd64"}})
				sees(t, run.out, run.code, 0, "verifier "+strings.Fields(command)[1])
			}
		}},
		{"a smoke job recording binary sizes", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTools(t, dir, map[string]string{"out/fitness-release-linux-amd64": "#!/bin/sh\necho '{\"size\":1}'\n"})
			run := runFitnessCI(t, ciStep{file: "distribution.yml", command: "record-binary-sizes", dir: dir,
				env: []string{"VERIFIER=out/fitness-release-linux-amd64", "PLATFORM=linux-amd64"}})
			sees(t, run.out, run.code, 0)
			record, err := os.ReadFile(filepath.Join(run.temp, "fitness-performance", "release-linux-amd64.json"))
			if err != nil || string(record) != "{\"size\":1}\n" {
				t.Errorf("the record must land in the performance folder, got %q (%v)", record, err)
			}
		}},
		{"a candidate installer that lost its run bits", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTools(t, dir, map[string]string{
				"out/fitness-install-1.2.3-linux-amd64": tool("installer"),
				"tests/requirements-shard.sh":           "echo \"shard $1 of $2 with $FITNESS_INSTALL\"\n",
			})
			run := runFitnessCI(t, ciStep{file: "distribution.yml", command: "run-smoke-suite", dir: dir, env: []string{"PLATFORM=linux-amd64"}})
			sees(t, run.out, run.code, 0, "installer --with-checks -- --policy=external")
			run = runFitnessCI(t, ciStep{file: "distribution.yml", command: "run-requirement-shard", dir: dir, env: []string{"PLATFORM=linux-amd64", "SHARD=2"}})
			real, err := filepath.EvalSymlinks(dir)
			mustDo(t, err)
			sees(t, run.out, run.code, 0, "shard 2 of 2 with "+filepath.Join(real, "out", "fitness-install-1.2.3-linux-amd64"))
		}},
	}
}
