package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/internal/repocheck"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/selection"
)

// localBins maps this run's local checks, from the repo's fitness/ folder
// and .fitnessrc.json's localChecks folder, by name. loadLocalChecks sets it
// once per run, before any check resolves.
var localBins map[string]repocheck.Check

// version is this fitness's release version, which the release build sets.
// A local check's minFitness is judged against it; a build without one,
// such as `go build`, meets every minFitness.
var version string

// built memoizes each Go local check's program for the run, so a check
// that resolves twice, as `fitness list` and a suite run do, builds once.
var built = struct {
	sync.Mutex
	paths map[string]string
}{paths: map[string]string{}}

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

// programFor finds the program of each check name resolves, failing on a
// missing one: a local check says why, and any other is not installed.
func programFor(name string) (checkkit.Program, error) {
	prog, err := findCheckBinary(name)
	if _, local := localBins[name]; err != nil && !local {
		err = fmt.Errorf("%w: fitness-check-%s", errNotInstalled, name)
	}
	return prog, err
}

// localProgram returns the program for a local check, building a Go one
// into the cache on first use.
func localProgram(c repocheck.Check) (checkkit.Program, error) {
	built.Lock()
	defer built.Unlock()
	if path, ok := built.paths[c.Name]; ok {
		return checkkit.Program{Path: path}, nil
	}
	cache, err := distribution.CacheDir()
	if err != nil {
		return checkkit.Program{}, err
	}
	path, err := repocheck.Build(c, cache, version, os.Stderr)
	if err != nil {
		return checkkit.Program{}, err
	}
	built.paths[c.Name] = path
	return checkkit.Program{Path: path}, nil
}

// asLocal applies the local check rules to a check's handshake. Every local
// check shows scope local, and an older fitness than its minFitness is
// noted. A folder check's handshake is optional: the folder names it, a
// handshake it does not answer is no info, and its README gives the summary
// the handshake leaves out.
func asLocal(r resolved) resolved {
	c, ok := localBins[r.name]
	if !ok {
		return r
	}
	r.needs = repocheck.NeedsNewer(r.name, r.desc.MinFitness, version)
	r.desc.Scope = "local"
	if c.Readme == "" {
		return r
	}
	r.desc.Name, r.describeErr = r.name, ""
	if r.desc.Summary == "" {
		r.desc.Summary = repocheck.Summary(c)
	}
	return r
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

// findCheckBinary returns the program for a check name: the repo's local
// check when one has that name, else a program of that name beside fitness,
// else the built-in check, run from this fitness executable. A program
// beside fitness that replaces a built-in check says so on stderr, so a
// stale one left from an older install does not run unnoticed.
func findCheckBinary(name string) (checkkit.Program, error) {
	if c, ok := localBins[name]; ok {
		return localProgram(c)
	}
	_, builtin := builtins[name]
	if bin, ok := besidePrograms[name]; ok {
		if builtin {
			fmt.Fprintf(os.Stderr, "fitness: %s replaces the built-in %s check; delete it to run the built-in one\n", bin, name)
		}
		return checkkit.Program{Path: bin}, nil
	}
	return builtinProgram(name)
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

// runGH runs a GitHub command with the repo's check settings, so an issue or
// PR body meets the same local and disabled checks as the repo's files.
func runGH(name string) int {
	if err := useRepoChecks(); err != nil {
		report.Error("fitness "+name, err)
		return 1
	}
	return metadata.RunGH(name)
}

// useRepoChecks checks the repo's settings under ADR 0009, as a suite run
// does, then points the GitHub commands at its local and disabled checks.
func useRepoChecks() error {
	_, cfg, _, err := blobConfig()
	if err != nil {
		return err
	}
	metadata.RepoChecks.Find = findCheckBinary
	metadata.RepoChecks.Off = func(name string) bool { return turnedOff(cfg, name) }
	return nil
}
