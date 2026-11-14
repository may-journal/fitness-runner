package main

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/selection"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
	"github.com/may-journal/fitness-runner/go/pkg/conf"
	"github.com/may-journal/fitness-runner/go/pkg/lang"
	"github.com/may-journal/fitness-runner/go/pkg/par"
	"github.com/may-journal/fitness-runner/go/pkg/repocheck"
)

// localBins maps this run's local checks, from the repo's fitness/ folder
// and .fitnessrc.json's localChecks folder, by name. loadLocalChecks sets it
// once per run, before any check resolves.
var localBins map[string]repocheck.Check

// version is this fitness's release version, which the release build sets.
// A local check's minFitness is judged against it; a build without one,
// such as `go build`, meets every minFitness.
var version string

// built memoizes each Go local check's build for the run, so a check that
// resolves twice builds once.
var built = struct {
	sync.Mutex
	builds map[string]*build
}{builds: map[string]*build{}}

// build is one check's build, done once.
type build struct {
	once sync.Once
	path string
	err  error
}

// turnedOff reports whether an external policy's disabledChecks names name,
// so a hook skips it rather than fail on an empty selection.
func turnedOff(cfg *conf.Config, name string) bool {
	return cfg != nil && cfg.Policy == selection.External && slices.Contains(cfg.DisabledChecks, name)
}

// loadLocalChecks finds the repo's local checks, so every later lookup
// searches them before the bundled programs.
func loadLocalChecks(cfg *conf.Config, root string) error {
	var configured string
	if cfg != nil {
		configured = cfg.LocalChecks
	}
	checks, err := repocheck.Find(root, configured)
	if err == nil {
		err = noSubcommand(checks)
	}
	if err == nil {
		localBins = checks
	}
	return err
}

// noSubcommand fails a local check named like a fitness subcommand, which
// `fitness <name>` could not run.
func noSubcommand(checks map[string]repocheck.Check) error {
	for name := range checks {
		if _, taken := subcommands[name]; taken || name == "help" {
			return fmt.Errorf("local check %s shares a fitness subcommand's name, so `fitness %s` could not run it; rename it", name, name)
		}
	}
	return nil
}

// programFor finds the program of each check name resolves. A missing one
// fails as "not installed", unless it names a built-in that is off.
func programFor(name string) (checkkit.Program, error) {
	prog, err := findCheckBinary(name)
	if _, local := localBins[name]; err != nil && !local && (!builtinsOff() || builtins[name] == nil) {
		err = fmt.Errorf("%w: fitness-check-%s", errNotInstalled, name)
	}
	return prog, err
}

// localProgram returns the program for a local check, building a Go one
// into the cache on first use.
func localProgram(c repocheck.Check) (checkkit.Program, error) {
	built.Lock()
	b := built.builds[c.Name]
	if b == nil {
		b = &build{}
		built.builds[c.Name] = b
	}
	built.Unlock()
	b.once.Do(func() { b.path, b.err = buildCheck(c) })
	return checkkit.Program{Path: b.path}, b.err
}

// buildCheck builds a Go local check into fitness's cache, or returns the
// program of any other.
func buildCheck(c repocheck.Check) (string, error) {
	cache, err := distribution.CacheDir()
	if err != nil {
		return "", err
	}
	return repocheck.Build(c, cache, version, os.Stderr)
}

// prebuild builds the Go local checks among names side by side, so a run waits
// only on the slowest build.
func prebuild(names []string) {
	par.Map(len(names), par.Processes(), func(i int) error {
		if c, ok := localBins[names[i]]; ok {
			_, err := localProgram(c)
			return err
		}
		return nil
	})
}

// asLocal applies the local check rules to a handshake. Scope defaults to local,
// and a folder check's handshake is optional.
func asLocal(r resolved) resolved {
	c, ok := localBins[r.name]
	if !ok {
		return r
	}
	r.needs = repocheck.NeedsNewer(r.name, r.desc.MinFitness, version)
	r.desc.Scope = cmp.Or(r.desc.Scope, "local")
	if c.Readme == "" {
		return r
	}
	r.desc.Name, r.describeErr = r.name, ""
	if r.desc.Summary == "" {
		r.desc.Summary = repocheck.Summary(c)
	}
	return r
}

// checkDirEnv hands a folder check its own folder, where its config.json
// lives, as the go/check SDK reads it.
func checkDirEnv(name string) []string {
	if c, ok := localBins[name]; ok && c.Folder != "" {
		return []string{"FITNESS_CHECK_DIR=" + c.Folder}
	}
	return nil
}

// brokenScan fails a paired local check, adr-NNNN or req-NNNN, that passed
// having judged no files. Its decision or requirement holds only while the
// check reads the files it governs, so an empty scan is a broken one.
func brokenScan(c resolved, o outcome) outcome {
	if _, local := localBins[c.name]; !local || !repocheck.Paired(c.name) || !emptyPass(o) {
		return o
	}
	o.res.Ok = false
	o.res.Errors = append(o.res.Errors, c.name+" judged no files, so its scan is broken; fix it to read the files its doc governs")
	return o
}

// emptyPass reports whether a check passed, by its own verdict, having
// judged no files.
func emptyPass(o outcome) bool {
	return o.exit == 0 && o.res.Ok && o.res.FilesChecked <= 0
}

// newEnough fails the first check whose minFitness this fitness does not
// meet, so a run stops before any check starts, naming the upgrade.
func newEnough(checks []resolved) error {
	for _, c := range checks {
		if c.needs != "" {
			return errors.New(c.needs)
		}
	}
	return nil
}

// findCheckBinary returns a local check, else the engine's judgment, else an
// installed program, else a built-in.
func findCheckBinary(name string) (checkkit.Program, error) {
	if c, ok := localBins[name]; ok {
		return localProgram(c)
	}
	if prog, ok := engineProgram(name); ok {
		return prog, nil
	}
	return installedOrBuiltin(name)
}

// engineProgram returns the engine's own program for a check it judges, when
// an installed program declares it or the built-ins are on. An installed
// program cannot judge alone, so it only declares the check; this says so.
func engineProgram(name string) (checkkit.Program, bool) {
	_, judged := engineJudges[name]
	bin, installed := installedPrograms[name]
	if !judged || !installed && builtinsOff() {
		return checkkit.Program{}, false
	}
	if installed {
		fmt.Fprintf(os.Stderr, "fitness: %s declares the %s check, which fitness judges itself\n", bin, name)
	}
	prog, err := builtinProgram(name)
	return prog, err == nil
}

// installedOrBuiltin returns an installed program, else the built-in of a
// name. An installed program that replaces a built-in says so on stderr, and
// with the built-ins off a missing one names its install.
func installedOrBuiltin(name string) (checkkit.Program, error) {
	if bin, ok := installedPrograms[name]; ok {
		noteReplaced(bin, name)
		return checkkit.Program{Path: bin}, nil
	}
	if builtinsOff() {
		return checkkit.Program{}, fmt.Errorf("check %s is not installed, and %s turns the built-in checks off; install it: %s", name, noBuiltinsFlag, installCommand(name))
	}
	return builtinProgram(name)
}

// noteReplaced says on stderr that bin replaces the built-in check name,
// unless no such built-in runs.
func noteReplaced(bin, name string) {
	if builtins[name] != nil && !builtinsOff() && !pinnedProgram(bin) {
		fmt.Fprintf(os.Stderr, "fitness: %s replaces the built-in %s check; delete it to run the built-in one\n", bin, name)
	}
}

// pinnedProgram reports whether bin lies in the installer's folder of pinned
// checks, which replace their built-ins on purpose.
func pinnedProgram(bin string) bool {
	dir := os.Getenv(checkkit.ChecksDirEnv)
	return dir != "" && filepath.Dir(bin) == filepath.Clean(dir)
}

// builtinProgram returns the program for built-in check name: this fitness
// executable, started as fitness-check-<name>.
func builtinProgram(name string) (checkkit.Program, error) {
	if _, ok := builtins[name]; !ok {
		return checkkit.Program{}, fmt.Errorf("no built-in check %s", name)
	}
	return checkkit.Builtin(name)
}

// checkCatalog is the built-in check list followed by the local checks that add
// a new name, in name order. A local check with a built-in name keeps that
// check's place.
func checkCatalog() []string {
	names := append([]string{}, allChecks...)
	for _, name := range localNames() {
		if !builtIn(name) {
			names = append(names, name)
		}
	}
	return names
}

// replacedChecks lists the local checks that share a built-in check's name.
func replacedChecks() []string {
	var replaced []string
	for _, name := range localNames() {
		if builtIn(name) {
			replaced = append(replaced, name)
		}
	}
	return replaced
}

// allowLocalChecks enforces ADR 0009 for the resolved policy and names each
// replaced built-in, so a run never swaps one in silently.
func allowLocalChecks(policy string) error {
	replaced := replacedChecks()
	if err := selection.AllowReplacements(policy, replaced); err != nil {
		return err
	}
	if len(replaced) > 0 {
		fmt.Fprintln(os.Stderr, "fitness: local checks replace built-in "+strings.Join(replaced, ","))
	}
	return nil
}

// repoChecks applies the repo's own check settings to the resolved names,
// under ADR 0009: local checks that replace a bundled one, then
// disabledChecks.
func repoChecks(cfg *conf.Config, policy string, names []string) ([]string, error) {
	if err := allowLocalChecks(policy); err != nil {
		return nil, err
	}
	var disabled []string
	if cfg != nil {
		disabled = cfg.DisabledChecks
	}
	return selection.Disable(policy, names, disabled, append(checkCatalog(), lang.Ignored()...))
}

// configPolicy is the repo's configured policy, org unless it says external.
func configPolicy(cfg *conf.Config) string {
	if cfg != nil && cfg.Policy == selection.External {
		return selection.External
	}
	return selection.Org
}

// localNames is the local check names in order.
func localNames() []string {
	names := make([]string, 0, len(localBins))
	for name := range localBins {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// builtIn reports whether a bundled check has the name.
func builtIn(name string) bool {
	return slices.Contains(allChecks, name) || slices.Contains(lang.Ignored(), name)
}

// runGH runs a GitHub command with the repo's check settings, so a body meets
// the same checks as the repo's files.
func runGH(name string) int {
	if err := useRepoChecks(); err != nil {
		report.Error("fitness "+name, err)
		return 1
	}
	return metadata.RunGH(name)
}

// runDependabotPR completes a Dependabot pull request with the repo's check settings.
func runDependabotPR() int {
	if err := useRepoChecks(); err != nil {
		report.Error("fitness dependabot-pr", err)
		return 1
	}
	return metadata.RunDependabotPR()
}

// useRepoChecks checks the repo's settings under ADR 0009 and points the GitHub
// commands at its local and disabled checks.
func useRepoChecks() error {
	_, cfg, _, err := blobConfig()
	if err != nil {
		return err
	}
	metadata.RepoChecks.Find = findCheckBinary
	metadata.RepoChecks.Off = func(name string) bool { return turnedOff(cfg, name) }
	return nil
}
