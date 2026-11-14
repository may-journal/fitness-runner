package conf

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// section is one package's config keys: a defaults value of the reading type
// and the Go source whose doc comments describe each key.
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

// Section is a registered package's keys in the config file. Loading keys
// takes a Section, and only Register makes one, so no code can read keys
// the schema does not describe.
type Section[T any] struct{}

// Register declares the keys a package reads from the config file, panicking
// when another section already declares a key.
func Register[T any](defaults T, source []byte) *Section[T] {
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
	return &Section[T]{}
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
