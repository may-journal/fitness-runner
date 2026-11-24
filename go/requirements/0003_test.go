// cspell:ignore zorbleflux

package requirements

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test0003_1(t *testing.T) {
	cases{
		{"every check passing, with docs whose front matter only another tool reads", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", architectedRepo())
			out, code := fitness(t, repo, nil)
			sees(t, out, code, 0, allPassed())
		}},
		{"a check that fails", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
			out, code := fitness(t, repo, nil)
			sees(t, out, code, 1, "README.md: disallowed **bold**", passedOf(1)+", 1 failed")
		}},
		{"a new check binary", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			out, _ := fitness(t, repo, nil)
			for _, bin := range bundleChecks(t, installOnly(t)) {
				if row(out, strings.TrimPrefix(bin, "fitness-check-")) == "" && bin != "fitness-check-go-test-coverage" {
					t.Errorf("%s is installed but fitness never ran it", bin)
				}
			}
		}},
		{"a tracked-file list from another repo, or from before the index changed", func(t *testing.T) {
			t.Parallel()
			other := example(t, "happyRepo", map[string]string{"other.md": "# Other\n\nSome **bold** text.\n"})
			installed, bundle := installOnly(t), t.TempDir()
			linkTools(bundle, installed, []string{"fitness"})
			placeRunner(t, filepath.Join(installed, "fitness"), filepath.Join(bundle, "fitness"))
			walk := "exec \"" + installedCheck(t, "markdown-no-bold-italic") + "\" --root "
			write(t, bundle, map[string]string{
				// Each runs the real check under the list its run handed it: one
				// after staging a new file, one on another repo.
				"fitness-check-staged-later": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\":\"staged-later\"}'; exit 0; }\n" +
					"printf '# New\\n\\nSome **bold** text.\\n' > new.md\ngit add new.md\n" + walk + "\"$PWD\"\n",
				"fitness-check-other-repo": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\":\"other-repo\"}'; exit 0; }\n" + walk + "\"" + other + "\"\n",
			})
			repo := example(t, "happyRepo", nil)
			out, code := execute(t, func() *exec.Cmd {
				cmd := exec.Command(filepath.Join(bundle, "fitness"), "--policy=external", "--checks=staged-later,other-repo")
				cmd.Dir, cmd.Env = repo, userEnv()
				return cmd
			})
			sees(t, out, code, 1, "new.md: disallowed **bold**", "other.md: disallowed **bold**")
		}},
		{"a check that panics, crashes, is killed, exits oddly, cannot start or describe itself, or meets prompts or silent proxies", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", with(goRepo(), map[string]string{"broken_test.go": "package adder\n\nfunc Test0001_2(t *testing.T {\n"}))
			out, code := fitness(t, repo, nil)
			sees(t, out, code, 1, "[requirements] Info:", "missing ',' in parameter list", "go-vet")
			if !strings.Contains(row(out, "requirements"), "info") || !strings.Contains(row(out, "go-vet"), "failed") {
				t.Errorf("requirements must show info while go-vet sets the exit code, got rows %q and %q", row(out, "requirements"), row(out, "go-vet"))
			}
			out, code = describeFails(t, example(t, "happyRepo", nil), "markdown-no-bold-italic")
			sees(t, out, code, 0, "--describe gave no metadata, so the check ran with defaults", "0 of 1 checks passed, 1 info")
			if !strings.Contains(row(out, "markdown-no-bold-italic"), "info") {
				t.Errorf("a check whose describe fails must still run and show info, got row %q", row(out, "markdown-no-bold-italic"))
			}
			out, code = fitness(t, example(t, "happyRepo", architectedRepo()), []string{killedVet(t)})
			sees(t, out, code, 0, "check ended without a result (signal: killed)", passedOf(1)+", 1 info")
			if !strings.Contains(row(out, "go-vet"), "info") {
				t.Errorf("a killed check must show info while the passing checks set exit 0, got row %q", row(out, "go-vet"))
			}
			repo = example(t, "happyRepo", with(architectedRepo(), map[string]string{"README.md": readme + "\nSome **bold** text.\n"}))
			summary := filepath.Join(t.TempDir(), "summary.md")
			out, code = fitness(t, repo, []string{killedVet(t), "GITHUB_ACTIONS=true", "GITHUB_WORKSPACE=" + repo, "GITHUB_STEP_SUMMARY=" + summary})
			sees(t, out, code, 1, passedOf(2)+", 1 failed, 1 info", "::error file=README.md",
				"::notice title=go-vet::check ended without a result (signal: killed)")
			written, _ := os.ReadFile(summary)
			sees(t, string(written), code, 1, "| go-vet | ℹ️ info |", "### ℹ️ Info")
			promptsRefused(t)
			offlineCommitAndPush(t)
		}},
		{"a fitness runner alone in its folder", func(t *testing.T) {
			t.Parallel()
			alone := filepath.Join(t.TempDir(), "fitness")
			placeRunner(t, filepath.Join(installOnly(t), "fitness"), alone)
			out, code := runAlone(t, alone, example(t, "happyRepo", architectedRepo()), "--policy=external", "--all")
			sees(t, out, code, 0, allPassed())
			bold := example(t, "happyRepo", map[string]string{"README.md": readme + "\nSome **strong** words.\n"})
			out, code = runAlone(t, alone, bold, "check", "markdown-no-bold-italic")
			sees(t, out, code, 1, `README.md: disallowed **bold**`+noBoldItalicRule, "0 of 1 checks passed")
		}},
	}.run(t)
}

func Test0003_2(t *testing.T) {
	cases{
		{"a check name that does not exist", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "no-such-check")
			sees(t, out, code, 1, "Unknown check: no-such-check")
		}},
		{"a .fitnessrc.json that is not valid JSON", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": "{\n"}), nil)
			sees(t, out, code, 1, "fitness setup: .fitnessrc.json:")
		}},
		{"asks for help", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "--help")
			sees(t, out, code, 0, "fitness — run the configured checks.", "fitness init")
		}},
		{"a hook name fitness does not know", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "hook", "no-such-hook")
			sees(t, out, code, 1, "fitness hook: unknown hook no-such-hook")
		}},
		{"a folder that is not a git repo", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			out, code := fitness(t, dir, nil, "init")
			sees(t, out, code, 1, "fitness init: git config: fatal: not in a git directory")
			if _, err := os.Stat(filepath.Join(dir, ".githooks")); err == nil {
				t.Error("fitness init wrote .githooks outside a git repo")
			}
		}},
		{"a folder that is not a git repo: it says why, naming the git call that failed, and exits 1", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, t.TempDir(), nil, "hook", "pre-push")
			sees(t, out, code, 1, "fitness hook: git rev-parse: fatal: not a git repository")
		}},
		{"the commit-msg hook with no message file", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "hook", "commit-msg")
			sees(t, out, code, 1, "fitness hook commit-msg: missing message file")
		}},
		{"a `--json` flag with no file, or a file it cannot write", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			out, code := fitness(t, repo, nil, "--policy=external", "--json")
			sees(t, out, code, 1, "use --json=<file> once")
			blocked := filepath.Join(t.TempDir(), "file")
			mustDo(t, os.WriteFile(blocked, []byte("x"), 0o644))
			out, code = fitness(t, repo, nil, "--policy=external", "--all", "--json="+filepath.Join(blocked, "fitness.json"))
			sees(t, out, code, 1, "fitness --json")
		}},
		{"a report file that is missing or not a `--json` report", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			dir := t.TempDir()
			page := filepath.Join(dir, "fitness.html")
			out, code := fitness(t, repo, nil, "boxes", filepath.Join(dir, "missing.json"), page)
			sees(t, out, code, 1, "fitness boxes", "missing.json")
			notReport := filepath.Join(dir, "other.json")
			mustDo(t, os.WriteFile(notReport, []byte(`{"name": "app"}`), 0o644))
			out, code = fitness(t, repo, nil, "boxes", notReport, page)
			sees(t, out, code, 1, "not a fitness --json report")
			out, code = fitness(t, repo, nil, "boxes", notReport)
			sees(t, out, code, 1, "use fitness boxes <report.json> <page.html>")
			if _, err := os.Stat(page); err == nil {
				t.Error("a failed run must not write a page")
			}
		}},
		{"names a folder that is missing, outside the repo, or holds no files", func(t *testing.T) {
			t.Parallel()
			repo := dirRepo(t)
			mustDo(t, os.Mkdir(filepath.Join(repo, "empty"), 0o755))
			for _, dir := range []string{"nowhere", "../elsewhere", "empty"} {
				out, code := fitness(t, repo, nil, kebabDocsCheck, "--dir="+dir)
				sees(t, out, code, 1, "--dir "+dir)
				if strings.Contains(out, "releaseNotes") || strings.Contains(out, "userGuide") {
					t.Errorf("--dir=%s judged the repo:\n%s", dir, out)
				}
			}
		}},
		{"a `--format` flag naming no known format", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "--policy=external", "--format=yaml")
			sees(t, out, code, 1, "use --format=json or --format=table once")
		}},
	}.run(t)
}

func Test0003_3(t *testing.T) {
	cases{
		{"bad files in two folders", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, dirRepo(t), nil, kebabDocsCheck, "--dir=guides")
			sees(t, out, code, 1, "guides/userGuide.md: filename must be kebab-case")
			if strings.Contains(out, "releaseNotes") {
				t.Errorf("the output names a file outside guides:\n%s", out)
			}
		}},
		{"names a folder of requirement docs whose tests are elsewhere", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", scriptRepo())
			out, code := fitness(t, repo, nil, "requirements", "--dir=docs")
			sees(t, out, code, 0)
			passedWith(t, out, "requirements", "2")
		}},
		{"an installed check whose `files` pattern finds only Go files, and one that finds Markdown", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{"notes/plan.md": "# Plan\n", "src/main.go": "package main\n"})
			bundle := bundleDiscovery(t)
			write(t, bundle, map[string]string{
				"fitness-check-go-files": scriptApplies("go-files", `{"name":"go-files","files":"\\.go$"}`),
				"fitness-check-md-files": scriptApplies("md-files", `{"name":"md-files","files":"\\.md$","exclude":"^CHANGELOG"}`),
			})
			out, code := runApplies(t, bundle, repo, "", "--policy=external", "--checks=go-files,md-files", "--dir=notes")
			sees(t, out, code, 1, "md-files ran", "Skipped (no matching files): go-files")
			absent(t, out, "go-files")
			out, code = runApplies(t, bundle, repo, "", "--policy=external", "--checks=go-files,md-files")
			sees(t, out, code, 1, "go-files ran", "md-files ran")
		}},
		{"an installed check whose `needs` pattern names a `package.json` the repo does not track", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			bundle := bundleDiscovery(t)
			write(t, bundle, map[string]string{
				"fitness-check-node-files": scriptApplies("node-files", `{"name":"node-files","needs":"(^|/)package\\.json$"}`),
				"fitness-check-any-files":  scriptApplies("any-files", `{"name":"any-files"}`),
			})
			out, code := runApplies(t, bundle, repo, "", "--policy=external", "--checks=node-files,any-files")
			sees(t, out, code, 1, "any-files ran", "Skipped (the repo tracks no file its needs pattern names): node-files")
			absent(t, out, "node-files")
		}},
		{"an installed check whose `files` pattern does not compile, and one that requires what this fitness does not know", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			bundle := bundleDiscovery(t)
			write(t, bundle, map[string]string{
				"fitness-check-lookahead": scriptApplies("lookahead", `{"name":"lookahead","files":"^(?!vendor/)"}`),
				"fitness-check-teleport":  scriptApplies("teleport", `{"name":"teleport","requires":["teleport"]}`),
				"fitness-check-any-files": scriptApplies("any-files", `{"name":"any-files"}`),
			})
			out, code := runApplies(t, bundle, repo, "", "--policy=external", "--checks=lookahead,any-files")
			sees(t, out, code, 1, `check lookahead files pattern "^(?!vendor/)" does not compile`,
				"go install github.com/may-journal/fitness-checks/cmd/fitness-check-lookahead@latest")
			absent(t, out, "lookahead", "any-files")
			out, code = runApplies(t, bundle, repo, "", "--policy=external", "--checks=teleport,any-files")
			sees(t, out, code, 1, "any-files ran", `Skipped (requires "teleport", which this fitness does not know; upgrade fitness): teleport`)
			absent(t, out, "teleport")
		}},
		{"two installed checks that judge a text, one of which requires the repo", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			bundle := bundleDiscovery(t)
			write(t, bundle, map[string]string{
				"fitness-check-text-only": scriptApplies("text-only", `{"name":"text-only","blob":true}`),
				"fitness-check-repo-wide": scriptApplies("repo-wide", `{"name":"repo-wide","blob":true,"requires":["repo"]}`),
			})
			out, code := runApplies(t, bundle, repo, "Some text.\n", "--blob", "-", "--checks=text-only,repo-wide")
			sees(t, out, code, 1, "text-only ran", "Skipped (requires repo): repo-wide")
			absent(t, out, "repo-wide")
		}},
		{"an installed check that passes and one the run skips", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{"notes/plan.md": "# Plan\n"})
			bundle := bundleDiscovery(t)
			write(t, bundle, map[string]string{
				"fitness-check-md-files": "#!/bin/sh\n[ \"$1\" = --describe ] && { printf '%s\\n' '{\"name\":\"md-files\",\"files\":\"\\\\.md$\"}'; exit 0; }\n" +
					"echo '{\"ok\":true,\"errors\":[],\"filesChecked\":1}'\n",
				"fitness-check-go-files": scriptApplies("go-files", `{"name":"go-files","files":"\\.go$"}`),
			})
			args := []string{"--policy=external", "--checks=md-files,go-files", "--dir=notes"}
			out, code := runApplies(t, bundle, repo, "", args...)
			sees(t, out, code, 0, "All 2 checks passed, 1 skipped", "Skipped (no matching files): go-files")
			if !strings.Contains(row(out, "go-files"), "skipped") {
				t.Errorf("go-files must show as a skipped row:\n%s", out)
			}
			out, code = runApplies(t, bundle, repo, "", append(args, "--format=json")...)
			sees(t, out, code, 0, `"status": "skipped"`, `"skipped": "no matching files"`)
			absent(t, out, "go-files")
		}},
		{"a check requiring `diff`", func(t *testing.T) {
			t.Parallel()
			bundle, repo := requiringCache(t, map[string]string{"diff-reads": `"diff"`})
			env := []string{"FITNESS_CACHE_DIR=" + t.TempDir()}
			out, code := runDiscovery(t, bundle, repo, env, "--policy=external", "--checks=diff-reads", "--all")
			sees(t, out, code, 0, "Skipped (requires diff, which only a run of changed files has): diff-reads")
			absent(t, out, "diff-reads")

			write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
			git(t, repo, "add", "-A")
			out, code = runDiscovery(t, bundle, repo, env, "--policy=external", "--checks=diff-reads")
			sees(t, out, code, 1, "diff-reads ran")
		}},
		{"installed checks of scope `git`, of scope `blob`, of scope `blob` finding Markdown, and one excluding the folder", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{"notes/plan.md": "# Plan\n"})
			out, code := runApplies(t, bundlePaths(t), repo, "", "--policy=external", "--checks=history-rule,body-rule,md-body,no-notes", "--dir=notes")
			sees(t, out, code, 1, "history-rule ran", "md-body ran", "Skipped (no matching files): body-rule, no-notes")
			absent(t, out, "body-rule", "no-notes")
		}},
		{"installed checks of scope `blob`, of `blob: true` finding Go files, of Go files alone, and of scope `git` with `blob: true`", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			bundle := bundlePaths(t)
			selected := "--checks=body-rule,go-text,go-files,git-text"
			out, code := runApplies(t, bundle, repo, "Some text.\n", "--blob", "-", selected)
			sees(t, out, code, 1, "body-rule ran", "Skipped (no matching files): go-text", "Skipped (needs files): go-files", "Skipped (requires history): git-text")
			absent(t, out, "go-text", "go-files", "git-text")
			out, code = runApplies(t, bundle, repo, "Some text.\n", "--blob", "-", selected, "--as", "x.go")
			sees(t, out, code, 1, "go-text ran", "Skipped (no matching files): body-rule", "Skipped (needs files): go-files")
			absent(t, out, "body-rule", "go-files", "git-text")
		}},
		{"a local check of scope `blob` and one of scope `github`", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{
				".fitnessrc.json":                `{"localChecks": "checks"}` + "\n",
				"checks/fitness-check-body-rule": scriptApplies("body-rule", checksPaths["body-rule"]),
				"checks/fitness-check-gh-rule":   scriptApplies("gh-rule", `{"name":"gh-rule","scope":"github"}`),
			})
			out, code := fitness(t, repo, nil, "list", "--json")
			sees(t, strings.ReplaceAll(out, `\`, ""), code, 0, `"files": "^.fitness/(github/|blob.md$)"`, `"requires": [`+"\n"+`      "github"`)
			out, code = fitness(t, repo, nil, "explain", "body-rule")
			sees(t, out, code, 0, "Files: `^\\.fitness/(github/|blob\\.md$)` · Exclude: `^\\.git/(COMMIT_EDITMSG|logs/HEAD)$` · Requires: -")
			out, code = fitness(t, repo, nil, "explain", "gh-rule")
			sees(t, out, code, 0, "Requires: github")
		}},
		{"a local check replacing a PR body check, whose `files` finds only comment bodies and which always fails", func(t *testing.T) {
			t.Parallel()
			for files, want := range map[string]int{`^\\.fitness/github/comment/`: 0, `^\\.fitness/github/pull/`: 1} {
				describe := `{"name":"markdown-no-bold-italic","blob":true,"files":"` + files + `"}`
				repo := example(t, "happyRepo", map[string]string{
					".fitnessrc.json": `{"policy": "external", "localChecks": "checks"}` + "\n",
					"checks/fitness-check-markdown-no-bold-italic": scriptApplies("markdown-no-bold-italic", describe),
				})
				out, code := boldPR(t, repo)
				if ran := strings.Contains(out, "markdown-no-bold-italic ran"); code != want || ran != (want == 1) {
					t.Errorf("with files %s, pr-check must exit %d and run the check only when the PR body's path reaches it, got %d:\n%s", files, want, code, out)
				}
			}
		}},
	}.run(t)
}

func Test0003_4(t *testing.T) {
	cases{
		{"slow checks, one of them a test runner", func(t *testing.T) {
			t.Parallel()
			// slow-zz is a test runner but sits last in catalog order, so it would
			// wait for a free pool slot.
			stages := map[string]string{"slow-zz": "tests"}
			for i := range runtime.NumCPU() {
				stages[fmt.Sprintf("slow-%02d", i)] = "fast"
			}
			out, code, stamps := timedRun(t, stages)
			sees(t, out, code, 0, fmt.Sprintf("All %d checks passed", len(strings.Fields(catalogRows))+len(stages)))
			describes, runs := stamps.phase(t, "describe", stages), stamps.phase(t, "run", stages)
			// One core gives the pool one slot, so only there may steps not overlap.
			if runtime.NumCPU() > 1 && overlapping(describes) < 2 {
				t.Errorf("the describe steps must overlap, but none started before the first ended: %v", describes)
			}
			if long, first := runs["slow-zz"], earliestEnd(runs, "slow-zz"); !long.start.Before(first) {
				t.Errorf("slow-zz is a test runner, so it must start before any other check ends: it started %v, the first end was %v", long.start, first)
			}
			keepsCatalogOrder(t, out, slices.Sorted(maps.Keys(stages)))
			showsProgress(t)
			runnerTakesNoSlot(t)
		}},
		{"a started check, like `cspell` or `duplicate-code` on 20,000 files, past the fast total, or whose git or ssh prompts", func(t *testing.T) {
			t.Parallel()
			// A 0.3s fast total stops prettier, the one fast check, so it starts on
			// a loaded machine. That total never stops go-test, in the test stage.
			for _, prelude := range []string{"", "trap '' TERM\n"} {
				repo := example(t, "happyRepo", hungTool(prelude))
				out, code := fitness(t, repo, []string{fastTotal(300)}, "--policy=external", "--checks=prettier,go-test")
				sees(t, out, code, 1, "Stopped at the 0.3s fast total")
				if !strings.Contains(row(out, "prettier"), "failed") {
					t.Errorf("a stopped check must report red, got row %q", row(out, "prettier"))
				}
				if strings.Contains(row(out, "go-test"), "failed") {
					t.Errorf("the fast total must never stop a test-stage check, got row %q", row(out, "go-test"))
				}
				childStopped(t, repo)
			}
			promptsTimeOut(t)
			timeoutNamesLastLine(t)
			hugeTreeHints(t)
			timesTrees(t)
		}},
		{"500,000 tracked files, symlinks or deleted ones, or a git call listing tracked or staged files that stalls", func(t *testing.T) {
			t.Parallel()
			for _, repo := range []string{linkedRepo(t), generatedTree(t, architectedRepo(), 3000)} {
				path, calls := loggedGit(t)
				out, code := fitness(t, repo, []string{path})
				sees(t, out, code, 0, allPassed())
				if n := occurrences(calls, "ls-files -z --stage"); n != 1 {
					t.Errorf("git must list the tracked files once, and the runner stat them once, for the whole run, but listed them %d times", n)
				}
				if n := occurrences(calls, "ls-files -z --cached"); n != 0 {
					t.Errorf("every check must use the shared list, but %d listed their own", n)
				}
			}
			hugeProductCommit(t)
			stalledGitStops(t)
		}},
		{"a check past the fast total, whose tool starts children or ignores the stop signal, while every CPU is busy", func(t *testing.T) {
			// Not parallel: it keeps every CPU busy, which would slow the tests beside it.
			busyCPUs(t)
			t.Run("tool with a child", func(t *testing.T) { stopsUnderLoad(t, "") })
			t.Run("tool ignoring TERM", func(t *testing.T) { stopsUnderLoad(t, "trap '' TERM\n") })
		}},
		{"a check that runs past the fast total", func(t *testing.T) {
			t.Parallel()
			out, code := slowCheck(t, "exec /bin/sleep 30", fastTotal(1000))
			sees(t, out, code, 1, "The fast checks ran past their 1s total. Slowest: slow 1")
		}},
		{"a test runner that runs past the test stage's total", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", localCheck("suite", passing))
			write(t, repo, map[string]string{"checks/fitness-check-suite": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"suite\", \"stage\": \"tests\"}'; exit 0; }\nexec /bin/sleep 30\n"})
			out, code := fitness(t, repo, []string{testsTotal(1000)}, "--policy=external", "--checks=suite,markdown-links")
			sees(t, out, code, 1, "Stopped at the 1s tests total", "The test runners ran past their 1s total. Slowest: suite 1")
		}},
		{"four Go modules whose tests each take six seconds", func(t *testing.T) {
			t.Parallel()
			// A fresh constant in each module keeps go test from replaying a cached
			// result, so every module's test sleeps.
			fresh := strconv.FormatInt(time.Now().UnixNano(), 10)
			files := map[string]string{}
			for _, dir := range []string{"a/", "b/", "c/", "d/"} {
				files[dir+"go.mod"] = "module example.com/m\n\ngo 1.22\n"
				files[dir+"m_test.go"] = "package m\n\nimport (\n\t\"testing\"\n\t\"time\"\n)\n\nconst fresh = " + fresh +
					"\n\nfunc TestSlow(t *testing.T) { time.Sleep(" + strconv.FormatInt(int64(moduleSleep), 10) + ") }\n"
			}
			start := time.Now()
			out, code := goTest(t, files)
			took := time.Since(start)
			sees(t, out, code, 0, "All 1 checks passed · 4 files scanned")
			if oneByOne := 4 * moduleSleep; took >= oneByOne {
				t.Errorf("go-test took %s on four modules whose tests sleep %s each; one module at a time takes %s", took, moduleSleep, oneByOne)
			}
		}},
		{"a pull request whose two scans each run a check for most of the fast total", func(t *testing.T) {
			t.Parallel()
			pause := "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"pause\"}'; exit 0; }\nsleep 5\necho '" + passing + "'\n"
			repo := prBase(t, with(localCheck("pause", passing), map[string]string{"checks/fitness-check-pause": pause}))
			commit(t, repo, noteGuide(t), "docs(changelog): note the guide")
			out, code := fitness(t, repo, append([]string{fastTotal(8000)}, prEnv...), "--policy=external", "--checks=pause")
			sees(t, out, code, 1, "Stage 1 of 2:", "Stage 2 of 2:", "Stopped at the 8s fast total", "The fast checks ran past their 8s total. Slowest: pause 8s")
		}},
		{"a pull request whose test runner runs past the fast total, beside fast checks that pass", func(t *testing.T) {
			t.Parallel()
			files := localCheck("suite", passing)
			files["checks/fitness-check-suite"] = "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"suite\", \"stage\": \"tests\"}'; exit 0; }\nsleep 4\necho '" + passing + "'\n"
			for _, name := range fastNames {
				files["checks/fitness-check-"+name] = localCheck(name, passing)["checks/fitness-check-"+name]
			}
			repo := prBase(t, files)
			commit(t, repo, noteGuide(t), "docs(changelog): note the guide")
			selected := "--checks=suite," + strings.Join(fastNames, ",")
			out, code := fitness(t, repo, append([]string{fastTotal(3000)}, prEnv...), "--policy=external", selected)
			sees(t, out, code, 0, "Stage 1 of 2:", "Stage 2 of 2:", "All 7 checks passed")
			if strings.Contains(out, "Stopped at") {
				t.Errorf("the wait on the test runner must spend none of the fast total:\n%s", out)
			}
		}},
		{"more checks that run past the fast total than the run starts at once", func(t *testing.T) {
			t.Parallel()
			files := localCheck("late", passing)
			var names []string
			for i := range runtime.NumCPU() {
				name := "hang" + strconv.Itoa(i)
				names = append(names, name)
				files["checks/fitness-check-"+name] = "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"" + name + "\"}'; exit 0; }\nexec /bin/sleep 30\n"
			}
			repo := example(t, "happyRepo", files)
			out, code := fitness(t, repo, []string{fastTotal(1000)}, "--policy=external", "--checks="+strings.Join(append(names, "late"), ","))
			sees(t, out, code, 1, "Stopped at the 1s fast total", "not run: the 1s fast total ran out before it started")
			if late := row(out, "late"); !strings.Contains(late, "│ info ") {
				t.Errorf("a check the fast total never let start must not fail, got row %q:\n%s", late, out)
			}
		}},
	}.run(t)
}

func Test0003_5(t *testing.T) {
	cases{
		{"a finding that names a file and line", func(t *testing.T) {
			t.Parallel()
			out, code, _ := inActions(t, example(t, "happyRepo", misspelled(1)))
			sees(t, out, code, 1, "::error file=README.md,line=9,col=5,title=cspell (unknown-word)::Unknown word (zorbleflux)")
		}},
		{"more findings than the annotation cap", func(t *testing.T) {
			t.Parallel()
			out, code, summary := inActions(t, example(t, "happyRepo", misspelled(13)))
			annotated := strings.Count(out, "\n::error ")
			findings := strings.Count(summary, "\n- ")
			want := fmt.Sprintf("::warning::%d more issue(s) not annotated; see the job summary", findings-annotated)
			if annotated != 10 {
				t.Errorf("want 10 annotations at the cap, got %d", annotated)
			}
			sees(t, out, code, 1, want)
		}},
		{"a run where every check passed", func(t *testing.T) {
			t.Parallel()
			out, code, _ := inActions(t, example(t, "happyRepo", nil))
			if strings.Contains(out, "::error") || strings.Contains(out, "::warning") || code != 0 {
				t.Errorf("a green run must add no annotations, got exit %d:\n%s", code, out)
			}
		}},
		{"a finding in a file whose name has a percent sign, with its fix on a second line", func(t *testing.T) {
			t.Parallel()
			out, code, _ := inActions(t, example(t, "happyRepo", map[string]string{"docs/notes-100%.md": "The zorbleflux sits here.\n"}))
			sees(t, out, code, 1, "::error file=docs/notes-100%25.md,line=1,col=5,title=cspell (unknown-word)::Unknown word (zorbleflux)%0AFix: ")
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "::") && strings.Contains(line, "100%") && !strings.Contains(line, "100%25") {
					t.Errorf("a percent sign must arrive escaped in one annotation, got %q", line)
				}
			}
		}},
		{"a run with failing checks", func(t *testing.T) {
			t.Parallel()
			_, code, summary := inActions(t, example(t, "happyRepo", misspelled(1)))
			sees(t, summary, code, 1, "## Fitness checks", "| cspell | ❌ fail |", "README.md:9:5 - Unknown word (zorbleflux)", "| prose-budget | ✅ pass |")
			for _, check := range strings.Fields(catalogRows) {
				if !strings.Contains(summary, "| "+check+" |") {
					t.Errorf("the job summary leaves out installed check %s:\n%s", check, summary)
				}
			}
		}},
		{"a run that fails before any check starts", func(t *testing.T) {
			t.Parallel()
			out, code, _ := inActions(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": "{\n"}))
			sees(t, out, code, 1, "::error::fitness setup: .fitnessrc.json:")
			if strings.Contains(out, "::error file=") {
				t.Errorf("a setup failure must not point at a file line:\n%s", out)
			}
		}},
		{"a job summary that earlier steps filled near its limit", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", misspelled(1))
			summary := filepath.Join(t.TempDir(), "summary.md")
			mustDo(t, os.WriteFile(summary, []byte(strings.Repeat("Earlier step output.\n", 52000)), 0o644))
			_, code := fitness(t, repo, []string{"GITHUB_ACTIONS=true", "GITHUB_WORKSPACE=" + repo, "GITHUB_STEP_SUMMARY=" + summary, "RUNNER_TEMP=" + t.TempDir()})
			written, _ := os.ReadFile(summary)
			sees(t, string(written), code, 1, "Summary truncated.", "Complete report saved on the runner at")
			complete, err := os.ReadFile(summary + ".fitness-report.md")
			if err != nil || !strings.Contains(string(complete), "Unknown word (zorbleflux)") {
				t.Errorf("the complete report must hold every finding: %v", err)
			}
			if len(written) > 1<<20 {
				t.Errorf("summary is %d bytes, over the 1 MiB limit", len(written))
			}
		}},
		{"a run inside a GitHub Actions job", func(t *testing.T) {
			t.Parallel()
			out, _, summary := inActions(t, example(t, "happyRepo", nil))
			sees(t, summary, 0, 0, "### Check timing", "| Check | Stage | Time |")
			times := timings(summary)
			if want := strings.Count(out, "  → "); len(times) != want {
				t.Errorf("timing table has %d checks, the run started %d:\n%s", len(times), want, summary)
			}
			if !slices.IsSortedFunc(times, func(a, b int64) int { return cmp.Compare(b, a) }) {
				t.Errorf("timing must run slowest first, got %v", times)
			}
		}},
		{"a repo with a misspelled word", func(t *testing.T) {
			t.Parallel()
			out, code, summary := inActions(t, example(t, "happyRepo", misspelled(1)))
			sees(t, out, code, 1, "::error file=README.md,line=9,col=5,title=cspell (unknown-word)::Unknown word (zorbleflux)"+
				"%0AFix: correct the spelling, or add the word to the words list in cspell.json%0AWhy: fitness explain cspell")
			sees(t, summary, code, 1, "- README.md:9:5 - Unknown word (zorbleflux)\n"+
				"  - Fix: correct the spelling, or add the word to the words list in cspell.json\n"+
				"  - Why: fitness explain cspell\n")
		}},
	}.run(t)
}

func Test0003_6(t *testing.T) {
	cases{
		{"a repo with a misspelled word and a broken link", func(t *testing.T) {
			t.Parallel()
			files := misspelled(1)
			files["README.md"] += "\nSee [the guide](missing.md).\n"
			run := asJSON(t, example(t, "happyRepo", files))
			if run.Ok {
				t.Error("a run with findings must report ok false")
			}
			// The misspelled word sits on line 9, as Test0003_5 annotates it, and
			// the link two lines below it.
			oneFinding(t, run, "cspell", "README.md", 9, "unknown-word")
			oneFinding(t, run, "markdown-links", "README.md", 11, "broken-relative-link")
		}},
		{"a repo whose commit message breaks a check that reports no findings", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			commit(t, repo, map[string]string{"notes.txt": "notes\n"}, "updated stuff")
			got := findingsOf(t, asJSON(t, repo), "semantic-commit")
			if len(got) == 0 {
				t.Fatal("semantic-commit must report its error lines as findings")
			}
			want := finding{Rule: "semantic-commit", Explain: "fitness explain semantic-commit"}
			for _, f := range got {
				if (finding{Rule: f.Rule, Explain: f.Explain}) != want || f.Message == "" {
					t.Errorf("a finding from an error line must name the check as its rule and its explain command, got %+v", f)
				}
			}
		}},
		{"a repo with passing and failing checks", func(t *testing.T) {
			t.Parallel()
			out, report := withReport(t, example(t, "happyRepo", misspelled(1)))
			if report.RunnerBytes <= 0 || report.ElapsedMs <= 0 {
				t.Errorf("the report must give the runner's size and the run's time, got %+v", report)
			}
			for _, c := range report.Checks {
				fullEntry(t, out, c)
			}
			verdicts(t, report)
		}},
		{"a Go module and a started check that runs past the fast total", func(t *testing.T) {
			t.Parallel()
			// Prettier is the one fast check, so it starts on a loaded machine. A Go
			// module gives go-test something to judge, so it runs rather than skips.
			files := hungTool("")
			files["go.mod"] = "module example.com/app\n\ngo 1.22\n"
			_, report := selectedReport(t, example(t, "happyRepo", files), []string{fastTotal(300)}, "--checks=prettier,go-test")
			prettier := reported(t, report, "prettier")
			if got := [3]any{prettier.TimedOut, prettier.Ok, prettier.Stage}; got != [3]any{true, false, "fast"} {
				t.Errorf("prettier must be stopped at the 300ms fast total, got %+v", prettier)
			}
			if goTest := reported(t, report, "go-test"); [2]any{goTest.TimedOut, goTest.Stage} != [2]any{false, "tests"} {
				t.Errorf("the fast total must never stop a test-stage check, got %+v", goTest)
			}
		}},
		{"a `--json` report from a run with passing and failing checks", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", misspelled(2))
			_, report := withReport(t, repo)
			page := drawn(t, repo, report)
			if outsideRequest.MatchString(page) {
				t.Error("the page must be self-contained, with no scripts or outside requests")
			}
			views := strings.Split(page, `<section class="view view-`)[1:]
			if len(views) != len(diagramViews) {
				t.Fatalf("the page must offer %v views, got %d", diagramViews, len(views))
			}
			for i, view := range views {
				oneBoxEach(t, diagramViews[i], view, report)
			}
			if !strings.Contains(views[2], `title="cspell: 2 error lines, failed"`) {
				t.Error("the error-lines view must size cspell by its 2 error lines and mark it failed")
			}
		}},
		{"a draft text", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "blob.json")
			out, code := piped(t, example(t, "happyRepo", nil), "Some **bold** text.\n", nil, "--blob", "-", "--json="+path)
			sees(t, out, code, 1, "disallowed **bold**")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("fitness --blob --json wrote no report: %v\n%s", err, out)
			}
			var report perfReport
			mustDo(t, json.Unmarshal(raw, &report))
			if reported(t, report, "markdown-no-bold-italic").Ok {
				t.Errorf("the report must mark markdown-no-bold-italic failed:\n%s", raw)
			}
			for _, c := range report.Checks {
				if c.Name == "duplicate-code" {
					t.Errorf("the report lists duplicate-code, which needs files and never judged the text:\n%s", raw)
				}
			}
		}},
		{"a performance record that `fitness --json` kept", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", localCheck("pause", passing))
			write(t, repo, map[string]string{"checks/fitness-check-pause": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"pause\"}'; exit 0; }\nsleep 2\necho '" + passing + "'\n"})
			record := filepath.Join(t.TempDir(), "fitness.json")
			out, code := fitness(t, repo, nil, "--policy=external", "--checks=pause,markdown-links", "--json="+record)
			sees(t, out, code, 0)
			out, code = fitness(t, repo, nil, "perf", record)
			sees(t, out, code, 0, "CHECK  STAGE  TIME", "The run took")
			var names []string
			for _, line := range strings.Split(out, "\n") {
				if m := perfRow.FindStringSubmatch(line); m != nil {
					names = append(names, m[1])
				}
			}
			if strings.Join(names, " ") != "pause markdown-links" {
				t.Errorf("fitness perf must list each check slowest first, got %v:\n%s", names, out)
			}
		}},
		{"a repo whose checks pass", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", localCheck("pause", passing))
			write(t, repo, map[string]string{"checks/fitness-check-pause": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"pause\"}'; exit 0; }\nsleep 2\necho '" + passing + "'\n"})
			out, code := fitness(t, repo, nil, "--policy=external", "--checks=pause,markdown-links")
			sees(t, out, code, 0, "Slowest: pause 2")
		}},
	}.run(t)
}

func Test0003_7(t *testing.T) {
	cases{
		{"a repo with fitness installed", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			out, code := fitness(t, repo, nil, "list")
			sees(t, out, code, 0, "CHECK SCOPE STAGE ENABLED SUMMARY", "Lists Go files that gofmt would change.")
			for _, check := range strings.Fields(catalogRows) {
				if !described(listRow(out, check)) {
					t.Errorf("fitness list shows %s without a scope, stage and summary: %q", check, listRow(out, check))
				}
				if strings.Count(out, "\n"+check+" ") != 1 {
					t.Errorf("fitness list shows %s other than once:\n%s", check, out)
				}
			}
		}},
		{"a repo whose config disables a check under external policy", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"policy": "external", "disabledChecks": ["gofmt"]}` + "\n"})
			checks := listJSON(t, repo)
			if len(checks) != len(strings.Fields(catalogRows)) {
				t.Fatalf("fitness list --json gave %d checks, want %d", len(checks), len(strings.Fields(catalogRows)))
			}
			for _, c := range checks {
				if c.Enabled == (c.Name == "gofmt") {
					t.Errorf("%s reads as enabled=%v", c.Name, c.Enabled)
				}
			}
		}},
		{"a repo with a local check", func(t *testing.T) {
			t.Parallel()
			files := localCheck("house-rule", passing)
			files["checks/fitness-check-house-rule"] = "#!/bin/sh\nif [ \"$1\" = --describe ]; then\n  echo '{\"name\": \"house-rule\", \"summary\": \"Keeps the house rule.\", \"scope\": \"files\"}'\n  exit 0\nfi\necho '" + passing + "'\n"
			out, code := fitness(t, example(t, "happyRepo", files), nil, "list")
			sees(t, out, code, 0, "Keeps the house rule.")
			if fields := strings.Fields(listRow(out, "house-rule")); len(fields) < 4 || fields[1] != "files" || fields[3] != "yes" {
				t.Errorf("fitness list shows the local check as %q, want scope files and enabled", listRow(out, "house-rule"))
			}
		}},
		{"a repo with a local check whose describe handshake fails", func(t *testing.T) {
			t.Parallel()
			files := localCheck("house-rule", passing)
			files["checks/fitness-check-house-rule"] = "#!/bin/sh\nif [ \"$1\" = --describe ]; then\n  exit 1\nfi\necho '" + passing + "'\n"
			out, code := fitness(t, example(t, "happyRepo", files), nil, "list")
			sees(t, out, code, 0, "Lists Go files that gofmt would change.")
			if row := listRow(out, "house-rule"); !strings.Contains(row, "(--describe gave no metadata") {
				t.Errorf("fitness list shows the broken local check as %q, want a note saying why", row)
			}
		}},
		{"a repo whose own Go check build removes the runner's file", func(t *testing.T) {
			t.Parallel()
			out, code := runnerGoneAtBuild(t)
			sees(t, out, code, 0, "fitness: building fitness/go-rule with go", "Lists Go files that gofmt would change.")
			if fields := strings.Fields(listRow(out, "gofmt")); len(fields) < 2 || fields[1] != "files" {
				t.Errorf("fitness list shows gofmt as %q, want the files scope its metadata declares", listRow(out, "gofmt"))
			}
			if strings.Contains(out, "--describe gave no metadata") {
				t.Errorf("built-in checks must list their metadata without starting a process, got:\n%s", out)
			}
		}},
		{"a repo with fitness installed: each prints the check's name, scope, stage and documentation", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			for _, check := range listJSON(t, repo) {
				out, code := fitness(t, repo, nil, "explain", check.Name)
				sees(t, strings.ReplaceAll(out, `\`, ""), code, 0, "# "+check.Name, "Scope: "+check.Scope+" · Stage: "+check.Stage, opening(check.Summary))
			}
		}},
		{"wants a check's documentation as data, the engine's own check from `go/docs/runner/local-branch.md`", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			got := explainJSON(t, repo, "local-branch")
			summary := listedSummary(t, repo, "local-branch")
			want := explainedCheck{Name: "local-branch", Summary: summary, Scope: got.Scope, Body: got.Body}
			if got != want || got.Scope == "" {
				t.Errorf("fitness explain --json gave name %q, scope %q, summary %q; fitness list says %q", got.Name, got.Scope, got.Summary, summary)
			}
			if "# local-branch\n\n"+got.Body != repoFile(t, "go/docs/runner/local-branch.md") || !strings.Contains(got.Body, "## Usage") {
				t.Errorf("the body is not go/docs/runner/local-branch.md after its title, with its usage:\n%s", got.Body)
			}
		}},
		{"mistypes a check name", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "explain", "prose-budgets")
			sees(t, out, code, 2, `no check named "prose-budgets"; did you mean prose-budget?`)
		}},
		{"a repo with a local check whose describe handshake fails: it prints a note saying why the check gave no metadata", func(t *testing.T) {
			t.Parallel()
			files := localCheck("house-rule", passing)
			files["checks/fitness-check-house-rule"] = "#!/bin/sh\nif [ \"$1\" = --describe ]; then\n  exit 1\nfi\necho '" + passing + "'\n"
			out, code := fitness(t, example(t, "happyRepo", files), nil, "explain", "house-rule")
			sees(t, out, code, 0, "# house-rule", "Note: --describe gave no metadata")
		}},
		{"a repo with fitness installed: each topic prints its Markdown file in `go/docs`, and the list shows the sentence it opens with", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			out, code := fitness(t, repo, nil, "docs")
			sees(t, out, code, 0, "protocol", "config")
			names := topicNames(out)
			for _, want := range []string{"config", "coverage", "github", "hooks", "languages", "local-checks", "protocol"} {
				if !slices.Contains(names, want) {
					t.Fatalf("fitness docs lists %q, not topic %q:\n%s", names, want, out)
				}
			}
			for _, name := range names {
				printsTopicFile(t, repo, out, name)
			}
		}},
		{"asks for a topic that does not exist", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "docs", "nope")
			sees(t, out, code, 2, `no topic named "nope"; the topics are:`, "protocol", "config")
		}},
		{"passes `docs` a flag, as `list` and `explain` take `--json`", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "docs", "--json")
			sees(t, out, code, 2, "name one topic, as `fitness docs [topic]`")
		}},
		{"a repo with fitness installed: the page, which llms.txt copies from `go/docs/runner/fitness.md`, explains fitness, its exit codes, every subcommand, the check the engine runs itself, where every installable check is listed, and every docs topic", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			page, code := fitness(t, repo, nil, "llms")
			sees(t, page, code, 0, "# fitness", "## Exit codes", "## Commands", "## Checks", "## Topics", "(https://github.com/may-journal/fitness-checks/tree/main/cmd)")
			if source := repoFile(t, "go/docs/runner/fitness.md"); !strings.HasSuffix("\n"+page, "\n"+source) || repoFile(t, "llms.txt") != source {
				t.Errorf("fitness llms and llms.txt do not both print go/docs/runner/fitness.md:\n%s", page)
			}
			for _, name := range subcommands {
				sees(t, page, code, 0, "`fitness "+name)
			}
			sees(t, page, code, 0, "- `local-branch`: ")
			topics, _ := fitness(t, repo, nil, "docs")
			for _, name := range topicNames(topics) {
				sees(t, page, code, 0, "`fitness docs "+name+"`")
			}
		}},
		{"asks for help", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "--help")
			sees(t, out, code, 0, "Run `fitness llms`")
			for _, name := range subcommands {
				sees(t, out, code, 0, "  fitness "+name)
			}
		}},
		{"looks for a check's README in the fitness repo", func(t *testing.T) {
			t.Parallel()
			readmes, err := filepath.Glob(filepath.Join("..", "cmd", "fitness-check-*", "README.md"))
			mustDo(t, err)
			if len(readmes) > 0 {
				t.Errorf("check READMEs remain in this repo: %v", readmes)
			}
			if !strings.Contains(repoFile(t, "docs/checks.md"), "https://github.com/may-journal/fitness-checks") {
				t.Error("docs/checks.md does not link fitness-checks, where each check's README lives")
			}
		}},
		{"reads the command list in the README", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			readme := repoFile(t, "README.md")
			_, usage, _ := strings.Cut(readme, "## Usage\n")
			_, list, _ := strings.Cut(usage, "```text\n")
			list, _, _ = strings.Cut(list, "```")
			if strings.Count(list, "  fitness ") < 10 {
				t.Fatalf("the README's Usage section has no command list:\n%s", usage)
			}
			out, code := fitness(t, repo, nil, "--help")
			if code != 0 || !strings.Contains(out, "\n"+list+"\n") {
				t.Errorf("fitness --help (exit %d) does not print the README's command list:\n%s\nREADME:\n%s", code, out, list)
			}
		}},
		{"a fitness checkout: rebuilding llms.txt and every generated region from the code and `go/docs` changes no committed file", func(t *testing.T) {
			t.Parallel()
			out, code := rebuildDocs(t, filepath.Join("..", ".."))
			sees(t, out, code, 0, `"ok":true`)
		}},
		{"a fitness checkout whose llms.txt was edited by hand and whose topic summary changed: the rebuild names each file that drifted", func(t *testing.T) {
			t.Parallel()
			root := docsSources(t)
			write(t, root, map[string]string{
				"llms.txt":         repoFile(t, "llms.txt") + "\nA line edited by hand.\n",
				"go/docs/hooks.md": strings.Replace(repoFile(t, "go/docs/hooks.md"), "installs the canonical", "installs the shared", 1),
			})
			out, code := rebuildDocs(t, root)
			sees(t, out, code, 1, "llms.txt: stale or edited by hand", "go/docs/runner/fitness.md: stale or edited by hand")
		}},
		{"a fitness checkout: every subcommand `fitness --help` lists and every key of `.fitnessrc.json` has a section in the Markdown people edit", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "--help")
			commands := unnamed(editedMarkdown(t), "`fitness ", helpCommands(out))
			keys := unnamed(withoutRegions(repoFile(t, "go/docs/config.md")), "- `", readmeConfigKeys(t))
			if code != 0 || len(commands) > 0 || len(keys) > 0 {
				t.Errorf("fitness --help exit %d; no section in README.md, docs or go/docs names these subcommands %q, and go/docs/config.md lists no keys %q", code, commands, keys)
			}
		}},
		{"a fitness checkout: every exported name of go/pkg/checkkit has a heading in go/docs/protocol.md", func(t *testing.T) {
			t.Parallel()
			if missing := undocumentedAPI(t); len(missing) > 0 {
				t.Errorf("go/docs/protocol.md has no heading for these go/pkg/checkkit names: %q", missing)
			}
		}},
	}.run(t)
}

func Test0003_8(t *testing.T) {
	cases{
		{"Go, Swift, TypeScript, JavaScript, Java, Kotlin and Rust files with `//` inside a string, template or raw string", func(t *testing.T) {
			t.Parallel()
			out, code := listComments(t, map[string]string{
				"go/app/main.go": "package main\n\n// Main prints a link.\nfunc main() {\n\tprintln(\"see https://example.com // inside a string /* too */\")\n\tprintln(" +
					backtick + "// inside a string" + backtick + ")\n}\n",
				"swift/App.swift": `let url = "https://example.com // inside a string"
let raw = #"// inside a string"#
let mixed = "\("// inside a string")" // after a string
`,
				"ts/app.ts": "const url = \"https://example.com // inside a string\";\nconst text = " +
					backtick + "/* inside a string */ \\" + backtick + " // inside a string ${url}" + backtick + "; // after a template\n",
				"js/app.js": `const quote = 'it\'s // inside a string';
// Prints the quote.
console.log(quote);
`,
				"java/App.java": `class App {
  String url = "http://example.com // inside a string";
  char slash = '/'; // one slash
}
`,
				"kotlin/App.kt": `val text = """
  // inside a string
"""
/* outer /* nested */ still outer */
`,
				"rust/main.rs": `fn first<'a>(s: &'a str) -> &'a str { s } // takes a lifetime
fn main() { let u = "http://x // inside a string"; }
let r = r#"a "q" // inside a string"#; // after a raw string
`,
			})
			listsExactly(t, out, code,
				"go/app/main.go:3: go doc: Main prints a link.",
				"java/App.java:3: java inline: one slash",
				"js/app.js:2: javascript doc: Prints the quote.",
				"kotlin/App.kt:4: kotlin doc: outer /* nested */ still outer",
				"rust/main.rs:1: rust inline: takes a lifetime",
				"rust/main.rs:3: rust inline: after a raw string",
				"swift/App.swift:3: swift inline: after a string",
				"ts/app.ts:2: typescript inline: after a template",
			)
		}},
		{"block, doc and inline comments in Go, Swift and TypeScript", func(t *testing.T) {
			t.Parallel()
			out, code := listComments(t, map[string]string{
				"go/doc.go": `// Package doc holds docs.
// It spans two lines.
package doc

/*
Block explains
the next value.
*/
var x = 1 // trailing note
`,
				"swift/Doc.swift": `import Foundation

/// Greets a person.
/// - Parameter name: who.
func greet(name: String) {}

/**
 * A block doc.
 */
let y = 2 /* inline block */
`,
				"ts/doc.ts": `/**
 * Adds two numbers.
 */
export function add(a: number, b: number): number {
  return a + b; // the sum
}
`,
			})
			listsExactly(t, out, code,
				"go/doc.go:1: go doc: Package doc holds docs. It spans two lines.",
				"go/doc.go:5: go doc: Block explains the next value.",
				"go/doc.go:9: go inline: trailing note",
				"swift/Doc.swift:3: swift doc: Greets a person. - Parameter name: who.",
				"swift/Doc.swift:7: swift doc: A block doc.",
				"swift/Doc.swift:10: swift inline: inline block",
				"ts/doc.ts:1: typescript doc: Adds two numbers.",
				"ts/doc.ts:5: typescript inline: the sum",
			)
		}},
		{"shell scripts, one without an extension, and YAML files with `#` comments and a quoted `#`", func(t *testing.T) {
			t.Parallel()
			out, code := listComments(t, map[string]string{
				"scripts/run.sh": `#!/bin/sh
# Runs the build.
echo "a # inside quotes" '# inside quotes' # after echo
cat <<EOF
# inside a here-document
EOF
size=${#HOME} # length
n=$((1 << SHIFT)) # shifted
# after the shift
`,
				"bin/deploy": "#!/usr/bin/env bash\n# Deploys the site.\n",
				"config/app.yml": `# App settings.
name: "a # inside quotes" # the name
url: http://example.com/#anchor
note: |
  # inside a block
list:
  - 'b # inside quotes'
`,
			})
			listsExactly(t, out, code,
				"bin/deploy:2: shell doc: Deploys the site.",
				"config/app.yml:1: yaml doc: App settings.",
				"config/app.yml:2: yaml inline: the name",
				"scripts/run.sh:2: shell doc: Runs the build.",
				"scripts/run.sh:3: shell inline: after echo",
				"scripts/run.sh:7: shell inline: length",
				"scripts/run.sh:8: shell inline: shifted",
				"scripts/run.sh:9: shell doc: after the shift",
			)
		}},
		{"tool directives, a license header and a generated file", func(t *testing.T) {
			t.Parallel()
			out, code := listComments(t, map[string]string{
				".gitattributes": "go/gen.go linguist-generated\n",
				"go/gen.go":      "// Package gen is written by a tool.\npackage gen\n",
				"go/tool.go": `// Copyright 2026 Example. Licensed under MIT.

//go:build linux

// Package tool keeps a note.
package tool

// cspell:ignore zorp
var zorp = 1
`,
				"swift/Lint.swift": "// swiftlint:disable force_cast\nlet a = 1 // keeps this one\n",
				"scripts/lint.sh":  "#!/bin/bash\n# shellcheck disable=SC2086\necho $1 # keeps this one\n",
				"ts/lint.ts":       "/* eslint-disable no-console */\nconsole.log(1); // keeps this one\n",
			})
			listsExactly(t, out, code,
				"go/tool.go:5: go doc: Package tool keeps a note.",
				"scripts/lint.sh:3: shell inline: keeps this one",
				"swift/Lint.swift:2: swift inline: keeps this one",
				"ts/lint.ts:2: typescript inline: keeps this one",
			)
		}},
		{"names a folder and the absolute path of a file", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{
				"go/a/a.go":   "// Package a is in the folder.\npackage a\n",
				"go/b/b.go":   "// Package b is in the folder too.\npackage b\n",
				"ts/named.ts": "// Named by its absolute path.\nexport {};\n",
				"ts/other.ts": "// Named by no path.\nexport {};\n",
			})
			out, code := fitness(t, repo, nil, "comments", "go", filepath.Join(repo, "ts", "named.ts"))
			listsExactly(t, out, code,
				"go/a/a.go:1: go doc: Package a is in the folder.",
				"go/b/b.go:1: go doc: Package b is in the folder too.",
				"ts/named.ts:1: typescript doc: Named by its absolute path.",
			)
		}},
		{"comments that only start with a tool name, such as `shellCheck is` and `SwiftLint is`", func(t *testing.T) {
			t.Parallel()
			out, code := listComments(t, map[string]string{
				"scripts/prose.sh":  "#!/bin/sh\n# shellCheck is a repo.\n# It keeps its second line.\n# shellcheck disable=SC2086\n# shellcheck  source=lib.sh\necho $1\n",
				"go/spell.go":       "// cSpell:ignore zorp\npackage spell\n\n// Spell keeps a note.\nvar zorp = 1\n",
				"swift/Prose.swift": "// SwiftLint is a linter.\nlet a = 1\n// swiftlint:disable force_cast\nlet b = 2\n",
			})
			listsExactly(t, out, code,
				"go/spell.go:4: go doc: Spell keeps a note.",
				"scripts/prose.sh:2: shell doc: shellCheck is a repo. It keeps its second line.",
				"swift/Prose.swift:1: swift doc: SwiftLint is a linter.",
			)
		}},
		{"comments out an import from a dist folder in a TypeScript file", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{
				".gitignore": buildOutputIgnore,
				"src/app.ts": "import ok from './local.js';\n// import bad from '../dist/foo.js';\n" +
					"/*\nconst z = require('pkg/dist/z.js');\n*/\nexport const app = ok;\n",
			})
			out, code := withPartner(t, repo, buildOutputCheck)
			sees(t, out, code, 0, "All 2 checks passed")
		}},
		{"sets Vitest thresholds of 90 below a commented-out line setting them to 100", func(t *testing.T) {
			t.Parallel()
			rp := standIn(t, map[string][]response{"vitest": {vcfFullRun}})
			config := strings.Replace(vcfConfig("90"), "      thresholds:",
				"      // thresholds: { branches: 100, functions: 100, lines: 100, statements: 100 },\n      thresholds:", 1)
			out, code := vcfRun(t, vcfRepo(t, map[string]string{"vitest.config.mjs": config}), rp.env)
			sees(t, out, code, 1, "Vitest coverage thresholds must be 100 for branches, functions, lines, and statements.")
		}},
		{"comments out the release a pom.xml pins and the line of a .java-version file", func(t *testing.T) {
			t.Parallel()
			out, code := javaVersionRun(t, map[string]string{
				"pom.xml":       "<project>\n  <!-- " + strings.TrimSpace(javaVersionRelease("25")) + " -->\n</project>\n",
				".java-version": "# 17\n",
			})
			sees(t, out, code, 1, "no JDK version is pinned; add a .java-version file naming the release")
		}},
		{"a Go, Swift and TypeScript comment with a sentence past its limit", func(t *testing.T) {
			t.Parallel()
			out, code := withPartner(t, example(t, "happyRepo", inComments(strings.TrimSpace(longSentence))), "prose-budget")
			sees(t, out, code, 1, atEachFile("a sentence has 30 words (max 23)")...)
		}},
		{"wraps code duplicated in two files in duplicate-code ignore markers", func(t *testing.T) {
			t.Parallel()
			marked := "// duplicate-code:ignore-start\n" + dupClone + "// duplicate-code:ignore-end\n"
			repo := example(t, "happyRepo", map[string]string{
				"src/a.js": marked + dupBlock("fa", 24, 3),
				"src/b.js": marked + dupBlock("fb", 24, 3),
			})
			out, code := withPartner(t, repo, "duplicate-code")
			sees(t, out, code, 0, "All 2 checks passed")
		}},
		{"a workflow job with no summary step and a cspell directive after `jobs:`", func(t *testing.T) {
			t.Parallel()
			workflow := "name: CI\non: push\njobs: # cspell:ignore foo\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: make test # cspell:disable-line\n"
			out, code := withPartner(t, example(t, "happyRepo", map[string]string{".github/workflows/ci.yml": workflow}), "workflow-reporting")
			sees(t, out, code, 1, ".github/workflows/ci.yml:4: job test has no step")
		}},
	}.run(t)
}
