// cspell:ignore Typeflag

package requirements

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test0002_1(t *testing.T) {
	cases{
		{"a repo whose `.fitnessrc.json` sets no `checkVersions`, or an empty one: it prints each check with its pin", func(t *testing.T) {
			t.Parallel()
			for _, config := range []string{"{}", `{"checkVersions": {}}`} {
				out, code := fitness(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": config}), nil, "check-versions")
				sees(t, out, code, 0, "markdown-links v1\n", "gofmt v1\n")
				if strings.Contains(out, "local-branch") {
					t.Errorf("the engine judges local-branch, so no install pins it:\n%s", out)
				}
			}
		}},
		{"a repo whose `checkVersions` pins one check to a release and one to a major", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"markdown-links": "v1.0.0", "gofmt": "v1"}}`})
			out, code := fitness(t, repo, nil, "check-versions")
			sees(t, out, code, 0, "gofmt v1\nmarkdown-links v1.0.0\n")
			if lines := strings.Count(strings.TrimSpace(pinLinesPins(out)), "\n") + 1; lines != 2 {
				t.Errorf("want exactly the two pinned checks, got %d lines:\n%s", lines, out)
			}
		}},
		{"a repo whose `checkVersions` holds a malformed pin or name, or pins a check the engine runs", func(t *testing.T) {
			t.Parallel()
			for config, want := range map[string]string{
				`{"checkVersions": {"gofmt": "1.0"}}`:       `checkVersions pins gofmt to "1.0"; use a major such as v1 or a release such as v1.2.3`,
				`{"checkVersions": {"local-branch": "v1"}}`: "checkVersions pins local-branch, which runs inside the engine, so remove it",
				`{"checkVersions": {"No Check": "v1"}}`:     "checkVersions names \"No Check\", which is no check name; `fitness list` names them",
			} {
				out, code := fitness(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": config}), nil, "check-versions")
				sees(t, out, code, 1, want)
			}
		}},
		{"a repo whose `checkVersions` pins `*` to a major and one check to a release", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"*": "v1", "gofmt": "v1.0.0"}}`})
			out, code := fitness(t, repo, nil, "check-versions")
			sees(t, out, code, 0, "markdown-links v1\n", "gofmt v1.0.0\n")
			if strings.Contains(out, "* v1") || strings.Contains(out, "local-branch") {
				t.Errorf("a * pin must spread over every released check and print no * line:\n%s", out)
			}
		}},
		{"a repo whose `checkVersions` pins `*`, behind a proxy that never answers, with no recorded release index", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"*": "v1"}}`})
			env := append(fiPrivateCache(t), "HTTPS_PROXY="+silentListener(t), "NO_PROXY=", "no_proxy=")
			out, code := fitness(t, repo, env, "check-versions")
			sees(t, out, code, 1, "pin each check in checkVersions instead")
		}},
	}.run(t)
}

func Test0002_2(t *testing.T) {
	cases{
		{"a repo that pins one check to a release and one to a major", func(t *testing.T) {
			t.Parallel()
			env := tagsInstall(t, "fitness-check-gofmt-v1.0.0", "fitness-check-gofmt-v0.9.0", "fitness-check-gofmt-extra-v1.9.0")
			repo := pinnedInstall(t, `{"markdown-links": "v1.0.0", "gofmt": "v1"}`)
			out, code := fiRun(t, repo, env, "--with-checks", "--install-only")
			sees(t, out, code, 0, "installed 2 checks in ")
			dir := installedInInstall.FindStringSubmatch(out)[2]
			entries, err := os.ReadDir(dir)
			mustDo(t, err)
			names := slices.Collect(func(yield func(string) bool) {
				for _, e := range entries {
					yield(e.Name())
				}
			})
			if !slices.Equal(names, []string{"fitness-check-gofmt", "fitness-check-markdown-links"}) {
				t.Fatalf("the folder must hold exactly the two pinned checks, got %q", names)
			}
			described, code := execute(t, func() *exec.Cmd { return exec.Command(filepath.Join(dir, "fitness-check-gofmt"), "--describe") })
			sees(t, described, code, 0, `"name":"gofmt"`, `"protocol":1`)
			again, code := fiRun(t, repo, env, "--with-checks", "--install-only")
			sees(t, again, code, 0, "installed 2 checks in "+dir)
			if strings.Contains(again, "downloading fitness-check-") || strings.Contains(again, "for the check release") {
				t.Errorf("a second install must reuse the cached checks and lookups:\n%s", again)
			}
		}},
		{"a repo that pins a check to a major with no release", func(t *testing.T) {
			t.Parallel()
			env := tagsInstall(t, "fitness-check-gofmt-v1.0.0")
			repo := pinnedInstall(t, `{"gofmt": "v2"}`)
			out, code := fiRun(t, repo, env, "--with-checks", "--install-only")
			sees(t, out, code, 0, "warning: no release of check gofmt matches v2, so it does not run", "installed 0 checks in ")
			out, code = fiRun(t, repo, env, "--with-checks", "--", "--no-builtins", "gofmt")
			sees(t, out, code, 1, "no release of check gofmt matches v2")
		}},
		{"a repo that pins one check", func(t *testing.T) {
			t.Parallel()
			repo := pinnedInstall(t, `{"markdown-links": "v1.0.0"}`)
			commit(t, repo, map[string]string{"broken.md": "# Broken\n\nSee [the guide](missing.md).\n"}, "docs(broken): link a missing guide")
			// The suite's own cache holds a fresh release index, so a busy network
			// cannot leave the index unanswered within its two seconds.
			env := tagsInstall(t)[1:]
			for _, run := range [][]string{{"--no-builtins"}, nil} {
				out, code := fiRun(t, repo, env, append(append([]string{"--with-checks", "--"}, run...), "--policy=external", "--all", "--checks=markdown-links")...)
				sees(t, out, code, 1, "missing.md", "0 of 1 checks passed")
				if strings.Contains(out, "replaces the built-in") {
					t.Errorf("an installed pinned check replaces its built-in on purpose, so no note:\n%s", out)
				}
			}
		}},
		{"a repo that pins one check, and GitHub refuses the latest release lookup", func(t *testing.T) {
			t.Parallel()
			cache := fiPrivateCache(t)
			stale := fmt.Sprintf(`{"tag":"v%s","checked":%q}`, rlCandidate(t), time.Now().Add(-time.Hour).Format(time.RFC3339))
			mustDo(t, os.WriteFile(filepath.Join(strings.TrimPrefix(cache[0], "FITNESS_CACHE_DIR="), "latest-release.json"), []byte(stale), 0o600))
			env := append(cache, rlReleases(t, http.StatusForbidden, "").env...)
			out, code := fiRun(t, pinnedInstall(t, `{"markdown-links": "v1.0.0"}`), env, "--with-checks", "--version", "latest", "--install-only")
			sees(t, out, code, 1, "GitHub did not answer the release lookup, so no check installs")
			if strings.Contains(out, "fitness-check-markdown-links") {
				t.Errorf("after an unanswered lookup the installer must ask for no check:\n%s", out)
			}
		}},
		{"a repo that pins one uncached check, behind a proxy that never answers", func(t *testing.T) {
			t.Parallel()
			cache := fiPrivateCache(t)
			repo := pinnedInstall(t, `{"markdown-links": "v1.0.0"}`)
			// The engine unpacks into the cache first, so only the check install is timed.
			_, _ = fiRun(t, t.TempDir(), cache, "--install-only")
			env := append(cache, "HTTPS_PROXY="+silentListener(t), "NO_PROXY=", "no_proxy=")
			start := time.Now()
			out, code := fiRun(t, repo, env, "--with-checks", "--install-only")
			sees(t, out, code, 1, "does not download")
			if took := time.Since(start); took > 5*time.Second {
				t.Errorf("behind a silent proxy the check install must give up within 5s, took %s:\n%s", took, out)
			}
			start = time.Now()
			again, _ := fiRun(t, repo, env, "--with-checks", "--install-only")
			if took := time.Since(start); took > 3*time.Second || strings.Contains(again, "for the check release index") {
				t.Errorf("a second install within the minute must not ask the silent index again, took %s:\n%s", took, again)
			}
		}},
		{"a repo pinning no check behind a silent proxy, and one pinning one, with `FITNESS_WITH_CHECKS=pinned` as hooks set it", func(t *testing.T) {
			t.Parallel()
			env := append(tagsInstall(t), "FITNESS_WITH_CHECKS=pinned")
			silent := append(slices.Clone(env), "HTTPS_PROXY="+silentListener(t), "NO_PROXY=", "no_proxy=")
			out, code := fiRun(t, example(t, "happyRepo", nil), silent, "--install-only")
			sees(t, out, code, 0)
			if strings.Contains(out, "asking ") || strings.Contains(out, "installed") {
				t.Errorf("behind a silent proxy, a repo that pins no check must read nothing for checks:\n%s", out)
			}
			out, code = fiRun(t, pinnedInstall(t, `{"markdown-links": "v1.0.0"}`), env, "--install-only")
			sees(t, out, code, 0, "installed 1 checks in ")
		}},
		{"a repo that pins three checks, with files that break two of them", func(t *testing.T) {
			t.Parallel()
			checks := "--checks=markdown-links,markdown-no-bold-italic,markdown-filename-kebab-case"
			repo := pinnedInstall(t, `{"markdown-links": "v1.0.0", "markdown-no-bold-italic": "v1.0.0", "markdown-filename-kebab-case": "v1.0.0"}`)
			commit(t, repo, map[string]string{"guide.md": "# Guide\n\nSee [the steps](missing.md) and **this**.\n"}, "docs(guide): add a guide")
			env := tagsInstall(t)
			builtIn, code := fiRun(t, repo, env, "--", "--policy=external", "--all", checks)
			sees(t, builtIn, code, 1, "missing.md", "disallowed **bold**")
			clean, code := fiRun(t, repo, env, "--with-checks", "--", "--no-builtins", "--policy=external", "--all", checks)
			sees(t, clean, code, 1, "installed 3 checks in ")
			if got, want := resultsClean(clean), resultsClean(builtIn); got != want {
				t.Errorf("the clean install must judge as the built-ins do\nclean:\n%s\nbuilt-in:\n%s", got, want)
			}
		}},
		{"a cached install with a damaged check binary", func(t *testing.T) {
			t.Parallel()
			env := fiPrivateCache(t)
			first, _ := fiRun(t, t.TempDir(), env, "--install-only")
			again, _ := fiRun(t, t.TempDir(), env, "--install-only")
			if fiLastLine(first) != fiLastLine(again) {
				t.Fatalf("an intact cache was not reused: %q then %q", fiLastLine(first), fiLastLine(again))
			}
			write(t, fiLastLine(first), map[string]string{"fitness-check-markdown-filename-kebab-case": "damaged\n"})
			out, code := fitness(t, example(t, "happyRepo", nil), env, "markdown-filename-kebab-case")
			sees(t, out, code, 0, "All 1 checks passed")
		}},
	}.run(t)
}

func Test0002_3(t *testing.T) {
	cases{
		{"check programs in the `FITNESS_CHECKS_DIR` folder, beside fitness and on `PATH`", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			bundle, configured, onPath := bundleDiscovery(t), t.TempDir(), t.TempDir()
			where := func(name, place string) string {
				return scriptDiscovery(`{"name":"`+name+`"}`, name+" ran from "+place)
			}
			write(t, configured, map[string]string{"fitness-check-first": where("first", "the configured folder")})
			write(t, bundle, map[string]string{
				"fitness-check-first":  where("first", "beside fitness"),
				"fitness-check-second": where("second", "beside fitness"),
			})
			write(t, onPath, map[string]string{
				"fitness-check-first":  where("first", "PATH"),
				"fitness-check-second": where("second", "PATH"),
				"fitness-check-third":  where("third", "PATH"),
			})
			env := []string{"FITNESS_CHECKS_DIR=" + configured, "PATH=" + onPath + string(os.PathListSeparator) + os.Getenv("PATH")}
			out, code := runDiscovery(t, bundle, repo, env, "--policy=external", "--checks=first,second,third")
			sees(t, out, code, 1, "first ran from the configured folder", "second ran from beside fitness", "third ran from PATH")
			describe, code := execute(t, func() *exec.Cmd {
				return exec.Command(filepath.Join(installOnly(t), "fitness-check-local-branch"), "--describe")
			})
			sees(t, describe, code, 0, `"name":"local-branch"`, `"protocol":1`)
		}},
		{"an installed check whose `--describe` reports protocol 2, and one that reports none", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			bundle := bundleDiscovery(t)
			write(t, bundle, map[string]string{
				"fitness-check-newer": scriptDiscovery(`{"name":"newer","protocol":2}`, "newer ran"),
				"fitness-check-older": "#!/bin/sh\n[ \"$FITNESS_PROTOCOL\" = 1 ] || exit 2\n[ \"$1\" = --describe ] && { echo '{\"name\":\"older\"}'; exit 0; }\n" +
					"echo '{\"ok\":true,\"filesChecked\":1}'\n",
			})
			out, code := runDiscovery(t, bundle, repo, nil, "--policy=external", "--checks=older,newer")
			sees(t, out, code, 1, "check newer speaks check protocol 2, but this fitness speaks protocol 1; reinstall it: go install github.com/may-journal/fitness-checks/cmd/fitness-check-newer@latest")
			if strings.Contains(out, "newer ran") {
				t.Errorf("a check of another protocol must not run:\n%s", out)
			}
			out, code = runDiscovery(t, bundle, repo, nil, "older")
			sees(t, out, code, 0, "All 1 checks passed")
		}},
		{"one installed check", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			bundle := bundleDiscovery(t)
			write(t, bundle, map[string]string{"fitness-check-only": scriptDiscovery(`{"name":"only"}`, "only ran")})
			out, code := runDiscovery(t, bundle, repo, nil, "--no-builtins", "--policy=external", "--all")
			sees(t, out, code, 1, "only ran", "0 of 1 checks passed")
			for _, selected := range [][]string{{"markdown-links"}, {"check", "markdown-links"}} {
				out, code = runDiscovery(t, bundle, repo, nil, append([]string{"--no-builtins"}, selected...)...)
				sees(t, out, code, 1, "markdown-links is not installed; install it: go install github.com/may-journal/fitness-checks/cmd/fitness-check-markdown-links@latest")
			}
		}},
		{"an installed program for `local-branch`, whose judgment lives in the engine", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			bundle := bundleDiscovery(t)
			write(t, bundle, map[string]string{"fitness-check-local-branch": scriptDiscovery(`{"name":"local-branch"}`, "installed local-branch ran")})
			for _, args := range [][]string{{"local-branch"}, {"--no-builtins", "local-branch"}} {
				out, code := runDiscovery(t, bundle, repo, nil, args...)
				sees(t, out, code, 0, "fitness-check-local-branch declares the local-branch check, which fitness judges itself", "skipped")
				if strings.Contains(out, "installed local-branch ran") {
					t.Errorf("an installed local-branch cannot judge, so fitness must not run it:\n%s", out)
				}
			}
		}},
		{"a Go test that runs fitness, in a repo that `go-test` judges", func(t *testing.T) {
			t.Parallel()
			// The runner stands alone, so only FITNESS_CHECKS_DIR leads it to the checks.
			runner := filepath.Join(bundleDiscovery(t), "fitness")
			checks := goTestOnThisEngine(t)
			files := map[string]string{
				"go.mod":    "module example.com/m\n\ngo 1.22\n",
				"m_test.go": strings.Replace(innerDiscovery, "FRESH", strconv.FormatInt(time.Now().UnixNano(), 10), 1),
			}
			out, code := goTest(t, files, "FITNESS_CHECKS_DIR="+checks, "REQ_FITNESS="+runner)
			sees(t, out, code, 0, "All 1 checks passed")
		}},
		{"lists this module's Go dependencies of every engine file, for any platform, apart from `checks.go`, which links the check", func(t *testing.T) {
			t.Parallel()
			imports, builtIns := engineImportsFree(t)
			if builtIns == 0 {
				t.Fatal("checks.go links no check package, so the test cannot tell it from the engine")
			}
			if len(imports) == 0 {
				return
			}
			out := goListFree(t, append([]string{"-deps", "-f", `{{.ImportPath}}: {{join .Imports " "}}`}, imports...)...)
			for _, line := range strings.Split(out, "\n") {
				pkg, imported, _ := strings.Cut(line, ": ")
				for _, dep := range slices.DeleteFunc(strings.Fields(imported), func(p string) bool { return !checkPackageFree(p) }) {
					t.Errorf("engine package %s imports check package %s", pkg, dep)
				}
			}
		}},
	}.run(t)
}

func Test0002_4(t *testing.T) {
	cases{
		{"a repo that pins one check and breaks it, and a home with no Fitness cache", func(t *testing.T) {
			t.Parallel()
			repo := pinnedInstall(t, `{"markdown-links": "v1.0.0"}`)
			commit(t, repo, map[string]string{"broken.md": "# Broken\n\nSee [the guide](missing.md).\n"}, "docs(broken): link a missing guide")
			engine := filepath.Join(installOnly(t), "fitness")
			env := cleanHomeFetch(t)
			run := func() (string, int) {
				return execute(t, func() *exec.Cmd {
					cmd := exec.Command(engine, "--policy=external", "--all", "--checks=markdown-links")
					cmd.Dir, cmd.Env = repo, env
					return cmd
				})
			}
			out, code := run()
			sees(t, out, code, 1, "fetched 1 checks this run reaches", "missing.md", "0 of 1 checks passed")
			again, code := run()
			sees(t, again, code, 1, "missing.md")
			if strings.Contains(again, "downloading fitness-check-") {
				t.Errorf("a second run must reuse the cached check and download nothing:\n%s", again)
			}
			bare, code := execute(t, func() *exec.Cmd {
				cmd := exec.Command(engine, "--no-builtins", "--policy=external", "--all", "--checks=markdown-links")
				cmd.Dir, cmd.Env = repo, cleanHomeFetch(t)
				return cmd
			})
			sees(t, bare, code, 1, "no check is installed, and --no-builtins turns the built-in checks off")
		}},
		{"a repo pinning a Markdown, a Go and a Swift check, and one with no fields, none cached", func(t *testing.T) {
			t.Parallel()
			for _, c := range []struct {
				staged  map[string]string
				reached string
				skipped string
			}{
				{map[string]string{"notes/plan.md": "# Plan\n"}, "md-files", "go-files, swift-files"},
				{map[string]string{"src/main.go": "package main\n"}, "go-files", "md-files, swift-files"},
				{map[string]string{"app/App.swift": "import Foundation\n"}, "swift-files", "md-files, go-files"},
			} {
				t.Run(c.reached, func(t *testing.T) {
					t.Parallel()
					r := serveOnDemand(t)
					out, code := fetchRunOnDemand(t, r, stagedOnDemand(t, c.staged))
					sees(t, out, code, 1, c.reached+" ran", "no-fields ran", "Skipped (no matching files): "+c.skipped)
					fetchedOnly(t, r, out, c.reached, "no-fields")
				})
			}
		}},
		{"the same repo, and the Markdown check's download does not match its index SHA-256", func(t *testing.T) {
			t.Parallel()
			r := serveOnDemand(t, "md-files")
			out, code := fetchRunOnDemand(t, r, stagedOnDemand(t, map[string]string{"notes/plan.md": "# Plan\n"}))
			sees(t, out, code, 1, "check md-files applies to this run's files but could not be fetched", "checksum mismatch", "no-fields ran")
			absent(t, out, "md-files")
			if !strings.Contains(row(out, "md-files"), "info") {
				t.Errorf("a check that could not be fetched must be an info row, never a pass:\n%s", out)
			}
		}},
		{"the same repo, and the hooks' `FITNESS_WITH_CHECKS=pinned`", func(t *testing.T) {
			t.Parallel()
			r := serveOnDemand(t)
			out, code := fiRun(t, stagedOnDemand(t, map[string]string{"notes/plan.md": "# Plan\n"}), hookedOnDemand(t, r), "--", "--policy=external", checksOnDemand)
			sees(t, out, code, 1, "md-files ran", "Skipped (no matching files): go-files, swift-files")
			if installedInInstall.MatchString(out) {
				t.Errorf("fitness-install must install no check up front:\n%s", out)
			}
			fetchedOnly(t, r, out, "md-files", "no-fields")
		}},
		{"a repo pinning `\"*\": \"v1\"`, the Markdown check cached at 1.0.0, and a release index recorded over ten minutes ago", func(t *testing.T) {
			t.Parallel()
			r := serveOnDemand(t)
			bundle := bundleDiscovery(t)
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"*": "v1"}}`})
			write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
			git(t, repo, "add", "-A")
			env := []string{"FITNESS_CACHE_DIR=" + r.cache}
			out, code := runDiscovery(t, bundle, repo, env, "--policy=external", "--checks=md-files")
			sees(t, out, code, 1, "md-files ran")
			cachedOnDemand(t, r, "md-files", "1.0.0")

			// Ten minutes on, the recorded index is stale, so the run asks GitHub again.
			r.publish(t, firstIndexOnDemand, time.Now().Add(-11*time.Minute))
			silent := append(slices.Clone(env), "HTTPS_PROXY="+silentListener(t), "NO_PROXY=", "no_proxy=")
			out, code = runDiscovery(t, bundle, repo, silent, "--policy=external", "--checks=md-files")
			sees(t, out, code, 1, "asking github.com for the check release index", "GitHub did not answer the check release index")
			fetchedOnly(t, r, out, "md-files")

			// The index lives on GitHub, so the test records the answer that read gets.
			r.publish(t, map[string]string{"md-files": "1.1.0", "go-files": "1.0.0", "swift-files": "1.0.0", "no-fields": "1.0.0", "yaml-files": "1.0.0"}, time.Now())
			out, code = runDiscovery(t, bundle, repo, env, "--policy=external", "--checks=md-files,yaml-files")
			sees(t, out, code, 1, "md-files ran", "Skipped (no matching files): yaml-files")
			cachedOnDemand(t, r, "md-files", "1.1.0")
			fetchedOnly(t, r, out, "md-files", "md-files")

			write(t, repo, map[string]string{"config/app.yaml": "name: app\n"})
			git(t, repo, "add", "-A")
			out, code = runDiscovery(t, bundle, repo, env, "--policy=external", "--checks=md-files,yaml-files")
			sees(t, out, code, 1, "md-files ran", "yaml-files ran")
			fetchedOnly(t, r, out, "md-files", "md-files", "yaml-files")
		}},
		{"a repo pinning a Markdown check, a Go check, and a body check the branch's PR needs, none cached", func(t *testing.T) {
			t.Parallel()
			r := serveOnDemand(t)
			bodyOnlyPaths(t, r)
			pins := `{"checkVersions": {"md-files": "v1", "go-files": "v1", "open-questions": "v1"}}`
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": pins})
			write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
			git(t, repo, "add", "-A")
			out, code := runDiscovery(t, bundleDiscovery(t), repo, []string{"FITNESS_CACHE_DIR=" + r.cache}, "--policy=external", "--checks=local-branch,md-files,go-files,open-questions")
			sees(t, out, code, 1, "md-files ran", "Skipped (no matching files): go-files")
			fetchedOnly(t, r, out, "md-files", "open-questions")
		}},
		{"a repo pinning a check pr-check runs and three it never runs, none cached", func(t *testing.T) {
			t.Parallel()
			r := serveOnDemand(t)
			r.archives["semantic-commit"] = archiveOnDemand(t, "semantic-commit")
			r.publish(t, map[string]string{"semantic-commit": "1.0.0", "md-files": "1.0.0", "go-files": "1.0.0", "swift-files": "1.0.0"}, time.Now())
			repo := pinnedInstall(t, `{"semantic-commit": "v1", "md-files": "v1", "go-files": "v1", "swift-files": "v1"}`)
			rp := standIn(t, map[string][]response{"gh": {noChangelogPR}})
			pr := map[string]any{"number": 1, "title": "Say what the packages are for", "body": prPR(11)}
			env := append(actionsEvent(t, "pull_request", map[string]any{"action": "edited", "pull_request": pr}), rp.env...)
			out, code := runDiscovery(t, bundleDiscovery(t), repo, append(env, "FITNESS_CACHE_DIR="+r.cache), "pr-check")
			sees(t, out, code, 1, "semantic-commit ran")
			fetchedOnly(t, r, out, "semantic-commit")
		}},
	}.run(t)
}

func Test0002_5(t *testing.T) {
	cases{
		{"the same repo", func(t *testing.T) {
			t.Parallel()
			r := serveOnDemand(t)
			out, code := fetchRunOnDemand(t, r, stagedOnDemand(t, map[string]string{"notes/plan.md": "# Plan\n"}), "--all")
			sees(t, out, code, 1, "md-files ran", "go-files ran", "swift-files ran", "no-fields ran")
			fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
		}},
		{"the same repo, and the hooks' `FITNESS_WITH_CHECKS=pinned`", func(t *testing.T) {
			t.Parallel()
			r := serveOnDemand(t)
			out, code := fiRun(t, stagedOnDemand(t, nil), hookedOnDemand(t, r), "--", "config")
			sees(t, out, code, 0, "installed 4 checks in ")
			fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
		}},
		{"a repo pinning a Markdown, a Go and a Swift check, and one with no fields, none cached", func(t *testing.T) {
			t.Parallel()
			r := serveOnDemand(t)
			bundle := bundleDiscovery(t)
			repo := stagedOnDemand(t, nil)
			env := []string{"FITNESS_CACHE_DIR=" + r.cache}
			out, code := runDiscovery(t, bundle, repo, env, "prefetch")
			sees(t, out, code, 0, "installed 4 checks in ")
			fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
			_, set, _ := strings.Cut(out, "installed 4 checks in ")
			set, _, _ = strings.Cut(set, "\n")
			pins, err := os.ReadFile(set + ".pins")
			mustDo(t, err)
			for name := range firstIndexOnDemand {
				if !strings.Contains(string(pins), name+" v") {
					t.Errorf("the check set must record the pin of %s:\n%s", name, pins)
				}
			}
			for name := range firstIndexOnDemand {
				cachedOnDemand(t, r, name, "1.0.0")
			}
			out, code = runDiscovery(t, bundle, repo, env, "--policy=external", checksOnDemand, "--all")
			sees(t, out, code, 1, "md-files ran", "go-files ran", "swift-files ran", "no-fields ran")
			fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
		}},
		{"a repo pinning every check, none installed, and a PR whose title is not a semantic commit", func(t *testing.T) {
			t.Parallel()
			rp := standIn(t, map[string][]response{"gh": {issuePR(t, 11, planPR("- [x] Write the tests."), "Plan"), noChangelogPR}})
			pr := map[string]any{"number": 1, "title": "Say what the packages are for", "body": prPR(11)}
			env := append(actionsEvent(t, "pull_request", map[string]any{"action": "edited", "pull_request": pr}), rp.env...)
			// An empty FITNESS_CHECKS_DIR hands down no checks, as on a contributor's machine.
			out, code := fitness(t, pinnedInstall(t, `{"*": "v1"}`), append(env, "FITNESS_CHECKS_DIR="), "pr-check")
			sees(t, out, code, 1, "type(scope): description")
			if strings.Contains(out, "no result from the check") {
				t.Errorf("pr-check judged the PR body without fetching its checks:\n%s", out)
			}
		}},
	}.run(t)
}

func Test0002_6(t *testing.T) {
	cases{
		{"a repo pinning `\"*\": \"v1\"`, every check cached, and a release index recorded 40 days ago", func(t *testing.T) {
			t.Parallel()
			r := serveOnDemand(t)
			bundle := bundleDiscovery(t)
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"*": "v1"}}`})
			env := []string{"FITNESS_CACHE_DIR=" + r.cache}
			out, code := runDiscovery(t, bundle, repo, env, "prefetch")
			sees(t, out, code, 0, "installed 4 checks in ")

			r.publish(t, firstIndexOnDemand, time.Now().Add(-40*24*time.Hour))
			silent := append(slices.Clone(env), "HTTPS_PROXY="+silentListener(t), "NO_PROXY=", "no_proxy=")
			out, code = runDiscovery(t, bundle, repo, silent, "--policy=external", checksOnDemand, "--all")
			sees(t, out, code, 1, "GitHub did not answer the check release index", "so this run works offline against the index cached",
				"md-files ran", "go-files ran", "swift-files ran", "no-fields ran")
			fetchedOnly(t, r, out, "go-files", "md-files", "no-fields", "swift-files")
		}},
		{"a cached Markdown check, and uncached ones requiring `github`, `network` or nothing", func(t *testing.T) {
			t.Parallel()
			r := onlineCache(t)
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"md-files": "v1", "gh-reads": "v1", "net-reads": "v1", "plain": "v1"}}`})
			write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
			git(t, repo, "add", "-A")
			bundle := bundleDiscovery(t)
			out, code := runDiscovery(t, bundle, repo, []string{"FITNESS_CACHE_DIR=" + r.cache}, "--policy=external", "--checks=md-files")
			sees(t, out, code, 1, "md-files ran")
			fetchedOnly(t, r, out, "md-files")

			silent := []string{"FITNESS_CACHE_DIR=" + r.cache, "HTTPS_PROXY=" + silentListener(t), "NO_PROXY=", "no_proxy=", "GITHUB_API_URL=https://api.github.com"}
			out, code = runDiscovery(t, bundle, repo, silent, "--policy=external", "--checks=md-files,gh-reads,net-reads,plain")
			fetchedOnly(t, r, out, "md-files")
			sees(t, out, code, 1, "md-files ran",
				"Skipped (requires github, and this run is offline: api.github.com did not answer): gh-reads",
				"Skipped (requires network, and this run is offline: api.github.com did not answer): net-reads",
				"check plain applies to this run's files but could not be fetched, so it judged nothing: this run is offline: api.github.com did not answer")
			absent(t, out, "gh-reads", "net-reads", "plain")
			for _, name := range []string{"gh-reads", "net-reads"} {
				if !strings.Contains(row(out, name), "skipped") {
					t.Errorf("%s must be a skipped row offline:\n%s", name, out)
				}
			}
		}},
	}.run(t)
}

func Test0002_7(t *testing.T) {
	cases{
		{"a cache where bundles, releases, local builds and sets went unused for 31 days, some still used or linked", func(t *testing.T) {
			t.Parallel()
			t.Run("only unused releases go", func(t *testing.T) {
				t.Parallel()
				c := prunableCache(t)
				out, code := pruneRun(t, c)
				sees(t, out, code, 1, "md-files ran", "fitness: pruned 5 cached engine and check releases unused for 30 days")
				present(t, out, c.kept...)
				gone(t, out, c.pruned...)
			})
			t.Run("a bundle settings and skills stop pointing into goes", func(t *testing.T) {
				t.Parallel()
				c := prunableCache(t)
				out, code := pruneRun(t, c)
				sees(t, out, code, 1, "pruned 5 cached")
				linked, skilled := bundleKeyCache(c.dir, "e"), bundleKeyCache(c.dir, "f")
				mustDo(t, os.WriteFile(c.settings, []byte(`{"hooks":{}}`), 0o600))
				mustDo(t, os.Remove(filepath.Join(c.skills, "fitness")))
				pruneDue(t, c.dir)
				out, code = pruneRun(t, c)
				sees(t, out, code, 1, "fitness: pruned 2 cached")
				gone(t, out, linked, skilled)
			})
			t.Run("a local Go check build in daily use stays", func(t *testing.T) {
				t.Parallel()
				c := prunableCache(t)
				repo := example(t, "happyRepo", goCheck(t, "go-rule", t.TempDir()))
				env := []string{"FITNESS_CACHE_DIR=" + c.dir, "FITNESS_CHECKS_DIR=" + c.handed}
				out, code := runDiscovery(t, c.engine, repo, env, "check", "go-rule")
				sees(t, out, code, 0, "fitness: building fitness/go-rule with go")
				builds, err := filepath.Glob(filepath.Join(c.dir, "checks", "*", "go-rule"))
				mustDo(t, err)
				if len(builds) != 1 {
					t.Fatalf("want one build of go-rule in the cache, got %v", builds)
				}
				mustDo(t, os.Chtimes(filepath.Dir(builds[0]), unusedCache, unusedCache))
				out, code = runDiscovery(t, c.engine, repo, env, "check", "go-rule")
				sees(t, out, code, 0, "All 1 checks passed")
				pruneDue(t, c.dir)
				out, code = runDiscovery(t, c.engine, repo, env, "check", "go-rule")
				sees(t, out, code, 0, "All 1 checks passed")
				absentText(t, out, "building fitness/go-rule")
				present(t, out, builds[0])
			})
			t.Run("folders an earlier prune set aside are settled", func(t *testing.T) {
				t.Parallel()
				c := prunableCache(t)
				taken := filepath.Join(c.dir, "checks", ".pruning-"+filepath.Base(handedCache(c)))
				mustDo(t, os.MkdirAll(taken, 0o700))
				free := filepath.Join(c.dir, "checks", "fitness-check-back-files-v1.0.0-"+platformCache+".tar")
				write(t, filepath.Join(c.dir, "checks", ".pruning-"+filepath.Base(free)), map[string]string{"fitness-check-back-files": "#!/bin/sh\n"})
				out, code := pruneRun(t, c)
				sees(t, out, code, 1, "md-files ran", "pruned 5 cached")
				gone(t, out, taken, filepath.Join(c.dir, "checks", ".pruning-"+filepath.Base(free)))
				present(t, out, handedCache(c), free)
			})
		}},
		{"a cache pruned an hour ago, and one more check release then went unused for 31 days", func(t *testing.T) {
			t.Parallel()
			c := prunableCache(t)
			out, code := pruneRun(t, c)
			sees(t, out, code, 1, "pruned 5 cached")
			release := filepath.Join(c.dir, "checks", "fitness-check-new-files-v1.0.0-"+platformCache+".tar")
			mustDo(t, os.Chtimes(release, unusedCache, unusedCache))
			out, code = pruneRun(t, c)
			sees(t, out, code, 1, "md-files ran")
			absentText(t, out, "pruned")
			present(t, out, release)

			// A day after the prune, a run prunes again, but only while GitHub
			// answers. Offline, a deleted check could not return.
			pruneDue(t, c.dir)
			indexAtCache(t, c.dir, time.Now().Add(-11*time.Minute))
			out, code = pruneRun(t, c, "HTTPS_PROXY="+silentListener(t), "NO_PROXY=", "no_proxy=", "GITHUB_API_URL=https://api.github.com")
			sees(t, out, code, 1, "md-files ran")
			absentText(t, out, "pruned")
			present(t, out, release)
			indexAtCache(t, c.dir, time.Now())
			out, code = pruneRun(t, c)
			sees(t, out, code, 1, "fitness: pruned 1 cached")
			gone(t, out, release)

			// A run that reads no index still prunes once GitHub answers its probe.
			other := prunableCache(t)
			indexAtCache(t, other.dir, time.Now().Add(-11*time.Minute))
			out, code = pruneRun(t, other)
			sees(t, out, code, 1, "fitness: pruned 5 cached")
		}},
		{"older engine and check versions beside newer cached ones", func(t *testing.T) {
			t.Parallel()
			t.Run("superseded versions go, with the unused sets linking them", func(t *testing.T) {
				t.Parallel()
				c := supersededCache(t)
				out, code := pruneRun(t, c.cacheCache)
				sees(t, out, code, 1, "md-files ran", "fitness: pruned 3 cached engine and check releases unused for 30 days or superseded")
				gone(t, out, c.pruned...)
				present(t, out, c.kept...)
			})
			t.Run("pinned, recently used and lately linked versions stay", func(t *testing.T) {
				t.Parallel()
				c := supersededCache(t)
				out, code := pruneRun(t, c.cacheCache)
				sees(t, out, code, 1, "pruned 3 cached")
				present(t, out, c.pinned, c.recent, c.live)

				// Once no set pins it exactly and a day passes unused, each goes too.
				mustDo(t, os.WriteFile(c.pinnedSet+".pins", []byte("pinned-files v1\n"), 0o600))
				mustDo(t, os.Chtimes(c.recent, twoDaysCache, twoDaysCache))
				pruneDue(t, c.dir)
				out, code = pruneRun(t, c.cacheCache)
				sees(t, out, code, 1, "fitness: pruned 3 cached")
				gone(t, out, c.pinned, c.recent, c.pinnedSet)
			})
			t.Run("the running engine and one asked for by version stay", func(t *testing.T) {
				t.Parallel()
				c := supersededCache(t)
				out, code := pruneRun(t, c.cacheCache)
				sees(t, out, code, 1, "md-files ran", "pruned 3 cached")
				present(t, out, c.running, c.engine)
				out, code = pruneRun(t, c.cacheCache)
				sees(t, out, code, 1, "md-files ran")

				env := fiPrivateCache(t)
				out, code = fiRun(t, t.TempDir(), env, "--install-only")
				sees(t, out, code, 0, "fitness: version=")
				bin := fiLastLine(out)
				key := filepath.Dir(bin)
				cache := filepath.Dir(key)
				newer := regexp.MustCompile(`^fitness-[0-9.]+-`).ReplaceAllString(filepath.Base(key), "fitness-99.0.0-")
				mustDo(t, os.MkdirAll(filepath.Join(cache, newer, "bundle-1"), 0o700))
				mustDo(t, os.Chtimes(key, twoDaysCache, twoDaysCache))
				indexAtCache(t, cache, time.Now())
				repo := example(t, "happyRepo", nil)
				out, _ = runDiscovery(t, c.engine, repo, []string{"FITNESS_CACHE_DIR=" + cache, "FITNESS_CHECKS_DIR=" + t.TempDir()}, "--policy=external", "--checks=md-files")
				absentText(t, out, "pruned")
				present(t, out, key)
			})
		}},
	}.run(t)
}
