package requirements

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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

func Test0102_1(t *testing.T) {
	t.Parallel()
	var view configView
	stdoutOf(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": configured}), &view, "config", "--json")
	seesEntry(t, view.Settings, configSetting{Key: "proseBudget.maxWords", Value: float64(200), Source: "file"})
	seesEntry(t, view.Settings, configSetting{Key: "timeoutMs", Value: float64(5000), Source: "default"})
	seesEntry(t, view.Checks, checkState{Name: "cspell", Enabled: false, Reason: "disabledChecks in .fitnessrc.json turns it off"})
	seesEntry(t, view.Checks, checkState{Name: "markdown-links", Enabled: true, Reason: "external policy runs every check"})
}

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
	Closed      *bool                `json:"additionalProperties"`
}

// printedSchema runs `fitness config schema` in a repo and decodes it.
func printedSchema(t *testing.T) schemaKey {
	t.Helper()
	var schema schemaKey
	stdoutOf(t, example(t, "happyRepo", nil), &schema, "config", "schema")
	return schema
}

func Test0102_2(t *testing.T) {
	t.Parallel()
	schema := printedSchema(t)
	if !refusesUnknownKeys(schema) {
		t.Errorf("the schema must describe an object that refuses unknown keys, got %+v", schema)
	}
	got := schema.Properties["proseBudget"].Properties["maxWords"]
	if want := (schemaKey{Type: "integer", Description: got.Description, Default: float64(300)}); !reflect.DeepEqual(got, want) {
		t.Errorf("proseBudget.maxWords must give its type and its default 300, got %+v", got)
	}
	if got.Description == "" {
		t.Error("proseBudget.maxWords must carry a description")
	}
}

// refusesUnknownKeys reports whether schema is an object that allows no
// key it does not list.
func refusesUnknownKeys(schema schemaKey) bool {
	return schema.Type == "object" && schema.Closed != nil && !*schema.Closed
}

func Test0102_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"disabledChecks": ["cspell"]}` + "\n"})
	out, code := fitness(t, repo, nil, "config")
	sees(t, out, code, 1, "A run stops before any check:", "org policy runs every check as the shared bar")
}

func Test0102_4(t *testing.T) {
	t.Parallel()
	described := describedKeys("", printedSchema(t))
	for _, key := range keysFitnessReads(t) {
		if described[key] == "" {
			t.Errorf("the code reads config key %s, which the schema does not describe", key)
		}
	}
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
