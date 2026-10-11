package architecture

import (
	"regexp"
	"strings"
)

var semanticID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

var fields = strings.Fields("id ref type parent path symbol name aliases owns requirements from to kind mode evidence target")

func (m *Model) declaration(text string, at Location) Declaration {
	values := map[string]string{}
	for _, field := range strings.Split(strings.Trim(text, "`"), ";") {
		m.field(values, strings.TrimSpace(field), at)
	}
	if values["ref"] != "" && len(values) != 1 {
		m.fail(at, "ref reuses a declaration; do not add fields")
	}
	d := Declaration{
		ID: values["id"], Ref: values["ref"], Type: values["type"], Parent: values["parent"],
		Path: values["path"], Symbol: values["symbol"], Name: values["name"],
		Aliases: list(values["aliases"]), Owns: list(values["owns"]), Requirements: list(values["requirements"]),
		From: values["from"], To: values["to"], Kind: values["kind"], Mode: values["mode"],
		Evidence: values["evidence"], Target: values["target"], At: at,
	}
	m.identity(d)
	return d
}

func (m *Model) field(values map[string]string, text string, at Location) {
	key, value, ok := strings.Cut(text, "=")
	key, value = strings.TrimSpace(key), strings.TrimSpace(value)
	if !ok || value == "" {
		m.fail(at, "malformed contract field %q; use key=value", text)
		return
	}
	if !contains(fields, key) {
		m.fail(at, "unsupported contract field %q", key)
	}
	if _, exists := values[key]; exists {
		m.fail(at, "duplicate contract field %q", key)
	}
	values[key] = value
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (m *Model) identity(d Declaration) {
	if (d.ID == "") == (d.Ref == "") {
		m.fail(d.At, "declare exactly one stable id or ref")
	}
	for _, id := range []string{d.ID, d.Ref, d.Parent, d.From, d.To} {
		if id != "" && !semanticID.MatchString(id) {
			m.fail(d.At, "invalid stable ID %q", id)
		}
	}
}
