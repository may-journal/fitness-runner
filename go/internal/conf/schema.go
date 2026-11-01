package conf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"github.com/may-journal/fitness-runner/go/internal/comments"
)

// Key is one key the config file may set, as a registered section declares it.
type Key struct {
	// Path is the key's dotted JSON path, such as proseBudget.maxWords.
	Path string `json:"key"`
	// Type is the key's JSON type: string, integer, number, boolean,
	// array, or object.
	Type string `json:"type"`
	// Items is the JSON type of an array's items.
	Items string `json:"items,omitempty"`
	// Enum lists the values a string key accepts, from its enum tag.
	Enum []string `json:"enum,omitempty"`
	// Description is the doc comment on the key's Go field.
	Description string `json:"description"`
	// Default is the value the key takes when the file leaves it unset,
	// or nil for none.
	Default any `json:"default,omitempty"`
	// Keys are an object's own keys.
	Keys []Key `json:"keys,omitempty"`
}

// Keys lists every key the registered sections declare, in registration
// and declaration order, with the doc of the type that holds them.
func Keys() ([]Key, error) {
	var out []Key
	for _, s := range registeredSections() {
		docs, err := comments.Types(s.source)
		if err != nil {
			return nil, err
		}
		out = append(out, keysOf(s.defaults, s.defaults.Type().Name(), "", docs)...)
	}
	return out, nil
}

// keysOf lists the keys of struct value v, whose fields' docs sit under
// docPrefix and whose keys sit under pathPrefix.
func keysOf(v reflect.Value, docPrefix, pathPrefix string, docs map[string]string) []Key {
	var out []Key
	for _, f := range jsonFields(v.Type()) {
		path := f.key
		if pathPrefix != "" {
			path = pathPrefix + "." + f.key
		}
		out = append(out, keyOf(v.FieldByIndex(f.field.Index), f.field, docPrefix+"."+f.field.Name, path, docs))
	}
	return out
}

// keyOf describes one field, listing an object's own keys and a default when
// it is not the zero value.
func keyOf(v reflect.Value, field reflect.StructField, docKey, path string, docs map[string]string) Key {
	k := Key{Path: path, Type: jsonType(field.Type), Enum: enum(field), Description: describe(field.Name, docs[docKey])}
	switch field.Type.Kind() {
	case reflect.Struct:
		k.Keys = keysOf(v, structDocs(field.Type, docKey), path, docs)
	case reflect.Slice:
		k.Items = jsonType(field.Type.Elem())
	}
	if k.Keys == nil && !v.IsZero() {
		k.Default = v.Interface()
	}
	return k
}

// structDocs returns where a struct field's own fields keep their docs: the
// type name, else the field.
func structDocs(t reflect.Type, docKey string) string {
	if t.Name() != "" {
		return t.Name()
	}
	return docKey
}

// describe turns a Go field's doc comment into a key's description by dropping
// the leading field name and "is".
func describe(name, doc string) string {
	rest, ok := strings.CutPrefix(doc, name+" ")
	if !ok || rest == "" {
		return doc
	}
	if after, is := strings.CutPrefix(rest, "is "); is && after != "" {
		rest = after
	}
	r := []rune(rest)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// enum lists the values a field's enum tag allows, or nil for any.
func enum(field reflect.StructField) []string {
	if tag := field.Tag.Get("enum"); tag != "" {
		return strings.Split(tag, ",")
	}
	return nil
}

// jsonTypes maps each Go kind a config key may have to its JSON Schema
// type; a struct is an object.
var jsonTypes = map[reflect.Kind]string{
	reflect.String: "string", reflect.Bool: "boolean", reflect.Int: "integer",
	reflect.Float64: "number", reflect.Slice: "array", reflect.Struct: "object",
}

// jsonType is the JSON Schema type of a Go type. A kind jsonTypes lacks
// panics, so a new key's type cannot slip into the schema unnamed.
func jsonType(t reflect.Type) string {
	name, ok := jsonTypes[t.Kind()]
	if !ok {
		panic(fmt.Sprintf("conf: no JSON Schema type for a %v key", t))
	}
	return name
}

// SchemaURL is the JSON Schema dialect the schema follows.
const SchemaURL = "https://json-schema.org/draft/2020-12/schema"

// Schema is a JSON Schema for the config file, built from the registered
// sections with one property per key.
func Schema() (map[string]any, error) {
	keys, err := Keys()
	if err != nil {
		return nil, err
	}
	docs, err := comments.Types(source)
	if err != nil {
		return nil, err
	}
	schema := object(keys)
	schema["$schema"] = SchemaURL
	schema["title"] = FileName
	schema["description"] = describe("Config", docs["Config"])
	schema["properties"].(map[string]any)["$schema"] = map[string]any{
		"type": "string", "description": "The URL of this schema, for an editor that checks the file as you type.",
	}
	return schema, nil
}

// object is the schema of an object holding keys, which allows no other.
func object(keys []Key) map[string]any {
	props := map[string]any{}
	for _, k := range keys {
		props[lastPart(k.Path)] = property(k)
	}
	return map[string]any{"type": "object", "properties": props, "additionalProperties": false}
}

// property is one key's schema.
func property(k Key) map[string]any {
	p := map[string]any{"type": k.Type}
	if k.Type == "object" {
		p = object(k.Keys)
	}
	p["description"] = k.Description
	if k.Items != "" {
		p["items"] = map[string]any{"type": k.Items}
	}
	if k.Enum != nil {
		p["enum"] = k.Enum
	}
	if k.Default != nil {
		p["default"] = k.Default
	}
	return p
}

// lastPart is the last name of a dotted path.
func lastPart(path string) string {
	return path[strings.LastIndex(path, ".")+1:]
}

// Setting is one key's value in a repo.
type Setting struct {
	// Key is the key's dotted JSON path.
	Key string `json:"key"`
	// Value is what the key holds: the file's value, else its default.
	Value any `json:"value"`
	// Source says where Value came from: file, default, or unset.
	Source string `json:"source"`
	// Description is the doc comment on the key's Go field.
	Description string `json:"description"`
}

// Settings lists each key in root's config file with its value and source,
// in the same order that Keys lists them.
func Settings(root string) ([]Setting, error) {
	if _, err := Load(root); err != nil {
		return nil, err
	}
	keys, err := Keys()
	if err != nil {
		return nil, err
	}
	file := map[string]any{}
	if raw, err := os.ReadFile(filepath.Join(root, FileName)); err == nil {
		_ = json.Unmarshal(raw, &file) // Load already parsed it
	}
	return settingsOf(keys, file), nil
}

// settingsOf flattens keys to their leaves, valued from the file's object.
func settingsOf(keys []Key, file map[string]any) []Setting {
	var out []Setting
	for _, k := range keys {
		value, set := file[lastPart(k.Path)]
		if k.Keys != nil {
			inner, _ := value.(map[string]any)
			out = append(out, settingsOf(k.Keys, inner)...)
			continue
		}
		out = append(out, setting(k, value, set))
	}
	return out
}

// setting values one leaf key: the file's value when it sets one, else
// the key's default, else unset.
func setting(k Key, value any, set bool) Setting {
	s := Setting{Key: k.Path, Value: value, Source: "file", Description: k.Description}
	if !set {
		s.Value, s.Source = k.Default, "default"
	}
	if !set && k.Default == nil {
		s.Source = "unset"
	}
	return s
}
