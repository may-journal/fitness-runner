package release

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// configFile declares Config, the shape of .fitnessrc.json, relative to the
// repository root.
const configFile = "go/pkg/conf/conf.go"

// configKeys reads the JSON keys of Config's fields, in declaration order:
// the config file's top-level engine keys.
func (c Config) configKeys() ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(c.Root, filepath.FromSlash(configFile)), nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var keys []string
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "Config" {
			return true
		}
		keys = structKeys(spec.Type)
		return false
	})
	if len(keys) == 0 {
		return nil, fmt.Errorf("%s: no Config struct with JSON keys", configFile)
	}
	return keys, nil
}

// jsonKey is the name in a field's json tag, or "".
func jsonKey(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	raw, _ := strconv.Unquote(field.Tag.Value)
	name, _, _ := strings.Cut(reflect.StructTag(raw).Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// structKeys are the JSON keys of a struct type's fields, or none for any
// other type.
func structKeys(t ast.Expr) []string {
	st, ok := t.(*ast.StructType)
	if !ok {
		return nil
	}
	var keys []string
	for _, field := range st.Fields.List {
		if key := jsonKey(field); key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}
