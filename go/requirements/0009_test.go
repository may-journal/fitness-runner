package requirements

import (
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/acceptance"
)

func Test0009_5(t *testing.T) {
	cases{
		{"a Go test with a t.Run case beside a Vitest test with a case row: the shared scan lists each test's cases under it", func(t *testing.T) {
			t.Parallel()
			out := acceptance.SharedScan(t, map[string]string{
				"cases/0001_test.go":  "package cases\n\nimport \"testing\"\n\nfunc Test0001_1(t *testing.T) {\n\tt.Run(\"0001.1.1 installs\", nil)\n}\n",
				"tests/cases.test.ts": "it('0002.1 saves', () => {\n  run([{ name: '0002.1.1 saves' }])\n})\n",
			})
			acceptance.Wants(t, out,
				`"ID":"0001.1","Cases":[{"Name":"0001.1.1 installs","Step":"0001.1.1","Loc":"cases/0001_test.go:6"}]`,
				`"ID":"0002.1","Cases":[{"Name":"0002.1.1 saves","Step":"0002.1.1","Loc":"tests/cases.test.ts:2"}]`)
		}},
	}.run(t)
}
