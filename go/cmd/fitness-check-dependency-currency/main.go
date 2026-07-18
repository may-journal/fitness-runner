// Command fitness-check-dependency-currency fails when any declared
// dependency is behind its latest published version — the Go port of the
// dependency-currency check. One deviation from the TypeScript original: it
// queries the npm registry directly over HTTP (abbreviated metadata,
// dist-tags.latest) instead of shelling out to `npm outdated --json`,
// enumerating the root manifest plus its workspaces itself. The soul is
// unchanged: internal @mayjournal/* packages are skipped, identical lines
// collapse to one, and an unreachable registry (or any garbage response)
// contributes nothing — fully offline degrades to a pass so commits are
// never blocked.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

const packageJSON = "package.json"

// defaultRegistry is the npm registry queried when no .npmrc overrides it.
const defaultRegistry = "https://registry.npmjs.org"

// internalScope marks workspace packages versioned in-repo, never
// "outdated" against the registry — the TS INTERNAL_SCOPE regex.
const internalScope = "@mayjournal/"

// lead is the first error line, verbatim from the TS enUS.Lead.
const lead = "Dependencies behind their latest published version — update them (npm install <pkg>@latest) or pin intentionally:"

// fetchWorkers bounds the concurrent registry requests.
const fetchWorkers = 8

// fetchBudget caps the whole fetch phase inside the check's 30s timeout so
// a slow registry degrades to a pass instead of the runner killing us.
const fetchBudget = 25 * time.Second

// requestTimeout caps one registry request within the overall budget.
const requestTimeout = 10 * time.Second

func main() {
	checkkit.Main(checkkit.Check{
		// The registry answers one request per dependency; on a cold cache
		// that can exceed the default 5s budget, so keep the TS check's 30s.
		Describe: checkkit.Describe{Name: "dependency-currency", TimeoutMs: 30000},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	filesChecked := len(walkfs.FilesByExt(root, packageJSON))
	manifests := collectManifests(root)
	latest := fetchLatest(registryFrom(root), uniqueDeps(manifests))
	errs := outdatedErrors(manifests, latest)
	if len(errs) > 0 {
		return checkkit.Fail(filesChecked, append([]string{lead}, errs...)...), nil
	}
	return checkkit.Pass(filesChecked), nil
}

// manifest is one package.json's declared external dependency names plus
// the node_modules directories consulted, nearest first, to resolve each
// installed version.
type manifest struct {
	deps        []string
	nodeModules []string
}

// pkgJSON is the slice of a package.json this check reads.
type pkgJSON struct {
	Version          string            `json:"version"`
	Workspaces       json.RawMessage   `json:"workspaces"`
	Dependencies     map[string]string `json:"dependencies"`
	DevDependencies  map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

// readPkg parses one package.json; ok is false when it is absent or invalid.
func readPkg(path string) (pkgJSON, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return pkgJSON{}, false
	}
	var pkg pkgJSON
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return pkgJSON{}, false
	}
	return pkg, true
}

// collectManifests returns the root manifest plus one per workspace dir
// resolved from the root "workspaces" globs. Workspace deps resolve against
// the workspace's own node_modules first, then the hoisted root one.
func collectManifests(root string) []manifest {
	rootPkg, ok := readPkg(filepath.Join(root, packageJSON))
	if !ok {
		return nil
	}
	rootModules := filepath.Join(root, "node_modules")
	out := []manifest{{deps: depNames(rootPkg), nodeModules: []string{rootModules}}}
	for _, dir := range workspaceDirs(root, rootPkg.Workspaces) {
		pkg, ok := readPkg(filepath.Join(dir, packageJSON))
		if !ok {
			continue
		}
		out = append(out, manifest{
			deps:        depNames(pkg),
			nodeModules: []string{filepath.Join(dir, "node_modules"), rootModules},
		})
	}
	return out
}

// workspaceDirs expands the "workspaces" field (either a glob array or the
// object form with a "packages" array) into existing directories under root.
func workspaceDirs(root string, raw json.RawMessage) []string {
	patterns := workspacePatterns(raw)
	seen := make(map[string]bool)
	var out []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
		if err != nil {
			continue
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || !info.IsDir() || seen[match] {
				continue
			}
			seen[match] = true
			out = append(out, match)
		}
	}
	sort.Strings(out)
	return out
}

// workspacePatterns decodes the raw "workspaces" value into its glob list.
func workspacePatterns(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}
	var obj struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Packages
	}
	return nil
}

// depNames merges a manifest's dependencies, devDependencies, and
// peerDependencies into a sorted unique name list, skipping the internal
// scope.
func depNames(pkg pkgJSON) []string {
	set := make(map[string]bool)
	for _, deps := range []map[string]string{pkg.Dependencies, pkg.DevDependencies, pkg.PeerDependencies} {
		for name := range deps {
			if strings.HasPrefix(name, internalScope) {
				continue
			}
			set[name] = true
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// uniqueDeps returns the sorted unique dependency names across manifests —
// each is one registry query.
func uniqueDeps(manifests []manifest) []string {
	set := make(map[string]bool)
	for _, m := range manifests {
		for _, name := range m.deps {
			set[name] = true
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// installedVersion resolves a dependency's installed version from the
// manifest's node_modules directories, nearest first; empty means missing.
func installedVersion(m manifest, name string) string {
	for _, dir := range m.nodeModules {
		if pkg, ok := readPkg(filepath.Join(dir, filepath.FromSlash(name), packageJSON)); ok {
			return pkg.Version
		}
	}
	return ""
}

// entryLine is "name: current → latest" when the dependency is behind its
// latest published version, else empty — the TS entryError. A dependency
// whose latest is unknown (fetch failed) contributes nothing; a declared
// but uninstalled one reports "missing".
func entryLine(name, installed, latest string) string {
	if latest == "" || installed == latest {
		return ""
	}
	current := installed
	if current == "" {
		current = "missing"
	}
	return name + ": " + current + " → " + latest
}

// outdatedErrors returns the sorted, de-duplicated behind-latest lines
// across every manifest. Identical lines (the same dep+version across
// several workspaces) collapse to one — bumping it is a single action.
func outdatedErrors(manifests []manifest, latest map[string]string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, m := range manifests {
		for _, name := range m.deps {
			line := entryLine(name, installedVersion(m, name), latest[name])
			if line == "" || seen[line] {
				continue
			}
			seen[line] = true
			out = append(out, line)
		}
	}
	sort.Strings(out)
	return out
}

// registryFrom resolves the registry URL from the nearest .npmrc "registry="
// entry — repo root first, then the home directory — defaulting to the
// public npm registry.
func registryFrom(root string) string {
	paths := []string{filepath.Join(root, ".npmrc")}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".npmrc"))
	}
	for _, path := range paths {
		if registry := npmrcRegistry(path); registry != "" {
			return registry
		}
	}
	return defaultRegistry
}

// npmrcRegistry returns the "registry" value from one .npmrc, trailing
// slash trimmed; empty when the file or the key is absent.
func npmrcRegistry(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(key) != "registry" {
			continue
		}
		if v := strings.TrimRight(strings.TrimSpace(value), "/"); v != "" {
			return v
		}
	}
	return ""
}

// fetchLatest resolves each name's dist-tags.latest from the registry with
// bounded concurrency. Any failure — network error, non-200, garbage body —
// leaves that name out of the map, so a fully unreachable registry yields
// an empty map and the check degrades to a pass.
func fetchLatest(registry string, names []string) map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), fetchBudget)
	defer cancel()
	client := &http.Client{Timeout: requestTimeout}
	jobs := make(chan string)
	out := make(map[string]string, len(names))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < fetchWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range jobs {
				if version := latestVersion(ctx, client, registry, name); version != "" {
					mu.Lock()
					out[name] = version
					mu.Unlock()
				}
			}
		}()
	}
	for _, name := range names {
		jobs <- name
	}
	close(jobs)
	wg.Wait()
	return out
}

// latestVersion asks the registry for one package's abbreviated metadata
// and returns dist-tags.latest; empty on any failure.
func latestVersion(ctx context.Context, client *http.Client, registry, name string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registry+"/"+url.PathEscape(name), nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.npm.install-v1+json")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var meta struct {
		DistTags map[string]string `json:"dist-tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return ""
	}
	return meta.DistTags["latest"]
}
