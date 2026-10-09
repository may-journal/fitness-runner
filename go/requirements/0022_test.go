package requirements

import (
	"strings"
	"testing"
)

// The real npm packages these tests declare. Each is retired upstream, so
// its latest published version no longer moves.
const (
	dcLeftPad  = "left-pad"        // latest 1.3.0
	dcIsNumber = "is-number"       // latest 7.0.0
	dcTypes    = "@types/left-pad" // latest 1.2.0
	dcInternal = "@mayjournal/fitness-shared"
)

// dcLead is the line that opens every failure.
const dcLead = "Dependencies behind their latest published version — update them (npm install <pkg>@latest) or pin intentionally:"

// dcUnreachable is a registry address nothing listens on.
const dcUnreachable = "registry=http://127.0.0.1:1\n"

// dependencyCurrency runs `fitness-install -- dependency-currency` on
// happyRepo with files written over it. Like a real repo, it tracks its
// manifests and ignores the installed node_modules.
func dependencyCurrency(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", with(map[string]string{".gitignore": "node_modules\n"}, files))
	return fitness(t, repo, env, "dependency-currency")
}

// dcInstalled is the node_modules manifest of name installed at version,
// under dir ("" for the repo root).
func dcInstalled(dir, name, version string) (string, string) {
	return dir + "node_modules/" + name + "/package.json", `{"name": "` + name + `", "version": "` + version + `"}` + "\n"
}

// dcRepo is a package.json declaring manifest plus each install given as
// dir, name, version triples.
func dcRepo(manifest string, installs ...string) map[string]string {
	files := map[string]string{"package.json": manifest + "\n"}
	for i := 0; i+2 < len(installs); i += 3 {
		path, body := dcInstalled(installs[i], installs[i+1], installs[i+2])
		files[path] = body
	}
	return files
}

func Test0022_1(t *testing.T) {
	out, code := dependencyCurrency(t, dcRepo(`{"dependencies": {"left-pad": "^1.3.0"}}`, "", dcLeftPad, "1.3.0"))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0022_2(t *testing.T) {
	files := dcRepo(`{"dependencies": {"left-pad": "^1.0.0"}, "devDependencies": {"is-number": "^6.0.0"}, "peerDependencies": {"@types/left-pad": "^1.0.0"}}`,
		"", dcLeftPad, "1.0.0", "", dcIsNumber, "6.0.0", "", dcTypes, "1.0.0")
	out, code := dependencyCurrency(t, files)
	sees(t, out, code, 1, dcLead, "left-pad: 1.0.0 → 1.3.0", "is-number: 6.0.0 → 7.0.0", "@types/left-pad: 1.0.0 → 1.2.0")
}

func Test0022_3(t *testing.T) {
	out, code := dependencyCurrency(t, dcRepo(`{"dependencies": {"left-pad": "^1.3.0"}}`))
	sees(t, out, code, 1, dcLead, "left-pad: missing → 1.3.0")
}

func Test0022_4(t *testing.T) {
	out, code := dependencyCurrency(t, dcRepo(`{"dependencies": {"`+dcInternal+`": "*"}}`))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0022_5(t *testing.T) {
	for _, workspaces := range []string{`["packages/*"]`, `{"packages": ["packages/a"]}`} {
		files := dcRepo(`{"workspaces": `+workspaces+`}`, "", dcLeftPad, "1.0.0")
		files["packages/a/package.json"] = `{"dependencies": {"left-pad": "^1.0.0"}}` + "\n"
		out, code := dependencyCurrency(t, files)
		sees(t, out, code, 1, dcLead, "left-pad: 1.0.0 → 1.3.0")
	}
}

func Test0022_6(t *testing.T) {
	files := dcRepo(`{"workspaces": ["packages/*"]}`, "", dcLeftPad, "1.0.0", "packages/a/", dcLeftPad, "1.1.0")
	for _, ws := range []string{"a", "b", "c"} {
		files["packages/"+ws+"/package.json"] = `{"devDependencies": {"left-pad": "^1.0.0"}}` + "\n"
	}
	out, code := dependencyCurrency(t, files)
	sees(t, out, code, 1, "left-pad: 1.1.0 → 1.3.0", "left-pad: 1.0.0 → 1.3.0")
	if n := strings.Count(read(out), "left-pad: 1.0.0 → 1.3.0"); n != 1 {
		t.Errorf("the shared install is listed %d times, want once:\n%s", n, out)
	}
}

func Test0022_7(t *testing.T) {
	files := dcRepo(`{"dependencies": {"left-pad": "^1.0.0"}}`, "", dcLeftPad, "1.0.0")
	files[".npmrc"] = dcUnreachable
	out, code := dependencyCurrency(t, files)
	sees(t, out, code, 0, "All 1 checks passed")

	home := t.TempDir()
	write(t, home, map[string]string{".npmrc": dcUnreachable})
	delete(files, ".npmrc")
	out, code = dependencyCurrency(t, files, "HOME="+home)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0022_8(t *testing.T) {
	out, code := dependencyCurrency(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "dependency-currency")
}
