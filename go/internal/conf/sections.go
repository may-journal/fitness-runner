package conf

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// section is one package's keys in the config file: a value of the Go type
// that reads them, holding each key's default, and the Go source declaring
// that type, whose doc comments describe each key.
type section struct {
	defaults reflect.Value
	source   []byte
}

// sections holds every registered section by its type, in the order each
// registered.
var sections = struct {
	sync.Mutex
	order  []reflect.Type
	byType map[reflect.Type]section
}{byType: map[reflect.Type]section{}}

// Register declares the keys a package reads from the config file with
// LoadAs: their Go type T, a T holding each key's default, and the Go source
// that declares T. The schema, `fitness config`, and its descriptions all
// come from these, so a package calls Register from an init function before
// it ever loads its keys. A key that another section already declares panics,
// since one key must mean one thing.
func Register[T any](defaults T, source []byte) {
	t := reflect.TypeFor[T]()
	sections.Lock()
	defer sections.Unlock()
	for _, other := range sections.order {
		if key, clash := sharedKey(t, other); clash {
			panic(fmt.Sprintf("conf: %v and %v both declare the %q key", t, other, key))
		}
	}
	if _, done := sections.byType[t]; !done {
		sections.order = append(sections.order, t)
	}
	sections.byType[t] = section{defaults: reflect.ValueOf(defaults), source: source}
}

// registered reports whether T's keys were registered.
func registered[T any]() error {
	t := reflect.TypeFor[T]()
	sections.Lock()
	defer sections.Unlock()
	if _, ok := sections.byType[t]; !ok {
		return fmt.Errorf("conf: %v reads %s keys no section declares; call conf.Register for it, so the schema lists them", t, FileName)
	}
	return nil
}

// registeredSections returns every section, in registration order.
func registeredSections() []section {
	sections.Lock()
	defer sections.Unlock()
	out := make([]section, len(sections.order))
	for i, t := range sections.order {
		out[i] = sections.byType[t]
	}
	return out
}

// sharedKey returns a top-level key both types declare, if any.
func sharedKey(a, b reflect.Type) (string, bool) {
	if a == b {
		return "", false
	}
	keys := map[string]bool{}
	for _, f := range jsonFields(b) {
		keys[f.key] = true
	}
	for _, f := range jsonFields(a) {
		if keys[f.key] {
			return f.key, true
		}
	}
	return "", false
}

// jsonField is one struct field the config file sets: its JSON key and its
// Go field.
type jsonField struct {
	key   string
	field reflect.StructField
}

// jsonFields lists t's fields that carry a JSON key, in declaration order.
func jsonFields(t reflect.Type) []jsonField {
	var out []jsonField
	for i := range t.NumField() {
		f := t.Field(i)
		if key := jsonKey(f); key != "" {
			out = append(out, jsonField{key, f})
		}
	}
	return out
}

// jsonKey is the field's key in the config file, or "" for none.
func jsonKey(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" || tag == "-" || !f.IsExported() {
		return ""
	}
	key, _, _ := strings.Cut(tag, ",")
	return key
}
