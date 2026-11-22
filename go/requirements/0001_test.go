package requirements

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func Test0001_1(t *testing.T) {
	cases{
		{"any folder", func(t *testing.T) {
			t.Parallel()
			out, code := fiRun(t, t.TempDir(), nil, "--help")
			sees(t, out, code, 0, "Usage of fitness-install", "-github-action", "-install-hook", "-install-only", "-version")
		}},
		{"options the installer cannot combine or read", func(t *testing.T) {
			t.Parallel()
			cases := map[string][]string{
				"flag provided but not defined: -unknown":                            {"--unknown"},
				"install-only does not accept runner arguments":                      {"--install-only", "--", "--all"},
				"install-hook cannot be combined with install-only or github-action": {"--install-hook", "--install-only"},
				`invalid release version "invalid"`:                                  {"--version", "invalid"},
			}
			for want, args := range cases {
				out, code := fiRun(t, t.TempDir(), nil, args...)
				sees(t, out, code, 1, "fitness: "+want)
			}
		}},
		{"a folder that is not a git repo", func(t *testing.T) {
			t.Parallel()
			out, code := fiRun(t, t.TempDir(), nil, "--install-hook")
			sees(t, out, code, 1, "fitness: find Git hook directory")
		}},
		{"an unformatted Go file and Go on the `PATH`", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{"main.go": "package main\nfunc main(){\n}\n"})
			out, code := fitness(t, repo, nil, "gofmt")
			sees(t, out, code, 1, "main.go: not gofmt-formatted (run: gofmt -w main.go)")
		}},
		{"an exact release version with a `v` prefix", func(t *testing.T) {
			t.Parallel()
			first, _ := fiRun(t, t.TempDir(), nil, "--install-only")
			match := fiVersion.FindStringSubmatch(first)
			if match == nil {
				t.Fatalf("no release version in:\n%s", first)
			}
			out, code := fiRun(t, t.TempDir(), nil, "--version", "v"+match[1], "--with-checks", "--install-only")
			sees(t, out, code, 0, "fitness: version="+match[1]+" ")
			checks := fiLastLine(out)
			programs := bundleChecks(t, checks)
			if !slices.Contains(programs, "fitness-check-gofmt") {
				t.Fatalf("printed folder has no fitness-check-gofmt among %v\n%s", programs, out)
			}
			for _, program := range programs {
				described, code := execute(t, func() *exec.Cmd {
					return exec.Command(filepath.Join(checks, program), "--describe")
				})
				name := strings.TrimPrefix(program, "fitness-check-")
				sees(t, described, code, 0, `"name":"`+name+`"`)
			}
		}},
		{"asks for the commit hook and skill links in one command", func(t *testing.T) {
			t.Parallel()
			folder := filepath.Join(t.TempDir(), "skills")
			out, code := fiRun(t, t.TempDir(), nil, "--install-hook", "--link-skills", folder)
			sees(t, out, code, 1, "link-skills does not combine with other modes")
			if _, err := os.Lstat(folder); err == nil {
				t.Errorf("skills folder %s was created", folder)
			}
		}},
		{"`fitness-install` built from source without release stamps", func(t *testing.T) {
			t.Parallel()
			bin := filepath.Join(t.TempDir(), "fitness-install")
			build := exec.Command("go", "build", "-o", bin, "./cmd/fitness-install")
			build.Dir = ".."
			if out, err := build.CombinedOutput(); err != nil {
				t.Fatalf("go build: %v\n%s", err, out)
			}
			out, err := exec.Command(bin, "--help").CombinedOutput()
			if !strings.Contains(string(out), `(default "latest")`) {
				t.Errorf("an unstamped fitness-install must default to latest (%v):\n%s", err, out)
			}
		}},
	}.run(t)
}

func Test0001_2(t *testing.T) {
	cases{
		{"a GitHub job asking for install only", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			out, code := fiRun(t, t.TempDir(), fiAction(dir, "FITNESS_INSTALL_ONLY=true"), "--github-action")
			sees(t, out, code, 0)
			bin := fiLastLine(out)
			sees(t, fiFile(t, filepath.Join(dir, "output")), 0, 0, "bin="+bin+"\n")
			sees(t, fiFile(t, filepath.Join(dir, "path")), 0, 0, bin+"\n")
			sees(t, fiFile(t, filepath.Join(dir, "summary")), 0, 0, "✅ fitness-install", "Verified Fitness binaries installed successfully.")
		}},
		{"GitHub job inputs the installer cannot use", func(t *testing.T) {
			t.Parallel()
			cases := map[string][]string{
				`parsing "maybe": invalid syntax`:         {"FITNESS_INSTALL_ONLY=maybe"},
				"choose action check or checks, not both": {"FITNESS_CHECK=gofmt", "FITNESS_CHECKS=gofmt,go-vet"},
				"is outside v99":                          {"FITNESS_VERSION=", "FITNESS_ACTION_REF=v99"},
			}
			for want, inputs := range cases {
				dir := t.TempDir()
				out, code := fiRun(t, t.TempDir(), fiAction(dir, inputs...), "--github-action")
				sees(t, out, code, 1, "::error::fitness-install: ", want)
				sees(t, fiFile(t, filepath.Join(dir, "summary")), 0, 0, "❌ fitness-install", want)
			}
		}},
	}.run(t)
}

func Test0001_3(t *testing.T) {
	cases{
		{"a GitHub token in `GH_TOKEN` or `GITHUB_TOKEN`", func(t *testing.T) {
			t.Parallel()
			version := rlCandidate(t)
			for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					lookups, gh := rlReleases(t, http.StatusOK, version), rlGh(t, "from-gh")
					out, code := rlLatest(t, fiPrivateCache(t), lookups, slices.Concat(gh.env, []string{name + "=from-" + name})...)
					sees(t, out, code, 0, "fitness: version="+version+" ")
					if got := lookups.sent(); !slices.Equal(got, []string{"Bearer from-" + name}) {
						t.Errorf("lookups sent Authorization %q, want one Bearer from-%s", got, name)
					}
					if n := gh.count("gh", "auth token"); n != 0 {
						t.Errorf("asked gh for a token %d times with %s set", n, name)
					}
				})
			}
		}},
		{"is logged in to `gh` and sets no token variable", func(t *testing.T) {
			t.Parallel()
			version := rlCandidate(t)
			lookups, gh := rlReleases(t, http.StatusOK, version), rlGh(t, "from-gh")
			out, code := rlLatest(t, fiPrivateCache(t), lookups, gh.env...)
			sees(t, out, code, 0, "fitness: version="+version+" ")
			if got := lookups.sent(); !slices.Equal(got, []string{"Bearer from-gh"}) {
				t.Errorf("lookups sent Authorization %q, want one Bearer from-gh", got)
			}
		}},
		{"ran a release lookup less than 10 minutes ago", func(t *testing.T) {
			t.Parallel()
			version := rlCandidate(t)
			cache, lookups, gh := fiPrivateCache(t), rlReleases(t, http.StatusOK, version), rlGh(t, "from-gh")
			first, _ := rlLatest(t, cache, lookups, gh.env...)
			again, code := rlLatest(t, cache, lookups, gh.env...)
			sees(t, again, code, 0, "fitness: latest resolved to v"+version, "fitness: version="+version+" ")
			if fiLastLine(first) != fiLastLine(again) {
				t.Errorf("second run installed %q, first %q", fiLastLine(again), fiLastLine(first))
			}
			if n := len(lookups.sent()); n != 1 {
				t.Errorf("two runs within 10 minutes sent %d lookups, want 1", n)
			}
		}},
		{"a recorded or installed version and GitHub refuses the lookup with HTTP 403", func(t *testing.T) {
			t.Parallel()
			version := rlCandidate(t)
			givens := map[string]func(*testing.T, []string, string){"recorded": rlRecorded, "installed": rlInstalled}
			for name, given := range givens {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					cache, gh := fiPrivateCache(t), rlGh(t, "from-gh")
					given(t, cache, version)
					out, code := rlLatest(t, cache, rlReleases(t, http.StatusForbidden, version), gh.env...)
					sees(t, out, code, 0, "fitness: warning: ", "HTTP 403; using "+version, "fitness: version="+version+" ")
					if n := strings.Count(out, "warning"); n != 1 {
						t.Errorf("printed %d warnings, want 1:\n%s", n, out)
					}
				})
			}
		}},
		{"an exact release version, as the GitHub job passes", func(t *testing.T) {
			t.Parallel()
			version := rlCandidate(t)
			lookups, gh := rlReleases(t, http.StatusOK, version), rlGh(t, "from-gh")
			out, code := fiRun(t, t.TempDir(), slices.Concat(fiPrivateCache(t), lookups.env, gh.env), "--version", "v"+version, "--install-only")
			sees(t, out, code, 0, "fitness: version="+version+" ")
			if n := len(lookups.sent()); n != 0 {
				t.Errorf("an exact version sent %d lookups, want 0", n)
			}
			if n := gh.count("gh", "auth token"); n != 0 {
				t.Errorf("an exact version asked gh for a token %d times", n)
			}
		}},
		{"a stale recorded version and a releases API that never answers", func(t *testing.T) {
			t.Parallel()
			version := rlCandidate(t)
			cache, gh, api := fiPrivateCache(t), rlGh(t, "from-gh"), silentListener(t)
			rlRecorded(t, cache, version)
			env := slices.Concat(cache, gh.env, []string{"GITHUB_API_URL=" + api, "GH_TOKEN=", "GITHUB_TOKEN="})
			start := time.Now()
			out, code := fiRun(t, t.TempDir(), env, "--version", "latest", "--install-only")
			took := time.Since(start)
			sees(t, out, code, 0, "fitness: asking "+strings.TrimPrefix(api, "http://")+" for the latest release (gives up after 5s)",
				"fitness: warning: ", "Client.Timeout exceeded", "; using "+version, "fitness: version="+version+" ")
			if took > rlSilentBudget {
				t.Errorf("a silent releases API held the run %s, want under %s", took.Round(time.Second), rlSilentBudget)
			}
		}},
	}.run(t)
}

func Test0001_4(t *testing.T) {
	cases{
		{"an empty skills folder for an agent", func(t *testing.T) {
			t.Parallel()
			folder := filepath.Join(t.TempDir(), "skills")
			out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
			sees(t, out, code, 0, "fitness: skill "+filepath.Join(folder, "add-requirements")+" linked to ")
			sees(t, fiFile(t, filepath.Join(folder, "add-requirements", "SKILL.md")), 0, 0, "name: add-requirements")
		}},
		{"a broken skill link and a skill folder of my own in an agent's skills folder", func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			mustDo(t, os.Symlink(filepath.Join(folder, "gone", "skills", "add-requirements"), filepath.Join(folder, "add-requirements")))
			write(t, folder, map[string]string{"mine/SKILL.md": "mine\n"})
			out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
			sees(t, out, code, 0, "add-requirements linked to ")
			sees(t, fiFile(t, filepath.Join(folder, "add-requirements", "SKILL.md")), 0, 0, "name: add-requirements")
			sees(t, fiFile(t, filepath.Join(folder, "mine", "SKILL.md")), 0, 0, "mine\n")
		}},
		{"a link to a dropped skill and a link into another tool's bundle in a skills folder", func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			fiRun(t, t.TempDir(), nil, "--link-skills", folder)
			bundle := filepath.Dir(mustLink(t, filepath.Join(folder, "add-requirements")))
			mustDo(t, os.Symlink(filepath.Join(bundle, "retired"), filepath.Join(folder, "retired")))
			other := filepath.Join(t.TempDir(), "bundle-1", "skills", "other")
			write(t, other, map[string]string{"SKILL.md": "other\n"})
			mustDo(t, os.Symlink(other, filepath.Join(folder, "foreign")))
			out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
			sees(t, out, code, 0, "retired removed")
			if _, err := os.Lstat(filepath.Join(folder, "retired")); err == nil {
				t.Errorf("link to the dropped skill stayed")
			}
			sees(t, fiFile(t, filepath.Join(folder, "foreign", "SKILL.md")), 0, 0, "other\n")
		}},
		{"an empty skills folder for an agent: every linked skill file is readable by other users", func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			fiRun(t, t.TempDir(), nil, "--link-skills", folder)
			for _, name := range []string{"add-requirements", filepath.Join("add-requirements", "SKILL.md")} {
				info, err := os.Stat(filepath.Join(folder, name))
				mustDo(t, err)
				if info.Mode().Perm()&0o044 == 0 {
					t.Errorf("%s mode %v is not readable by others", name, info.Mode().Perm())
				}
			}
		}},
		{"an empty skills folder for an agent: the fitness skill is linked there, every link in it resolves, and each check the engine runs itself has its doc comment", func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
			sees(t, out, code, 0, "fitness: skill "+filepath.Join(folder, "fitness")+" linked to ")
			skill := fiFile(t, filepath.Join(folder, "fitness", "SKILL.md"))
			sees(t, skill, 0, 0, "name: fitness\n", "](../../docs/checks.md)", "](../../docs/requirements)", "](../../docs/architecture)")
			source := filepath.Join("..", "..", "skills", "fitness")
			if kept := fiFile(t, filepath.Join(source, "SKILL.md")); kept != skill {
				t.Fatalf("linked fitness skill differs from the one kept in %s", source)
			}
			for _, link := range skillLink.FindAllStringSubmatch(skill, -1) {
				resolves(t, filepath.Join(source, link[1]), "fitness skill link "+link[1])
			}
			for _, name := range checkNames(t) {
				resolves(t, filepath.Join("..", "cmd", "fitness-check-"+name, "doc.go"), "doc comment the fitness skill names for "+name)
			}
		}},
		{"an empty skills folder for an agent: no linked skill is named for a single check", func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			fiRun(t, t.TempDir(), nil, "--link-skills", folder)
			linked, err := os.ReadDir(folder)
			mustDo(t, err)
			if len(linked) == 0 {
				t.Fatal("no skill was linked")
			}
			checks := map[string]bool{}
			for _, name := range checkNames(t) {
				checks[name] = true
			}
			for _, skill := range linked {
				if checks[skill.Name()] {
					t.Errorf("skill %s is named for a single check", skill.Name())
				}
			}
		}},
	}.run(t)
}

func Test0001_5(t *testing.T) {
	cases{
		{"the onboarding skill linked into an agent's skills folder", func(t *testing.T) {
			t.Parallel()
			skill := onboardingSkill(t)
			sees(t, skill, 0, 0, "## Install and run", "docs/ci.md#download-the-go-installer", "docs/ci.md#run-checks",
				"docs/ci.md#commit-hook", "## Read a failure", "skills/fitness/SKILL.md", "## Plans and pull requests", "fitness-check-plan-trailer/README.md")
		}},
		{"the onboarding skill linked into an agent's skills folder: every link names a file and a real heading, not a code comment, that exist in the repo", func(t *testing.T) {
			t.Parallel()
			links := onboardingLink.FindAllStringSubmatch(onboardingSkill(t), -1)
			if len(links) == 0 {
				t.Fatal("the onboarding skill links no doc")
			}
			for _, link := range links {
				checkSkillLink(t, link[1], link[2])
			}
		}},
	}.run(t)
}

func Test0001_6(t *testing.T) {
	cases{
		{"a Claude Code settings file with settings and hooks of my own", func(t *testing.T) {
			t.Parallel()
			installer(t)
			file := filepath.Join(t.TempDir(), "settings.json")
			mine := `{"theme": "dark", "hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "my-check"}]}]}}`
			mustDo(t, os.WriteFile(file, []byte(mine), 0o600))
			claudeSettings(t, file)
			settings := claudeSettings(t, file)
			if settings["theme"] != "dark" {
				t.Errorf("my theme setting was lost: %v", settings)
			}
			ours, others := fitnessHandlers(settings)
			if len(others) != 1 || others[0] != "my-check" {
				t.Errorf("my own hook must stay once, got %v", others)
			}
			for _, event := range hookEvents {
				installedOnce(t, event, ours[event])
			}
			keptMode(t, file, 0o600)
			skillDeclares(t, settings)
		}},
	}.run(t)
}

func Test0001_7(t *testing.T) {
	cases{
		{"docs and entry files whose front matter holds only fitness keys, plain or in an HTML comment", func(t *testing.T) {
			t.Parallel()
			entry := bareEntryText("2026.10.11.0900", "Feat: one change.")
			repo, out, code := stripRun(t, map[string]string{
				"docs/guide.md": "---\nfitnessFunctions:\n  - prose-budget\nrelatedConfigurations: ['../.fitnessrc.json']\n---\n\n" + stripBody,
				"changelog/2026-10-11-0900-one-change.md": entryFileText("2026.10.11.0900", "Feat: one change."),
				".github/PULL_REQUEST_TEMPLATE.md":        "<!--\n---\nfitnessFunctions: ['pr-structure']\n---\n-->\n<!-- One sentence. -->\n> REPLACE-ME\n",
			})
			sees(t, out, code, 0, "stripped docs/guide.md", "stripped changelog/2026-10-11-0900-one-change.md",
				"stripped .github/PULL_REQUEST_TEMPLATE.md", "fitness strip-front-matter: stripped 3 files\n")
			holds(t, repo, "docs/guide.md", stripBody)
			holds(t, repo, "changelog/2026-10-11-0900-one-change.md", entry)
			holds(t, repo, ".github/PULL_REQUEST_TEMPLATE.md", "<!-- One sentence. -->\n> REPLACE-ME\n")
		}},
		{"docs whose front matter also holds another tool's key, like an issue template", func(t *testing.T) {
			t.Parallel()
			bug := "---\nrelatedConfigurations: ['../../.fitnessrc.json']\nname: Bug\nlabels: Bug\n\n---\n\n## Steps\n"
			skill := "---\nname: tidy\ndescription: Tidy docs.\nfitnessFunctions:\n  - prose-budget\nrelatedConfigurations: ['../../.fitnessrc.json']\n---\n\n# Tidy\n"
			repo, out, code := stripRun(t, map[string]string{".github/ISSUE_TEMPLATE/bug.md": bug, "skills/tidy/SKILL.md": skill})
			sees(t, out, code, 0, "stripped .github/ISSUE_TEMPLATE/bug.md: front matter keeps name, labels",
				"stripped skills/tidy/SKILL.md: front matter keeps name, description")
			holds(t, repo, ".github/ISSUE_TEMPLATE/bug.md", "---\nname: Bug\nlabels: Bug\n\n---\n\n## Steps\n")
			holds(t, repo, "skills/tidy/SKILL.md", "---\nname: tidy\ndescription: Tidy docs.\n---\n\n# Tidy\n")
		}},
		{"docs with no front matter, front matter without fitness keys, or a leading rule", func(t *testing.T) {
			t.Parallel()
			plain := "# Plain\n\nShort and clean.\n"
			titled := "---\ntitle: Notes\n---\n\n# Notes\n"
			ruled := "---\n\nA note set between two rules.\n\n---\n\n# Ruled\n"
			repo, out, code := stripRun(t, map[string]string{"plain.md": plain, "docs/notes.md": titled, "docs/ruled.md": ruled})
			sees(t, out, code, 0, "fitness strip-front-matter: stripped 0 files\n")
			if strings.Contains(out, "plain.md") || strings.Contains(out, "notes.md") {
				t.Errorf("the output lists a file it left alone:\n%s", out)
			}
			holds(t, repo, "plain.md", plain)
			holds(t, repo, "docs/notes.md", titled)
			holds(t, repo, "docs/ruled.md", ruled)
		}},
		{"docs with fitness front matter and CRLF endings or a byte order mark, and a doc with an empty block", func(t *testing.T) {
			t.Parallel()
			repo, out, code := stripRun(t, map[string]string{
				"docs/crlf.md":  "---\r\nfitnessFunctions:\r\n  - prose-budget\r\n---\r\n\r\n# Crlf\r\n",
				"docs/bom.md":   "\ufeff---\nfitnessFunctions: ['prose-budget']\n---\n\n# Bom\n",
				"docs/empty.md": "---\n---\n\n# Empty\n",
			})
			sees(t, out, code, 0, "stripped docs/crlf.md", "stripped docs/bom.md", "stripped docs/empty.md")
			holds(t, repo, "docs/crlf.md", "# Crlf\r\n")
			holds(t, repo, "docs/bom.md", "\ufeff# Bom\n")
			holds(t, repo, "docs/empty.md", "# Empty\n")
		}},
		{"docs that fail no-fitness-front-matter, one beside another tool's key", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{
				"docs/guide.md":        "---\nfitnessFunctions:\n  - prose-budget\n---\n\n# Guide\n",
				"skills/tidy/SKILL.md": "---\nname: tidy\nfitnessFunctions:\n  - prose-budget\n---\n\n# Tidy\n",
			})
			out, code := fitness(t, repo, nil, "no-fitness-front-matter")
			sees(t, out, code, 1, "docs/guide.md: front matter holds", "skills/tidy/SKILL.md: front matter holds")
			out, code = fitness(t, repo, nil, "strip-front-matter")
			sees(t, out, code, 0, "fitness strip-front-matter: stripped 2 files\n")
			out, code = fitness(t, repo, nil, "no-fitness-front-matter")
			sees(t, out, code, 0, "All 1 checks passed")
		}},
	}.run(t)
}

func Test0001_8(t *testing.T) {
	cases{
		{"ADR files and asks for a dry run", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", migrateRepo())
			out, code, rp := migrate(t, repo, migrateGitHub(listedIssues(t, keptSQLite(t))), "--dry-run")
			sees(t, out, code, 0,
				"kept docs/architecture/adr/0001-use-sqlite.md: "+sqliteIssue, "would edit "+sqliteIssue,
				`would open "[ADR] Small files" from fitness/adr-0001/README.md`,
				"would remove docs/architecture/adr/0002-sync.md",
				"would close the issue for docs/architecture/adr/0002-sync.md as superseded",
				"would close the issue for docs/architecture/adr/0003-cache.md as rejected",
				"would rewrite docs/guide.md", "would rewrite fitness/adr-0001/README.md")
			if calls := rp.calls("gh"); len(calls) != rp.count("gh", "repo view")+rp.count("gh", "issue list") {
				t.Errorf("a dry run must only read GitHub, got gh calls %q", calls)
			}
			if status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output(); err != nil || len(status) > 0 {
				t.Errorf("a dry run must change no file, got:\n%s", status)
			}
		}},
		{"ADR files in an `adr` folder and a check README holding its decision", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", migrateRepo())
			out, code, rp := migrate(t, repo, migrateGitHub(listedIssues(t)))
			sees(t, out, code, 0, "opened "+sqliteIssue+" from docs/architecture/adr/0001-use-sqlite.md",
				"removed docs/architecture/adr/0001-use-sqlite.md", "rewrote docs/guide.md")
			wantCall(t, rp, "an ADR issue opened with the file's sections", "issue create", "--label ADR", "[ADR] Use SQLite",
				"> Use SQLite\\n\\n## Background\\n\\nWe sync offline.", "## Decision\\n\\nStore notes in SQLite",
				"## Currently Known Consequences\\n\\nOne file holds every note.")
			if rp.called("gh", "issue create", "## Status") {
				t.Errorf("want no Status section, as the issue's state is its status, got gh calls %q", rp.calls("gh"))
			}
			wantCall(t, rp, "the superseded ADR's issue opened with its quoted summary as the pitch", "issue create", "[ADR] Sync whole files",
				"--body-file - > Whole files sync simply.\\n\\n## Background\\n\\nNotes change offline.")
			wantCall(t, rp, "the rejected ADR's issue closed", "issue close "+cacheIssue, "--reason not planned", "Rejected, as the file it came from said.")
			wantCall(t, rp, "the superseded ADR's issue closed naming the issue that replaces it",
				"issue close "+syncIssue, "--reason not planned", "Superseded by "+sqliteIssue)
			wantCall(t, rp, "the superseded check's issue closed naming the issue that replaces it",
				"issue close "+oldIssue, "--reason not planned", "Superseded by "+filesIssue)
			if rp.called("gh", "issue close "+sqliteIssue) {
				t.Errorf("want an accepted ADR's issue left open, got gh calls %q", rp.calls("gh"))
			}
			wantCall(t, rp, "the first issue's link to the second ADR pointed at its issue", "issue edit "+sqliteIssue, "as [ADR 0002]("+syncIssue+") says")
			wantFiles(t, repo, map[string]string{
				"docs/guide.md":                     "# Guide\n\nRead [the storage decision](" + sqliteIssue + ") and [small files](" + filesIssue + ").\n",
				"docs/architecture/adr/README.md":   "# ADRs\n\n- [Use SQLite](" + sqliteIssue + ")\n- [Sync](" + syncIssue + ")\n",
				"fitness/adr-0001/README.md":        "# 0001 — Small files\n\nSmall files read fast.\n\n" + filesIssue + "\n\n## Check\n\nFails a long Markdown file.\n",
				"fitness/adr-0002/README.md":        "# Old rule\n\nAn old rule.\n\n" + oldIssue + "\n\nStatus: Superseded by " + filesIssue + "\n",
				"docs/architecture/adr/template.md": migrateRepo()["docs/architecture/adr/template.md"],
			})
			if _, err := os.Stat(filepath.Join(repo, "docs", "architecture", "adr", "0002-sync.md")); !os.IsNotExist(err) {
				t.Errorf("want the migrated ADR file removed, got %v", err)
			}
			out, code = fitness(t, repo, nil, "adr-as-issue")
			sees(t, out, code, 0, "All 1 checks passed")
		}},
		{"reruns adr-migrate from a subfolder after issues for its files exist", func(t *testing.T) {
			t.Parallel()
			listed := listedIssues(t, keptSQLite(t), [4]string{syncIssue, "docs/architecture/adr/0002-sync.md", "Sync whole files.", "CLOSED"})
			repo := example(t, "happyRepo", with(migrateRepo(), map[string]string{
				".github/ISSUE_TEMPLATE/adr.md": "---\nname: ADR\nabout: A decision.\nlabels: ADR\n---\n\n> REPLACE-ME\n\n## Why\n\n## Decision\n\n## Effects\n",
			}))
			out, code, rp := migrate(t, filepath.Join(repo, "docs"), migrateGitHub(listed), "--repo=may-journal/example")
			sees(t, out, code, 0, "kept docs/architecture/adr/0001-use-sqlite.md: "+sqliteIssue)
			wantCall(t, rp, "a new issue under the repo template's headings", "issue create", "[ADR] Small files",
				"## Why\\n\\n", "## Decision\\n\\n", "## Effects\\n\\n")
			if rp.called("gh", "issue create", "[ADR] Use SQLite") || rp.called("gh", "issue create", "[ADR] Sync whole files") {
				t.Errorf("want the existing issues kept, not opened again, got gh calls %q", rp.calls("gh"))
			}
			wantCall(t, rp, "the kept issue's link to a moved ADR file pointed at its issue", "issue edit "+sqliteIssue, "Synced as [ADR 0002]("+syncIssue+").")
			if rp.called("gh", "issue close "+syncIssue) {
				t.Errorf("want a closed superseded issue left as it is, got gh calls %q", rp.calls("gh"))
			}
			out, code, rp = migrate(t, repo, migrateGitHub(listed), "--repo=may-journal/example")
			sees(t, out, code, 0, "no ADR files to move")
			if rp.called("gh", "issue create") {
				t.Errorf("a rerun after the files moved must open nothing, got gh calls %q", rp.calls("gh"))
			}
		}},
	}.run(t)
}
