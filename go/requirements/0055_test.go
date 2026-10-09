package requirements

import (
	"strings"
	"testing"
)

// repeatedStringLiterals runs `fitness-install -- repeated-string-literals`
// on happyRepo with files written over it.
func repeatedStringLiterals(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "repeated-string-literals")
}

// rslLines returns n source lines that each use the literal value.
func rslLines(value string, n int) string {
	return strings.Repeat("use('"+value+"');\n", n)
}

func Test0055_1(t *testing.T) {
	t.Parallel()
	out, code := repeatedStringLiterals(t, map[string]string{
		"src/a.ts": rslLines("active", 2),
		"src/b.ts": rslLines("active", 1),
	})
	sees(t, out, code, 1, `"active" appears 3 times (src/a.ts:1, src/a.ts:2, src/b.ts:1) — extract a shared constant`)
}

func Test0055_2(t *testing.T) {
	t.Parallel()
	out, code := repeatedStringLiterals(t, map[string]string{"src/a.ts": rslLines("active", 2)})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0055_3(t *testing.T) {
	t.Parallel()
	src := "import x from 'active';\nconst y = require('active');\nexport { z } from 'active';\n" +
		"// 'active'\n/* 'active' */\nconst re = /'active'/g;\nconst s = `'active'`;\n"
	out, code := repeatedStringLiterals(t, map[string]string{"src/a.ts": src})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0055_4(t *testing.T) {
	t.Parallel()
	src := rslLines("ok", 3) + rslLines("utf8", 3) + rslLines("inherit", 3) + rslLines("object", 3) + rslLines("use strict", 3)
	out, code := repeatedStringLiterals(t, map[string]string{"src/a.js": src})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0055_5(t *testing.T) {
	t.Parallel()
	out, code := repeatedStringLiterals(t, map[string]string{"src/a.ts": rslLines("active", 7)})
	sees(t, out, code, 1, `"active" appears 7 times (src/a.ts:1, src/a.ts:2, src/a.ts:3, src/a.ts:4, src/a.ts:5, +2 more)`)
}

func Test0055_6(t *testing.T) {
	t.Parallel()
	out, code := repeatedStringLiterals(t, map[string]string{"src/a.ts": rslLines("twice", 3) + rslLines("thrice", 4)})
	sees(t, out, code, 1, `"thrice" appears 4 times`, `"twice" appears 3 times`)
	if strings.Index(out, `"thrice"`) > strings.Index(out, `"twice"`) {
		t.Errorf("want thrice (4) reported before twice (3):\n%s", out)
	}
}

func Test0055_7(t *testing.T) {
	t.Parallel()
	out, code := repeatedStringLiterals(t, map[string]string{
		"src/a.test.ts":  rslLines("active", 2),
		"src/b.bench.ts": rslLines("active", 1),
	})
	sees(t, out, code, 1, `"active" appears 3 times (src/a.test.ts:1, src/a.test.ts:2, src/b.bench.ts:1)`)
}

func Test0055_8(t *testing.T) {
	t.Parallel()
	out, code := repeatedStringLiterals(t, map[string]string{
		".fitnessrc.json": `{"repeatedStringLiterals": {"allow": ["active"]}}` + "\n",
		"src/a.ts":        rslLines("active", 3),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}
