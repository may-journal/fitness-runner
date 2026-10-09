package requirements

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// The packages these tests declare, each with its saved latest version.
const (
	dcLeftPad  = "left-pad"        // latest 1.3.0
	dcIsNumber = "is-number"       // latest 7.0.0
	dcTypes    = "@types/left-pad" // latest 1.2.0
	dcInternal = "@mayjournal/fitness-shared"
)

// dcLead is the line that opens every failure.
const dcLead = "Dependencies behind their latest published version — update them (npm install <pkg>@latest) or pin intentionally:"

// dcUnreachable is a registry address nothing listens on.
const dcUnreachable = "http://127.0.0.1:1"

// dcDoc is a package's abbreviated registry document, as npm serves it for
// Accept: application/vnd.npm.install-v1+json. The last version is latest.
func dcDoc(name string, versions ...string) string {
	all := map[string]any{}
	for _, v := range versions {
		all[v] = map[string]any{"name": name, "version": v, "dist": map[string]string{"shasum": strings.Repeat("0", 40)}}
	}
	doc, _ := json.Marshal(map[string]any{
		"name":      name,
		"modified":  "2024-01-01T00:00:00.000Z",
		"dist-tags": map[string]string{"latest": versions[len(versions)-1]},
		"versions":  all,
	})
	return string(doc)
}

// dcRegistry serves the saved documents for every package these tests
// declare. The internal package has a newer published version, so it would
// fail if the check did not skip it.
func dcRegistry(t *testing.T) string {
	t.Helper()
	return registry(t, map[string]string{
		dcLeftPad:  dcDoc(dcLeftPad, "1.0.0", "1.1.0", "1.3.0"),
		dcIsNumber: dcDoc(dcIsNumber, "6.0.0", "7.0.0"),
		dcTypes:    dcDoc(dcTypes, "1.0.0", "1.2.0"),
		dcInternal: dcDoc(dcInternal, "9.9.9"),
	})
}

// dcAnswer is a registry that answers every request with status and body,
// and counts the requests it received.
func dcAnswer(t *testing.T, status int, body string) (string, *atomic.Int32) {
	t.Helper()
	hits := new(atomic.Int32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL, hits
}

// dcNpmrc is an .npmrc pointing npm at registry.
func dcNpmrc(registry string) string {
	return "registry=" + registry + "\n"
}

// dependencyCurrency runs `fitness-install -- dependency-currency` on
// happyRepo with files written over it. Like a real repo, it tracks its
// manifests and ignores the installed node_modules.
func dependencyCurrency(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", with(map[string]string{".gitignore": "node_modules\n"}, files))
	return fitness(t, repo, append([]string{"HOME=" + t.TempDir()}, env...), "dependency-currency")
}

// dcInstalled is the node_modules manifest of name installed at version,
// under dir ("" for the repo root).
func dcInstalled(dir, name, version string) (string, string) {
	return dir + "node_modules/" + name + "/package.json", `{"name": "` + name + `", "version": "` + version + `"}` + "\n"
}

// dcRepo is a package.json declaring manifest, an .npmrc pointing at
// registry, plus each install given as dir, name, version triples.
func dcRepo(registry, manifest string, installs ...string) map[string]string {
	files := map[string]string{"package.json": manifest + "\n", ".npmrc": dcNpmrc(registry)}
	for i := 0; i+2 < len(installs); i += 3 {
		path, body := dcInstalled(installs[i], installs[i+1], installs[i+2])
		files[path] = body
	}
	return files
}

func Test0022_1(t *testing.T) {
	t.Parallel()
	out, code := dependencyCurrency(t, dcRepo(dcRegistry(t), `{"dependencies": {"left-pad": "^1.3.0"}}`, "", dcLeftPad, "1.3.0"))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0022_2(t *testing.T) {
	t.Parallel()
	files := dcRepo(dcRegistry(t), `{"dependencies": {"left-pad": "^1.0.0"}, "devDependencies": {"is-number": "^6.0.0"}, "peerDependencies": {"@types/left-pad": "^1.0.0"}}`,
		"", dcLeftPad, "1.0.0", "", dcIsNumber, "6.0.0", "", dcTypes, "1.0.0")
	out, code := dependencyCurrency(t, files)
	sees(t, out, code, 1, dcLead, "left-pad: 1.0.0 → 1.3.0", "is-number: 6.0.0 → 7.0.0", "@types/left-pad: 1.0.0 → 1.2.0")
}

func Test0022_3(t *testing.T) {
	t.Parallel()
	out, code := dependencyCurrency(t, dcRepo(dcRegistry(t), `{"dependencies": {"left-pad": "^1.3.0"}}`))
	sees(t, out, code, 1, dcLead, "left-pad: missing → 1.3.0")
}

func Test0022_4(t *testing.T) {
	t.Parallel()
	out, code := dependencyCurrency(t, dcRepo(dcRegistry(t), `{"dependencies": {"`+dcInternal+`": "*"}}`))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0022_5(t *testing.T) {
	t.Parallel()
	for _, workspaces := range []string{`["packages/*"]`, `{"packages": ["packages/a"]}`} {
		files := dcRepo(dcRegistry(t), `{"workspaces": `+workspaces+`}`, "", dcLeftPad, "1.0.0")
		files["packages/a/package.json"] = `{"dependencies": {"left-pad": "^1.0.0"}}` + "\n"
		out, code := dependencyCurrency(t, files)
		sees(t, out, code, 1, dcLead, "left-pad: 1.0.0 → 1.3.0")
	}
}

func Test0022_6(t *testing.T) {
	t.Parallel()
	files := dcRepo(dcRegistry(t), `{"workspaces": ["packages/*"]}`, "", dcLeftPad, "1.0.0", "packages/a/", dcLeftPad, "1.1.0")
	for _, ws := range []string{"a", "b", "c"} {
		files["packages/"+ws+"/package.json"] = `{"devDependencies": {"left-pad": "^1.0.0"}}` + "\n"
	}
	out, code := dependencyCurrency(t, files)
	sees(t, out, code, 1, "left-pad: 1.1.0 → 1.3.0", "left-pad: 1.0.0 → 1.3.0")
	if n := strings.Count(read(out), "left-pad: 1.0.0 → 1.3.0"); n != 1 {
		t.Errorf("the shared install is listed %d times, want once:\n%s", n, out)
	}
}

// dcBroken is one registry that cannot tell the latest version, named in
// the repo or the home .npmrc.
type dcBroken struct {
	name, url string
	hits      *atomic.Int32
	home      bool
}

// dcBrokenRegistries are an unreachable registry, one answering an error,
// and one answering garbage, in the repo .npmrc, then garbage in the home one.
func dcBrokenRegistries(t *testing.T) []dcBroken {
	failing, failHits := dcAnswer(t, http.StatusInternalServerError, `{"error":"Internal Server Error"}`)
	garbage, garbageHits := dcAnswer(t, http.StatusOK, "<html>proxy login</html>")
	homeGarbage, homeHits := dcAnswer(t, http.StatusOK, `{"dist-tags": `)
	return []dcBroken{
		{name: "unreachable", url: dcUnreachable},
		{name: "server error", url: failing, hits: failHits},
		{name: "garbage", url: garbage, hits: garbageHits},
		{name: "home garbage", url: homeGarbage, hits: homeHits, home: true},
	}
}

func Test0022_7(t *testing.T) {
	t.Parallel()
	for _, broken := range dcBrokenRegistries(t) {
		files := dcRepo(broken.url, `{"dependencies": {"left-pad": "^1.0.0"}}`, "", dcLeftPad, "1.0.0")
		home := t.TempDir()
		if broken.home {
			delete(files, ".npmrc")
			write(t, home, map[string]string{".npmrc": dcNpmrc(broken.url)})
		}
		out, code := dependencyCurrency(t, files, "HOME="+home)
		sees(t, out, code, 0, "All 1 checks passed")
		if broken.hits != nil && broken.hits.Load() == 0 {
			t.Errorf("%s: the check never asked the registry", broken.name)
		}
	}
}

func Test0022_8(t *testing.T) {
	t.Parallel()
	out, code := dependencyCurrency(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "dependency-currency")
}
