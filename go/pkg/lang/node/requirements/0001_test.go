// Package requirements proves the Node module's docs/requirements as a user
// meets them. Each test runs the installer at FITNESS_INSTALL in a copied
// example repo.
package requirements

import (
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/acceptance"
)

func TestMain(m *testing.M) {
	acceptance.Main(m)
}

func Test0001_1(t *testing.T) {
	acceptance.Cases{
		{"case table rows with plain, quoted and template keys, computed names, other keys, a ternary, objects outside an array, a nested object and a test without an ID: the check sees each case name, then its location, then the step a name opens with", func(t *testing.T) {
			t.Parallel()
			suite := "describe('0001 Install', () => {\n  it('0001.1 installs', () => {\n    const rows = [\n" +
				"      { name: '0001.1.1 installs', want: 1 },\n      { \"name\": \"keyed\" },\n      { want: 2, name: `0001.1.2: raw` },\n" +
				"      { name: `built ${n}` }, { name: label }, { name: 'a' + b }, { id: 'no name' },\n    ]\n" +
				"    const shown = ok ? name : 'not a case'\n" +
				"    screen.getByRole('button', { name: 'Save' }); expect(user).toEqual({ name: 'Ann' }); run([{ name: 'row', input: { name: 'inner' } }])\n  })\n" +
				"  it('helper', () => {\n    run([{ name: '0001.1.3 outside' }])\n  })\n})\n"
			out := acceptance.SharedScan(t, map[string]string{"tests/cases.test.ts": suite})
			at := func(line string) string { return `"Loc":"tests/cases.test.ts:` + line + `"` }
			acceptance.Wants(t, out, `"ID":"0001.1","Cases":[`+
				`{"Name":"0001.1.1 installs","Step":"0001.1.1",`+at("4")+`},`+
				`{"Name":"keyed",`+at("5")+`},`+
				`{"Name":"0001.1.2: raw","Step":"0001.1.2",`+at("6")+`},`+
				`{"Name":"row",`+at("10")+`}],`+at("2"),
				`"Name":"it(\"helper\")","ID":"",`+at("12"))
		}},
	}.Run(t)
}

func Test0001_2(t *testing.T) {
	acceptance.Cases{
		{"a Vitest suite with a describe, a nested describe.each, an it.each, a test outside any group and a computed title: the check sees each describe group and each table is marked", func(t *testing.T) {
			t.Parallel()
			suite := "describe('0001 Install', () => {\n  it('0001.1 installs', () => {})\n  describe.each(rows)('nested %s', () => {\n" +
				"    it.each(rows)('0001.2 adds %d', () => {})\n  })\n})\ntest('0002.1 alone', () => {})\n" +
				"describe('save ' + mode, () => {\n  it('0003.1 saves', () => {})\n})\n"
			out := acceptance.SharedScan(t, map[string]string{"tests/service.test.ts": suite})
			at := func(line string) string { return `"Loc":"tests/service.test.ts:` + line + `"` }
			install := `{"Title":"0001 Install",` + at("1")
			acceptance.Wants(t, out,
				`"ID":"0001.1",`+at("2")+`,"File":"tests/service.test.ts","Groups":[`+install+`}],"Module"`,
				`"Groups":[`+install+`},{"Title":"nested %s",`+at("3")+`,"Table":true}],"Table":true,"Module"`,
				`"ID":"0002.1",`+at("7")+`,"File":"tests/service.test.ts","Module"`,
				`"Groups":[{"Title":"",`+at("8")+`}],"Module"`)
		}},
	}.Run(t)
}
