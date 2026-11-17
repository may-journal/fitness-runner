package distribution

import (
	"context"
	"fmt"

	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
	"github.com/may-journal/fitness-runner/go/pkg/par"
)

// PlannedCheck is one pin the release index resolves without a download.
// Program is the cached program, if any, and Describe holds the fields the
// index copies, which decide whether a run fetches it.
type PlannedCheck struct {
	Pin      CheckPin
	Program  string
	Describe checkkit.Describe
}

// PlanChecks resolves each pin through the release index, downloading no
// check, and fails when the index cannot be read so every pin installs.
func (i *Installer) PlanChecks(ctx context.Context, pins []CheckPin) ([]PlannedCheck, error) {
	ctx, cancel := context.WithTimeout(ctx, checkLookupTimeout)
	defer cancel()
	index, err := i.readIndex(ctx)
	if err != nil {
		return nil, err
	}
	names, kept := i.released(pins, index.releases(pins, i.Platform))
	planned := make([]PlannedCheck, len(kept))
	for k, release := range kept {
		pin := pinNamed(pins, names[k])
		planned[k] = PlannedCheck{Pin: pin, Describe: index.describe(pin)}
		if program := i.checkPath(pin.Name, release); cachedProgram(program) {
			planned[k].Program = program
		}
	}
	return planned, nil
}

// pinNamed is the pin of check name among pins.
func pinNamed(pins []CheckPin, name string) CheckPin {
	for _, pin := range pins {
		if pin.Name == name {
			return pin
		}
	}
	return CheckPin{Name: name}
}

// describe is the part of pin's --describe the index copies: which files
// reach the check and what else it requires. A check requiring GitHub gets
// its scope, which the engine's GitHub probe reads.
func (index checkIndex) describe(pin CheckPin) checkkit.Describe {
	version := index.version(pin)
	d := checkkit.Describe{Name: pin.Name, Files: version.Files, Exclude: version.Exclude, Needs: version.Needs, Requires: version.Requires}
	if checkkit.Requiring(d, checkkit.RequiresGitHub) {
		d.Scope = checkkit.ScopeGitHub
	}
	return d
}

// FetchChecks installs each pinned check side by side, verified by SHA-256,
// returning each program by name and why each other one failed.
func (i *Installer) FetchChecks(ctx context.Context, pins []CheckPin) (programs map[string]string, failed map[string]error) {
	ctx, cancel := context.WithTimeout(ctx, installTotal)
	defer cancel()
	releases, err := i.resolvePins(ctx, pins)
	if err != nil {
		return map[string]string{}, failAll(pins, err)
	}
	failed = unreleased(pins, releases)
	names, kept := i.released(pins, releases)
	return i.installEach(ctx, names, kept, failed), failed
}

// failAll gives every pin the same failure.
func failAll(pins []CheckPin, err error) map[string]error {
	failed := map[string]error{}
	for _, pin := range pins {
		failed[pin.Name] = err
	}
	return failed
}

// unreleased fails each pin no release matches.
func unreleased(pins []CheckPin, releases []checkRelease) map[string]error {
	failed := map[string]error{}
	for k, release := range releases {
		if release.Tag == "" {
			failed[pins[k].Name] = fmt.Errorf("no release of check %s matches %s", pins[k].Name, pins[k].Pin)
		}
	}
	return failed
}

// installEach installs each release side by side. A failure stops no other
// install; it is recorded in failed.
func (i *Installer) installEach(ctx context.Context, names []string, releases []checkRelease, failed map[string]error) map[string]string {
	errs := make([]error, len(releases))
	paths := par.Map(len(releases), checkWorkers, func(k int) string {
		path, err := i.installCheck(ctx, names[k], releases[k])
		errs[k] = err
		return path
	})
	programs := map[string]string{}
	for k, err := range errs {
		if err != nil {
			failed[names[k]] = err
			continue
		}
		programs[names[k]] = paths[k]
	}
	return programs
}

// CheckSet returns a cache folder linking exactly programs, for
// FITNESS_CHECKS_DIR.
func (i *Installer) CheckSet(programs []string) (string, error) {
	return checkSet(i.Cache, programs, i.pinned)
}
