package requirements

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func Test0004_1(t *testing.T) {
	cases{
		{"a local check with a new name", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", localCheck("house-rule", passing))
			out, code := fitness(t, repo, nil)
			sees(t, out, code, 0, allPassedWith(1))
			passedWith(t, out, "house-rule", "1")
		}},
		{"a local check with a new name: only the local check runs", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", localCheck("house-rule", passing))
			out, code := fitness(t, repo, nil, "house-rule")
			sees(t, out, code, 0, "All 1 checks passed")
			passedWith(t, out, "house-rule", "1")
		}},
		{"a shell check in `fitness/<name>/` with a README and no config", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", shellCheck("house-rule", "", passing))
			out, code := fitness(t, repo, nil)
			sees(t, out, code, 0, allPassedWith(1))
			passedWith(t, out, "house-rule", "1")
		}},
		{"a Go check in `fitness/<name>/` built on the check SDK", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", goCheck(t, "go-rule", t.TempDir()))
			out, code := fitness(t, repo, nil, "check", "go-rule")
			sees(t, out, code, 0, "fitness: building fitness/go-rule with go", "All 1 checks passed")
			passedWith(t, out, "go-rule", "3")
			out, code = fitness(t, repo, nil, "check", "go-rule")
			sees(t, out, code, 0, "All 1 checks passed")
			passedWith(t, out, "go-rule", "3")
			if strings.Contains(out, "building fitness/go-rule") {
				t.Errorf("a second run of an unchanged Go check built it again:\n%s", out)
			}
		}},
		{"checks in `fitness/` that find a problem", func(t *testing.T) {
			t.Parallel()
			files := with(shellCheck("house-rule", "", `{"ok": false, "errors": ["the house rule is broken"], "filesChecked": 1}`),
				with(goCheck(t, "go-rule", t.TempDir()), map[string]string{
					"notes.md": "# Notes\n\nTODO: write these.\n",
					"fitness/go-rule/main_test.go": "package main\n\nimport (\n\t\"testing\"\n\n\t\"github.com/may-journal/fitness-runner/go/check\"\n)\n\nfunc TestRule(t *testing.T) { check.Test(t, rule) }\n\n" +
						"func TestFixture(t *testing.T) {\n\tif check.Judge(t, rule, check.Repo(t, map[string]string{\"a.md\": \"TODO\"})).Ok {\n\t\tt.Error(\"a TODO must fail\")\n\t}\n}\n",
				}))
			repo := example(t, "happyRepo", files)
			out, code := fitness(t, repo, nil, "check", "house-rule")
			sees(t, out, code, 1, "the house rule is broken")
			if row(out, "go-rule") != "" || row(out, "markdown-links") != "" {
				t.Errorf("fitness check house-rule ran other checks too:\n%s", out)
			}
			unitTested(t, repo)
		}},
		{"a repo with a check in `fitness/`", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", shellCheck("house-rule", "", passing))
			got := listedCheck{}
			for _, c := range listJSON(t, repo) {
				if c.Name == "house-rule" {
					got = c
				}
			}
			if got.Scope != "local" || got.Summary != "Keeps the house in order." {
				t.Errorf("fitness list shows house-rule with scope %q and summary %q, want local and the README's first sentence", got.Scope, got.Summary)
			}
			out, code := fitness(t, repo, nil, "explain", "house-rule")
			sees(t, out, code, 0, "# house-rule", "Scope: local", "Keeps the house in order. Each room stays tidy.", "## Fix", "Tidy the room the error names.")
		}},
		{"a check in `fitness/` that declares the test stage", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", shellCheck("suite-rule", `{"stage": "tests"}`, passing))
			for _, c := range listJSON(t, repo) {
				if c.Name == "suite-rule" && c.Stage != "tests" {
					t.Errorf("fitness list shows suite-rule in stage %q, want the tests stage it declares", c.Stage)
				}
			}
		}},
	}.run(t)
}

func Test0004_2(t *testing.T) {
	cases{
		{"external policy and a local check named like a bundled check", func(t *testing.T) {
			t.Parallel()
			files := with(localCheck("markdown-no-bold-italic", passing), map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
			out, code := fitness(t, example(t, "happyRepo", files), nil)
			sees(t, out, code, 0, "fitness: local checks replace built-in markdown-no-bold-italic", allPassed())
			if strings.Contains(out, "disallowed **bold**") {
				t.Errorf("the bundled markdown-no-bold-italic ran beside its local replacement:\n%s", out)
			}
			replacing := with(localCheck("markdown-no-bold-italic", passing), map[string]string{".fitnessrc.json": `{"policy": "external", "localChecks": "checks"}` + "\n"})
			out, code = boldPR(t, example(t, "happyRepo", replacing))
			sees(t, out, code, 0, "fitness: local checks replace built-in markdown-no-bold-italic", "Passed.")
		}},
		{"external policy and disabledChecks naming a failing check: the run follows it and the turned-off check never runs", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{
				".fitnessrc.json": disabling,
				"README.md":       readme + "\nSome **bold** text.\n",
			})
			out, code := fitness(t, repo, nil)
			sees(t, out, code, 0, allPassedWith(-1))
			if row(out, "markdown-no-bold-italic") != "" {
				t.Errorf("disabled markdown-no-bold-italic still ran:\n%s", out)
			}
			out, code = boldPR(t, example(t, "happyRepo", nil))
			sees(t, out, code, 1, `disallowed \*\*bold`)
			out, code = boldPR(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"policy": "external", "disabledChecks": ["markdown-no-bold-italic"]}` + "\n"}))
			sees(t, out, code, 0, "Passed.")
		}},
	}.run(t)
}

func Test0004_3(t *testing.T) {
	cases{
		{"org policy and a local check named like a bundled check", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", localCheck("markdown-no-bold-italic", passing))
			out, code := fitness(t, repo, nil, "--all")
			sees(t, out, code, 1, "local check markdown-no-bold-italic replaces a built-in check, and org policy runs the built-in checks as the shared bar")
		}},
		{"org policy and disabledChecks naming a check", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": disabling})
			out, code := fitness(t, repo, nil, "--all")
			sees(t, out, code, 1, "disabledChecks turns off markdown-no-bold-italic, and org policy runs every check as the shared bar")
		}},
		{"a repo whose org policy config turns off a check", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"disabledChecks": ["cspell"]}` + "\n"})
			out, code := fitness(t, repo, nil, "config")
			sees(t, out, code, 1, "A run stops before any check:", "org policy runs every check as the shared bar")
		}},
	}.run(t)
}

func Test0004_4(t *testing.T) {
	cases{
		{"a check setting whose directory or check is missing, outside the repo, or named like a subcommand: it is red naming the setting, then naming the fix", func(t *testing.T) {
			t.Parallel()
			for setting, want := range map[string]string{
				`{"localChecks": "checks"}`:             `fitness setup: .fitnessrc.json localChecks directory "checks" does not exist`,
				`{"localChecks": "../checks"}`:          `.fitnessrc.json localChecks "../checks" must be a path inside the repo`,
				`{"disabledChecks": ["no-such-check"]}`: `disabledChecks: unknown or empty check "no-such-check"`,
			} {
				repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": setting + "\n"})
				out, code := fitness(t, repo, nil)
				sees(t, out, code, 1, want)
			}
			out, code := fitness(t, example(t, "happyRepo", localCheck("boxes", passing)), nil)
			sees(t, out, code, 1, "local check boxes shares a fitness subcommand's name")
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"localChecks": "checks"}` + "\n"})
			mustDo(t, os.Symlink(t.TempDir(), filepath.Join(repo, "checks")))
			out, code = fitness(t, repo, nil)
			sees(t, out, code, 1, `.fitnessrc.json localChecks "checks" must be a path inside the repo`)
		}},
		{"a local check file without execute permission", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", localCheck("house-rule", passing))
			mustDo(t, os.Chmod(filepath.Join(repo, "checks", "fitness-check-house-rule"), 0o644))
			out, code := fitness(t, repo, nil)
			sees(t, out, code, 1, "local check checks/fitness-check-house-rule is not an executable file; run chmod +x checks/fitness-check-house-rule")
		}},
		{"a Go check and no go on PATH", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", goCheck(t, "go-rule", t.TempDir()))
			out, code := fitness(t, repo, []string{pathWithout(t, "go")}, "check", "go-rule")
			sees(t, out, code, 1, "local check fitness/go-rule is Go source, and go is not on PATH; install Go from https://go.dev/dl")
		}},
		{"a check folder without a README, without a program, without `fitness/go.mod`, or with Go that does not build", func(t *testing.T) {
			t.Parallel()
			goRule := goCheck(t, "go-rule", t.TempDir())
			for want, files := range map[string]map[string]string{
				"local check fitness/house-rule has no README.md":                                              {"fitness/house-rule/check": verdictScript(passing)},
				"local check fitness/house-rule has no program; add an executable check or a main.go":          {"fitness/house-rule/README.md": "# House rule\n\nKeeps the house.\n"},
				"local check fitness/go-rule is Go source, so fitness/go.mod must exist":                       with(goRule, map[string]string{"fitness/go.mod": ""}),
				"local check fitness/go-rule does not build; fix it so `go -C fitness build ./go-rule` passes": with(goRule, map[string]string{"fitness/go-rule/main.go": "package main\n\nfunc main() { undefined() }\n"}),
			} {
				out, code := fitness(t, example(t, "happyRepo", files), nil)
				sees(t, out, code, 1, want)
			}
		}},
		{"a check whose minFitness is newer than the installed fitness", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", shellCheck("new-rule", `{"minFitness": "999.0.0"}`, passing))
			out, code := fitness(t, repo, nil, "check", "new-rule")
			sees(t, out, code, 1, "check new-rule needs fitness 999.0.0 or later", "--version=999.0.0")
		}},
	}.run(t)
}

func Test0004_5(t *testing.T) {
	cases{
		{"a repo whose config turns off a check and names its Plan approvers", func(t *testing.T) {
			t.Parallel()
			var view configView
			stdoutOf(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": configured}), &view, "config", "--json")
			seesEntry(t, view.Settings, configSetting{Key: "planApprovers", Value: []any{"your-login"}, Source: "file"})
			seesEntry(t, view.Settings, configSetting{Key: "scopedPullRequests", Value: nil, Source: "unset"})
			seesEntry(t, view.Checks, checkState{Name: "cspell", Enabled: false, Reason: "disabledChecks in .fitnessrc.json turns it off"})
			seesEntry(t, view.Checks, checkState{Name: "markdown-links", Enabled: true, Reason: "external policy runs every check"})
		}},
		{"a repo", func(t *testing.T) {
			t.Parallel()
			schema := printedSchema(t)
			if !refusesUnknownKeys(schema) {
				t.Errorf("the schema must describe an object that refuses unknown keys, got %+v", schema)
			}
			got := schema.Properties["policy"]
			if want := (schemaKey{Type: "string", Description: got.Description, Default: "org"}); !reflect.DeepEqual(got, want) {
				t.Errorf("policy must give its type and its default org, got %+v", got)
			}
			if got.Description == "" {
				t.Error("policy must carry a description")
			}
			mapOfStrings(t, schema.Properties["checkVersions"])
		}},
		{"a repo, whose checks declare every key they read, an opt-in check's among them", func(t *testing.T) {
			t.Parallel()
			var schema schemaKey
			// An empty FITNESS_CHECKS_DIR hands down no checks, as on a contributor's machine.
			stdoutWith(t, example(t, "happyRepo", nil), []string{"FITNESS_CHECKS_DIR="}, &schema, "config", "schema")
			for path, want := range checkKeyDefaults {
				describesKey(t, schema, path, want)
			}
		}},
		{"a local check that declares a config key", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", with(shellCheck("house-rule", houseDescribe("house-rule"), passing),
				map[string]string{".fitnessrc.json": `{"houseRule": {"limit": 5}}` + "\n"}))
			var view configView
			stdoutOf(t, repo, &view, "config", "--json")
			seesEntry(t, view.Settings, configSetting{Key: "houseRule.limit", Value: float64(5), Source: "file"})
			var schema schemaKey
			stdoutOf(t, repo, &schema, "config", "schema")
			if got := schema.Properties["houseRule"].Properties["limit"]; got.Type != "integer" || got.Default != float64(3) {
				t.Errorf("the schema must list houseRule.limit as an integer defaulting to 3, got %+v", got)
			}
			out, code := fitness(t, repo, nil, "explain", "house-rule")
			sees(t, out, code, 0, "## Config keys", "- `houseRule.limit` (integer, default 3): Most rooms a house may have.")
		}},
		{"two local checks that declare one config key", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", with(shellCheck("house-rule", houseDescribe("house-rule"), passing),
				shellCheck("home-rule", houseDescribe("home-rule"), passing)))
			out, code := fitness(t, repo, nil, "config")
			sees(t, out, code, 1, `checks home-rule and house-rule both declare the "houseRule" config key`)
		}},
		{"a Go local check that declares its options type's keys", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", keysOfCheck(t))
			var schema schemaKey
			stdoutOf(t, repo, &schema, "config", "schema")
			if got := schema.Properties["houseRule"].Properties["limit"]; got.Type != "integer" || got.Default != float64(3) {
				t.Errorf("the schema must list houseRule.limit as an integer defaulting to 3, got %+v", got)
			}
		}},
		{"a local check that declares an engine key", func(t *testing.T) {
			t.Parallel()
			describe := `{"name": "house-rule", "config": [{"key": "policy", "type": "string", "description": "Whose rules."}]}`
			repo := example(t, "happyRepo", shellCheck("house-rule", describe, passing))
			out, code := fitness(t, repo, nil, "config")
			sees(t, out, code, 1, `check house-rule declares the engine's "policy" config key, which only the engine may declare`)
		}},
		{"a local check that declares a nested path as a top-level key", func(t *testing.T) {
			t.Parallel()
			describe := `{"name": "house-rule", "config": [{"key": "houseRule.limit", "type": "integer", "description": "Most rooms."}]}`
			repo := example(t, "happyRepo", shellCheck("house-rule", describe, passing))
			out, code := fitness(t, repo, nil, "config")
			sees(t, out, code, 1, `check house-rule declares a bad config key: "houseRule.limit" must be one name under ""`)
		}},
		{"fitness source code that reads config keys through the sections it registers", func(t *testing.T) {
			t.Parallel()
			described := describedKeys("", printedSchema(t))
			for _, key := range keysFitnessReads(t) {
				if described[key] == "" {
					t.Errorf("the code reads config key %s, which the schema does not describe", key)
				}
			}
		}},
	}.run(t)
}

func Test0004_6(t *testing.T) {
	cases{
		{"a workflow job whose `timeout-minutes` of 10 or more has no comment citing its observed maximum", func(t *testing.T) {
			t.Parallel()
			out, code := timeoutsCheck(t, map[string]string{".github/workflows/ci.yml": timeoutJob("    # The build is slow.\n", "12")})
			sees(t, out, code, 1, ".github/workflows/ci.yml:7: timeout-minutes 12 cites no observed maximum and date",
				"add a comment above it such as `# Observed max 4m12s on 2026-11-08.`")
		}},
		{"a workflow job whose long timeout cites its observed maximum and date, and a short one without a comment", func(t *testing.T) {
			t.Parallel()
			out, code := timeoutsCheck(t, map[string]string{
				".github/workflows/ci.yml":   timeoutJob("    # Observed max 5m15s on 2026-11-05.\n", "11"),
				".github/workflows/lint.yml": timeoutJob("", "3"),
			})
			sees(t, out, code, 0, "All 1 checks passed")
			passedWith(t, out, "workflow-timeouts", "2")
		}},
		{"long timeouts that are quoted, cite hours, cite on their own line, or cite an impossible date", func(t *testing.T) {
			t.Parallel()
			out, code := timeoutsCheck(t, map[string]string{
				".github/workflows/quoted.yml":     timeoutJob("", `"30"`),
				".github/workflows/bad-date.yml":   timeoutJob("    # Observed max 4m12s on 2026-13-45.\n", "10"),
				".github/workflows/hours.yml":      timeoutJob("    # observed max 1h12m on 2026-11-08.\n", "150"),
				".github/workflows/same-line.yml":  timeoutJob("", "12 # Observed max 5m on 2026-11-08."),
				".github/workflows/expression.yml": timeoutJob("", "${{ inputs.minutes }}"),
			})
			sees(t, out, code, 1, ".github/workflows/quoted.yml:6: timeout-minutes 30 cites no observed maximum and date",
				".github/workflows/bad-date.yml:7: timeout-minutes 10 cites no observed maximum and date")
			for _, passing := range []string{"hours.yml", "same-line.yml", "expression.yml"} {
				if strings.Contains(out, passing+":") {
					t.Errorf("workflow-timeouts must pass %s:\n%s", passing, out)
				}
			}
		}},
	}.run(t)
}

func Test0004_7(t *testing.T) {
	cases{
		{"a required check that cannot judge, named in the config or on the command line", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", cannotJudge("")), nil, "--all")
			sees(t, out, code, 0, "check ended without a result (exit status 3)", "1 info")
			for _, run := range []struct {
				config string
				args   []string
			}{
				{`, "requiredChecks": ["house-rule"]`, []string{"--all"}},
				{"", []string{"--require=house-rule", "--all"}},
				{`, "requiredChecks": ["cspell"]`, []string{"--require=house-rule,cspell", "--all"}},
			} {
				out, code := fitness(t, example(t, "happyRepo", cannotJudge(run.config)), nil, run.args...)
				sees(t, out, code, 1, "required check could not judge, so the run fails", "check ended without a result (exit status 3)", "1 failed")
				if !strings.Contains(row(out, "house-rule"), "failed") {
					t.Errorf("a required check that cannot judge must show failed, got row %q", row(out, "house-rule"))
				}
			}
		}},
		{"a required name that is no check, from either source", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", cannotJudge(`, "requiredChecks": ["no-such-check"]`)), nil, "--all")
			sees(t, out, code, 1, `requiredChecks: unknown or empty check "no-such-check"; the known checks are `, "house-rule")
			out, code = fitness(t, example(t, "happyRepo", nil), nil, "--require=no-such-check", "--all")
			sees(t, out, code, 1, `--require: unknown or empty check "no-such-check"; the known checks are `, "cspell")
		}},
		{"a required check the run leaves out or the config turns off", func(t *testing.T) {
			t.Parallel()
			out, code := fitness(t, example(t, "happyRepo", nil), nil, "--policy=external", "--checks=markdown-links", "--require=cspell")
			sees(t, out, code, 1, "--require names cspell, which this run leaves out")
			config := `{"policy": "external", "disabledChecks": ["cspell"], "requiredChecks": ["cspell"]}` + "\n"
			out, code = fitness(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": config}), nil, "--all")
			sees(t, out, code, 1, "requiredChecks names cspell, which disabledChecks turns off")
		}},
	}.run(t)
}

// cannotJudge is a repo whose local check house-rule exits 3 without a
// result, with more appended to its config's keys.
func cannotJudge(more string) map[string]string {
	return map[string]string{
		".fitnessrc.json":                 `{"localChecks": "checks"` + more + "}\n",
		"checks/fitness-check-house-rule": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"house-rule\"}'; exit 0; }\nexit 3\n",
	}
}
