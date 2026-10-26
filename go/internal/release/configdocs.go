package release

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// configKey is one .fitnessrc.json key a check reads, as its Go struct
// field declares it with a `check:"<name>"` tag.
type configKey struct {
	// Check is the check that reads the key.
	Check string
	// Key is the field's JSON key.
	Key string
	// Doc is the field's doc comment.
	Doc string
	// Fields are the nested keys, when the field is a struct.
	Fields []configKey
}

// configKeys reads every struct field tagged `check:"<name>"` in the Go
// sources under go/internal, by check name. A tagged field or one of its
// nested fields without a doc comment is an error, so every key a check
// reads is explained.
func (c Config) configKeys() (map[string][]configKey, error) {
	keys := map[string][]configKey{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(filepath.Join(c.Root, "go", "internal"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !packageSource(entry) {
			return err
		}
		return c.readTagged(fset, path, keys)
	})
	return keys, err
}

// readTagged adds the check-tagged fields of the Go file at path to keys,
// parsing only a file whose text holds such a tag.
func (c Config) readTagged(fset *token.FileSet, path string, keys map[string][]configKey) error {
	src, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(src, []byte(`check:"`)) {
		return err
	}
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return err
	}
	return taggedFields(fset, file, keys)
}

// structOf is the struct type t names directly, or as the element of a
// slice, array, pointer, or map value, so a list of objects shows its keys.
func structOf(t ast.Expr) (*ast.StructType, bool) {
	switch t := t.(type) {
	case *ast.StructType:
		return t, true
	case *ast.ArrayType:
		return structOf(t.Elt)
	case *ast.StarExpr:
		return structOf(t.X)
	case *ast.MapType:
		return structOf(t.Value)
	}
	return nil, false
}

// taggedFields adds each check-tagged field in file to keys.
func taggedFields(fset *token.FileSet, file *ast.File, keys map[string][]configKey) error {
	var err error
	ast.Inspect(file, func(n ast.Node) bool {
		field, ok := n.(*ast.Field)
		if !ok || err != nil {
			return err == nil
		}
		check := tag(field, "check")
		if check == "" {
			return true
		}
		var key configKey
		key, err = readKey(fset, field)
		key.Check = check
		keys[check] = append(keys[check], key)
		return false
	})
	return err
}

// readKey reads a field's JSON key, doc comment, and nested keys.
func readKey(fset *token.FileSet, field *ast.Field) (configKey, error) {
	key := configKey{Key: tag(field, "json"), Doc: fieldDoc(field)}
	if key.Doc == "" {
		return key, fmt.Errorf("%s: config key %q has no doc comment", fset.Position(field.Pos()), key.Key)
	}
	nested, ok := structOf(field.Type)
	if !ok {
		return key, nil
	}
	for _, inner := range nested.Fields.List {
		sub, err := readKey(fset, inner)
		if err != nil {
			return key, err
		}
		key.Fields = append(key.Fields, sub)
	}
	return key, nil
}

// fieldDoc is a field's doc comment, else its line comment, on one line,
// its Go name replaced by its JSON key.
func fieldDoc(field *ast.Field) string {
	text := field.Doc.Text()
	if text == "" {
		text = field.Comment.Text()
	}
	text = strings.Join(strings.Fields(text), " ")
	if len(field.Names) == 1 {
		if rest, ok := strings.CutPrefix(text, field.Names[0].Name+" "); ok {
			text = "`" + tag(field, "json") + "` " + rest
		}
	}
	return text
}

// tag is the name part of a field's struct tag key, or "".
func tag(field *ast.Field, key string) string {
	if field.Tag == nil {
		return ""
	}
	raw, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return ""
	}
	name, _, _ := strings.Cut(reflect.StructTag(raw).Get(key), ",")
	return name
}

// configMarkdown renders a check's config keys as a Markdown section, or ""
// when it reads none.
func configMarkdown(keys []configKey) string {
	if len(keys) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Configuration\n\nThese `.fitnessrc.json` keys tune the check.\n\n")
	for _, key := range keys {
		writeKey(&b, key, "")
	}
	return b.String()
}

// writeKey writes key as a list item at indent, then the keys nested in it
// one level deeper.
func writeKey(b *strings.Builder, key configKey, indent string) {
	b.WriteString(indent + "- " + keyLine(key) + "\n")
	for _, sub := range key.Fields {
		writeKey(b, sub, indent+"    ")
	}
}

// keyLine is a key's doc comment, led by the key itself when the comment
// does not already open with it.
func keyLine(key configKey) string {
	name := "`" + key.Key + "`"
	if strings.HasPrefix(key.Doc, name) {
		return key.Doc
	}
	return name + ": " + key.Doc
}

// renamed is key under the name path, with its doc comment's opening name
// changed to match.
func renamed(key configKey, path string) configKey {
	if rest, ok := strings.CutPrefix(key.Doc, "`"+key.Key+"`"); ok {
		key.Doc = "`" + path + "`" + rest
	}
	key.Key = path
	return key
}
