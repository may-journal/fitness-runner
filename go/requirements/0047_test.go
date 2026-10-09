package requirements

import "testing"

// nodeVersionCheck runs `fitness-install -- node-version` on happyRepo with
// files written over it and env added.
func nodeVersionCheck(t *testing.T, files map[string]string, env []string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, "node-version")
}

// nodeVersionRepo is a JavaScript repo whose `.nvmrc` holds nvmrc.
func nodeVersionRepo(nvmrc string) map[string]string {
	return map[string]string{"package.json": "{}\n", ".nvmrc": nvmrc}
}

func Test0047_1(t *testing.T) {
	t.Parallel()
	out, code := nodeVersionCheck(t, nil, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "node-version")
}

func Test0047_2(t *testing.T) {
	t.Parallel()
	out, code := nodeVersionCheck(t, map[string]string{"package.json": "{}\n"}, nil)
	sees(t, out, code, 1, "missing .nvmrc")
}

func Test0047_3(t *testing.T) {
	t.Parallel()
	out, code := nodeVersionCheck(t, nodeVersionRepo("1\n"), nil)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0047_4(t *testing.T) {
	t.Parallel()
	out, code := nodeVersionCheck(t, nodeVersionRepo("v1.2.3\n"), nil)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0047_5(t *testing.T) {
	t.Parallel()
	out, code := nodeVersionCheck(t, nodeVersionRepo("999\n"), nil)
	sees(t, out, code, 1, "does not satisfy .nvmrc (requires 999.x). Run: nvm use")
}

func Test0047_6(t *testing.T) {
	t.Parallel()
	out, code := nodeVersionCheck(t, nodeVersionRepo("latest\n"), nil)
	sees(t, out, code, 1, "does not satisfy .nvmrc (requires NaN.x). Run: nvm use")
}

func Test0047_7(t *testing.T) {
	t.Parallel()
	out, code := nodeVersionCheck(t, nodeVersionRepo("1\n"), []string{pathWithout(t, "node")})
	sees(t, out, code, 1, "node not found on PATH. Install Node.js")
}
