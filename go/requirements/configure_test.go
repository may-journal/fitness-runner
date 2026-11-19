package requirements

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// allPassedWith is the summary of an all-passing run with extra checks beyond
// the bundled catalog (negative when some are off).
func allPassedWith(extra int) string {
	return fmt.Sprintf("All %d checks passed", len(strings.Fields(catalogRows))+extra)
}

// disabling is a config that turns markdown-no-bold-italic off.
const disabling = `{"disabledChecks": ["markdown-no-bold-italic"]}` + "\n"

// passing is a local check's verdict that judged one file and found nothing.
const passing = `{"ok": true, "errors": [], "filesChecked": 1}`

// localCheck is a repo whose checks directory holds one local check, name,
// that follows the check protocol and prints verdict.
func localCheck(name, verdict string) map[string]string {
	return map[string]string{
		".fitnessrc.json":              `{"localChecks": "checks"}` + "\n",
		"checks/fitness-check-" + name: "#!/bin/sh\nif [ \"$1\" = --describe ]; then\n  echo '{\"name\": \"" + name + "\"}'\n  exit 0\nfi\necho '" + verdict + "'\n",
	}
}

// boldPR runs pr-check in repo on a PR whose description uses bold, closing
// a Plan that is done.
func boldPR(t *testing.T, repo string) (string, int) {
	t.Helper()
	body := strings.Replace(prPR(11), "This change is a test fixture.", "This change is a **bold** test fixture.", 1)
	plan := issuePR(t, 11, planPR("- [x] Write the tests."), "Plan")
	rp := standIn(t, map[string][]response{"gh": {plan, noChangelogPR}})
	pr := map[string]any{"number": 1, "title": "test(app): add one file", "body": body}
	return fitness(t, repo, append(rp.env, actionsEvent(t, "pull_request", map[string]any{"action": "opened", "pull_request": pr})...), "pr-check")
}

// configView is what `fitness config --json` prints, as an agent reads it.
type configView struct {
	Policy   string          `json:"policy"`
	Settings []configSetting `json:"settings"`
	Checks   []checkState    `json:"checks"`
}

// configSetting is one setting in that view.
type configSetting struct {
	Key    string `json:"key"`
	Value  any    `json:"value"`
	Source string `json:"source"`
}

// checkState is one check's entry in that view.
type checkState struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

// stdoutOf runs the one-liner with args in repo, decodes stdout apart from
// stderr as JSON into v, failing unless it exits 0.
func stdoutOf(t *testing.T, repo string, v any, args ...string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := started(t, func() *exec.Cmd {
		stdout.Reset()
		stderr.Reset()
		cmd := exec.Command(installer(t), append([]string{"--"}, args...)...)
		cmd.Dir, cmd.Env = repo, userEnv()
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		return cmd
	}, (*exec.Cmd).Run)
	if err := json.Unmarshal(stdout.Bytes(), v); err != nil || cmd.ProcessState.ExitCode() != 0 {
		t.Fatalf("fitness %v must exit 0 printing JSON, got exit %d, %v:\n%s%s", args, cmd.ProcessState.ExitCode(), err, &stdout, &stderr)
	}
}

// configured is a config that turns off cspell under external policy and
// sets one prose limit.
const configured = `{"policy": "external", "disabledChecks": ["cspell"], "proseBudget": {"maxWords": 200}}` + "\n"

// seesEntry asserts entries hold want, matched by its first field.
func seesEntry[T any](t *testing.T, entries []T, want T) {
	t.Helper()
	name := reflect.ValueOf(want).Field(0).Interface()
	for _, e := range entries {
		if reflect.ValueOf(e).Field(0).Interface() == name {
			if !reflect.DeepEqual(e, want) {
				t.Errorf("%v must show %+v, got %+v", name, want, e)
			}
			return
		}
	}
	t.Errorf("the view lists no %v", name)
}

// schemaKey is one property in the printed schema.
type schemaKey struct {
	Type        string               `json:"type"`
	Description string               `json:"description"`
	Default     any                  `json:"default"`
	Properties  map[string]schemaKey `json:"properties"`
	// Others is false for an object that refuses unknown keys, or the
	// schema of a map's values.
	Others json.RawMessage `json:"additionalProperties"`
}

// printedSchema runs `fitness config schema` in a repo and decodes it.
func printedSchema(t *testing.T) schemaKey {
	t.Helper()
	var schema schemaKey
	stdoutOf(t, example(t, "happyRepo", nil), &schema, "config", "schema")
	return schema
}

// mapOfStrings fails unless key is an object whose free keys hold strings,
// as checkVersions pins are.
func mapOfStrings(t *testing.T, key schemaKey) {
	t.Helper()
	if key.Type != "object" || strings.Join(strings.Fields(string(key.Others)), "") != `{"type":"string"}` {
		t.Errorf("checkVersions must be a map of string pins, got %+v", key)
	}
}

// refusesUnknownKeys reports whether schema is an object that allows no
// key it does not list.
func refusesUnknownKeys(schema schemaKey) bool {
	return schema.Type == "object" && string(schema.Others) == "false"
}

// describedKeys maps each key path in schema to its description.
func describedKeys(prefix string, schema schemaKey) map[string]string {
	out := map[string]string{}
	for name, p := range schema.Properties {
		path := strings.TrimPrefix(prefix+"."+name, ".")
		out[path] = p.Description
		for inner, doc := range describedKeys(path, p) {
			out[inner] = doc
		}
	}
	return out
}

// keysFitnessReads lists the JSON keys of each type a conf.Register call
// names, since only a registered Section loads keys.
func keysFitnessReads(t *testing.T) []string {
	t.Helper()
	var keys []string
	err := filepath.WalkDir("..", func(dir string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			keys = append(keys, packageKeys(t, dir)...)
		}
		return err
	})
	mustDo(t, err)
	if len(keys) == 0 {
		t.Fatal("found no conf.Register call in the source; the search is broken")
	}
	return keys
}

// packageKeys lists the keys each Register call in one package's source reads.
func packageKeys(t *testing.T, dir string) []string {
	t.Helper()
	types, calls := map[string]*ast.StructType{}, []string{}
	files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	for _, file := range files {
		if !strings.HasSuffix(file, "_test.go") {
			calls = append(calls, readSource(t, file, types)...)
		}
	}
	var keys []string
	for _, name := range calls {
		keys = append(keys, structKeys("", types[name], types)...)
	}
	return keys
}

// readSource records file's struct types in types and returns the type
// names its Register calls declare.
func readSource(t *testing.T, file string, types map[string]*ast.StructType) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	mustDo(t, err)
	var calls []string
	ast.Inspect(parsed, func(n ast.Node) bool {
		if spec, ok := n.(*ast.TypeSpec); ok {
			types[spec.Name.Name], _ = spec.Type.(*ast.StructType)
		}
		calls = append(calls, registerType(t, file, n)...)
		return true
	})
	return calls
}

// registerType returns the type a Register[T] call names, failing the test if
// T is inferred, which source search cannot read.
func registerType(t *testing.T, file string, n ast.Node) []string {
	if call, ok := n.(*ast.CallExpr); ok && strings.HasSuffix(exprName(call.Fun), "Register") {
		t.Errorf("%s: name the type a Register call reads, as conf.Register[T]", file)
	}
	index, ok := n.(*ast.IndexExpr)
	if !ok || !strings.HasSuffix(exprName(index.X), "Register") {
		return nil
	}
	return []string{exprName(index.Index)}
}

// exprName renders an identifier or selector as written, such as
// conf.Register, or "" for anything else.
func exprName(x ast.Expr) string {
	switch e := x.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprName(e.X) + "." + e.Sel.Name
	}
	return ""
}

// structKeys lists the key paths a struct reads, under prefix, following
// nested structs and the package's named struct types.
func structKeys(prefix string, st *ast.StructType, types map[string]*ast.StructType) []string {
	if st == nil {
		return nil
	}
	var keys []string
	for _, field := range st.Fields.List {
		key := jsonTag(field)
		if key == "" {
			continue
		}
		path := strings.TrimPrefix(prefix+"."+key, ".")
		keys = append(append(keys, path), structKeys(path, nestedStruct(field.Type, types), types)...)
	}
	return keys
}

// nestedStruct is the struct a field's type spells out or names, if any.
func nestedStruct(x ast.Expr, types map[string]*ast.StructType) *ast.StructType {
	if st, ok := x.(*ast.StructType); ok {
		return st
	}
	return types[exprName(x)]
}

// jsonTag is a field's JSON key, or "" when it has none.
func jsonTag(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`")).Get("json")
	key, _, _ := strings.Cut(tag, ",")
	return key
}

// unitTested asserts that go test in repo's fitness folder passes the check's
// fixture test and fails its repo test on the finding.
func unitTested(t *testing.T, repo string) {
	t.Helper()
	out, failed := goTestFitness(t, repo)
	if !failed || !strings.Contains(out, "notes.md: holds a TODO") || strings.Contains(out, "FAIL: TestFixture") {
		t.Errorf("go test in fitness/ must pass the fixture test and fail on the repo's finding:\n%s", out)
	}
}

// goTestFitness runs `go -C fitness test ./...` in repo, as its developer does, and
// returns what it printed and whether it failed.
func goTestFitness(t *testing.T, repo string) (string, bool) {
	t.Helper()
	cmd := exec.Command("go", "-C", "fitness", "test", "./...")
	cmd.Dir, cmd.Env = repo, userEnv()
	out, err := cmd.CombinedOutput()
	return string(out), err != nil
}

// houseReadme is a folder check's README: a title, the rule, and the fix.
const houseReadme = "# House rule\n\nKeeps the house in order. Each room stays tidy.\n\n## Fix\n\nTidy the room the error names.\n"

// shellCheck is a repo with one shell check, name, and a README.
// The check answers --describe with describe, else ignores the handshake, and prints verdict; an empty verdict hangs.
func shellCheck(name, describe, verdict string) map[string]string {
	script := verdictScript(verdict)
	if describe != "" {
		script = strings.Replace(script, "#!/bin/sh\n", "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '"+describe+"'; exit 0; }\n", 1)
	}
	return map[string]string{"fitness/" + name + "/README.md": houseReadme, "fitness/" + name + "/check": script}
}

// verdictScript is a check program that prints verdict, or hangs when it
// is empty.
func verdictScript(verdict string) string {
	if verdict == "" {
		return "#!/bin/sh\nexec /bin/sleep 30\n"
	}
	return "#!/bin/sh\necho '" + verdict + "'\n"
}

// goCheck is a repo with one Go check, built on this checkout's SDK, that fails
// each Markdown file holding a TODO. marker goes into its source, so a test that
// needs a cold build gets source no other test built.
func goCheck(t *testing.T, name, marker string) map[string]string {
	t.Helper()
	sdk, err := filepath.Abs("..")
	mustDo(t, err)
	return map[string]string{
		"fitness/go.mod": "module example.com/repo/fitness\n\ngo 1.26\n\nrequire github.com/may-journal/fitness-runner/go v0.0.0\n\n" +
			"replace github.com/may-journal/fitness-runner/go => " + sdk + "\n",
		"fitness/" + name + "/README.md": "# No to-do markers\n\nFails a Markdown file that holds a to-do marker.\n",
		"fitness/" + name + "/main.go":   goRuleSource(marker),
	}
}

// goRuleSource is the Go check goCheck writes, with marker in a comment.
func goRuleSource(marker string) string {
	return "// Package main is a check built from " + marker + ".\npackage main\n\n" +
		"import (\n\t\"strings\"\n\n\t\"github.com/may-journal/fitness-runner/go/check\"\n)\n\n" +
		"var rule = check.Check{Run: func(ctx *check.Context) (check.Result, error) {\n" +
		"\tfiles := ctx.InScope(ctx.Files(\".md\"))\n\tvar found []check.Finding\n" +
		"\tfor _, f := range files {\n\t\ttext, err := ctx.Read(f)\n\t\tif err != nil {\n\t\t\treturn check.Result{}, err\n\t\t}\n" +
		"\t\tif strings.Contains(text, \"TODO\") {\n\t\t\tfound = append(found, check.Finding{File: f, Rule: \"no-todo\", Message: \"holds a TODO\", Fix: \"do it\"})\n\t\t}\n\t}\n" +
		"\treturn check.Found(len(files), found...), nil\n}}\n\n" +
		"func main() { check.Main(rule) }\n"
}

// timeoutJob is a workflow whose job sets timeout-minutes to minutes, with
// comment on the line above it, line 6.
func timeoutJob(comment, minutes string) string {
	return "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n" + comment +
		"    timeout-minutes: " + minutes + "\n    steps:\n      - run: make\n"
}

// timeoutsCheck runs this repo's workflow-timeouts check, built on this
// checkout's SDK, on happyRepo with workflows written over it.
func timeoutsCheck(t *testing.T, workflows map[string]string) (string, int) {
	t.Helper()
	sdk, err := filepath.Abs("..")
	mustDo(t, err)
	files := map[string]string{"fitness/go.mod": "module example.com/repo/fitness\n\ngo 1.26\n\n" +
		"require github.com/may-journal/fitness-runner/go v0.0.0\n\nreplace github.com/may-journal/fitness-runner/go => " + sdk + "\n"}
	for _, name := range []string{"main.go", "README.md"} {
		body, err := os.ReadFile(filepath.Join(sdk, "..", "fitness", "workflow-timeouts", name))
		mustDo(t, err)
		files["fitness/workflow-timeouts/"+name] = string(body)
	}
	for name, body := range workflows {
		files[name] = body
	}
	return fitness(t, example(t, "happyRepo", files), nil, "check", "workflow-timeouts")
}
