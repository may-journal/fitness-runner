package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

// pendingCheck is a pinned check not fetched yet. Its pin and index fields
// stand in for the handshake until the run's files show it applies.
type pendingCheck struct {
	pin  string
	desc checkkit.Describe
}

// pending holds, by name, the checks a suite run fetches only once its
// files reach them. fetchedPrograms are the programs the run holds.
var (
	pending         map[string]pendingCheck
	fetchedPrograms []string
)

// fetchFailures says, by name, why each reached check could not be fetched,
// so a later stage does not ask again.
var fetchFailures = map[string]string{}

// fetchedResolved holds each check fetched on demand, by name. A later stage
// uses it in place of its stand-in.
var fetchedResolved = map[string]resolved{}

// bodyCommands judge the Actions event's bodies with the checks they name,
// so each fetches a check only when it first runs it.
var bodyCommands = []string{"pr-check", "dependabot-pr", "issue-check", "plan-check", "close-check", "dupe-check"}

// defersFetch reports whether the command line fetches each check once it
// reaches it: a plain run, the pre-commit hook, or a GitHub command. Other
// commands fetch up front.
func defersFetch(args []string) bool {
	if len(args) == 0 {
		return true
	}
	if judgesEvent(args) {
		return true
	}
	if args[0] == "hook" {
		return slices.Equal(args, []string{"hook"}) || args[1] == "pre-commit"
	}
	return suiteRun(args)
}

// judgesEvent reports whether args judge the Actions event's bodies: one of
// bodyCommands, or comment-check with no draft inside GitHub Actions.
func judgesEvent(args []string) bool {
	if args[0] == "comment-check" {
		return len(args) == 1 && os.Getenv("GITHUB_EVENT_NAME") != ""
	}
	return slices.Contains(bodyCommands, args[0])
}

// findOrFetch finds the program of check name, fetching it first when
// pending, so a GitHub command downloads only the checks it runs.
func findOrFetch(name string) (checkkit.Program, error) {
	if _, ok := pending[name]; ok && fetchFailures[name] == "" {
		fetchPending(name)
	}
	if note := fetchFailures[name]; note != "" {
		return checkkit.Program{}, errors.New(note)
	}
	return findCheckBinary(name)
}

// fetchPending fetches and describes one pending check, as a suite run does,
// and hands it to the run. A failure stands for the rest of the run.
func fetchPending(name string) {
	fresh, err := fetchNamed([]string{name})
	if err != nil {
		// No installer means no pending check can be fetched.
		failPending(pendingNames(), err)
		return
	}
	if err := newEnough(fresh); err != nil {
		failPending([]string{name}, err)
		return
	}
	if note := fetchFailures[name]; note != "" {
		fmt.Fprintf(os.Stderr, "fitness: warning: %s\n", note)
	}
	keepProgram(name)
}

// keepProgram adds a fetched check's program to the installed ones, so it
// runs even when its check set could not be linked.
func keepProgram(name string) {
	r, ok := fetchedResolved[name]
	if !ok || installedPrograms[name] != "" {
		return
	}
	if installedPrograms == nil {
		installedPrograms = map[string]string{}
	}
	installedPrograms[name] = r.prog.Path
}

// failPending notes why each named check was not fetched, and warns once.
func failPending(names []string, why error) {
	for _, name := range names {
		fetchFailures[name] = fmt.Sprintf("check %s could not be fetched, so it judged nothing: %v", name, why)
	}
	fmt.Fprintf(os.Stderr, "fitness: warning: could not fetch %s: %v\n", strings.Join(names, ", "), why)
}

// suiteRun reports whether args run the whole suite on files: no command,
// no text and no check name.
func suiteRun(args []string) bool {
	_, subcommand := subcommands[args[0]]
	spec, _ := parseArgv(args)
	return !subcommand && !hasBlobFlag(args) && spec == ""
}

// hookPins narrows missing to the checks a commit-msg hook runs, which
// judge the message alone; any other command line keeps them all.
func hookPins(argv []string, missing map[string]string) map[string]string {
	if len(argv) < 2 || !slices.Equal(argv[:2], []string{"hook", "commit-msg"}) {
		return missing
	}
	kept := map[string]string{}
	for _, name := range commitMsgChecks() {
		if pin, ok := missing[name]; ok {
			kept[name] = pin
		}
	}
	return kept
}

// planFetches records each missing pin as pending, after linking the cached
// ones. False means the release index cannot be read, so every pin installs.
func planFetches(missing map[string]string) bool {
	installer, err := distribution.New("", "", os.Stderr)
	var planned []distribution.PlannedCheck
	if err == nil {
		planned, err = installer.PlanChecks(context.Background(), pinList(missing))
	}
	if err != nil {
		return false
	}
	recordPlanned(planned)
	if len(fetchedPrograms) > 0 {
		linkFetched(installer)
	}
	return true
}

// recordPlanned keeps each cached program and makes the rest pending.
func recordPlanned(planned []distribution.PlannedCheck) {
	pending = map[string]pendingCheck{}
	for _, p := range planned {
		if p.Program != "" {
			fetchedPrograms = append(fetchedPrograms, p.Program)
			continue
		}
		pending[p.Pin.Name] = pendingCheck{pin: p.Pin.Pin, desc: p.Describe}
	}
}

// pendingNames lists the pending checks. None is installed, since only
// missing pins become pending.
func pendingNames() []string {
	return slices.Collect(maps.Keys(pending))
}

// pinList lists pins in name order.
func pinList(pins map[string]string) []distribution.CheckPin {
	list := make([]distribution.CheckPin, 0, len(pins))
	for _, name := range slices.Sorted(maps.Keys(pins)) {
		list = append(list, distribution.CheckPin{Name: name, Pin: pins[name]})
	}
	return list
}

// linkFetched links every program the run holds into one check set and
// hands it to the run. A set that cannot be made leaves the run as it was.
func linkFetched(installer *distribution.Installer) {
	dir, err := installer.CheckSet(fetchedPrograms)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fitness: warning: could not link the fetched checks: %v\n", err)
		return
	}
	addFetched(dir)
}

// standIn is the resolved entry of a pending check, with no program. Its
// index fields take the place of a handshake until the run fetches it.
func standIn(name string) (resolved, bool) {
	p, ok := pending[name]
	if _, local := localBins[name]; !ok || local {
		return resolved{}, false
	}
	return resolved{name: name, desc: p.desc, pending: true}, true
}

// fetchReached fetches, side by side, the pending checks the run reaches,
// putting each in place of its stand-in. env comes back naming the check
// set that holds them, and a fetched check too new for this fitness fails.
func fetchReached(root string, checks []resolved, env, shared []string) ([]resolved, []string, error) {
	checks = withFetched(checks)
	names := toFetch(root, checks, env, shared)
	var fresh []resolved
	var err error
	if len(names) > 0 {
		fresh, err = fetchNamed(names)
	}
	if err != nil {
		return nil, nil, err
	}
	return withFetched(checks), withChecksDir(env), newEnough(fresh)
}

// withChecksDir names in env the check set the run now holds, which an
// earlier stage may have changed. A run holding none keeps env as it is.
func withChecksDir(env []string) []string {
	dir := os.Getenv(checkkit.ChecksDirEnv)
	if dir == "" {
		return env
	}
	return append(withoutVar(env, checkkit.ChecksDirEnv), checkkit.ChecksDirEnv+"="+dir)
}

// toFetch names the pending checks the run reaches, lacks nothing for, and
// has not failed to fetch. fetchesEvery runs fetch all the others.
func toFetch(root string, checks []resolved, env, shared []string) []string {
	want := checks
	if !fetchesEvery(checks, env) {
		want, _ = applicable(root, checks, env, shared)
	}
	run := runOf(env)
	var names []string
	for _, c := range want {
		if fetchable(c, run) {
			names = append(names, c.name)
		}
	}
	return names
}

// fetchable reports whether c is pending, not failed yet, and lacks nothing
// in run.
func fetchable(c resolved, run runContext) bool {
	return c.pending && fetchFailures[c.name] == "" && run.lackReason(c) == ""
}

// fetchesEvery reports whether the run fetches every pending check: a full
// run, or one local-branch judges, since it runs other checks on bodies.
func fetchesEvery(checks []resolved, env []string) bool {
	return scopedFiles(env) == nil || slices.ContainsFunc(checks, func(c resolved) bool { return c.name == localBranch })
}

// withFetched puts each check fetched so far in place of its stand-in.
func withFetched(checks []resolved) []resolved {
	out := make([]resolved, len(checks))
	for i, c := range checks {
		if r, ok := fetchedResolved[c.name]; ok && c.pending {
			c = r
		}
		out[i] = c
	}
	return out
}

// fetchNamed fetches and describes the named pending checks, noting why
// each that could not be fetched did not, and returns the fetched ones.
func fetchNamed(names []string) ([]resolved, error) {
	installer, err := distribution.New("", "", os.Stderr)
	if err != nil {
		return nil, err
	}
	programs, failed := installer.FetchChecks(context.Background(), pendingPins(names))
	noteFailures(failed)
	got := slices.Sorted(maps.Keys(programs))
	if len(got) > 0 {
		takeFetched(got, programs)
		linkFetched(installer)
		fmt.Fprintf(os.Stderr, "fitness: fetched %d checks this run reaches\n", len(got))
	}
	return describeFetched(got, programs), nil
}

// pendingPins lists the pins of the named pending checks.
func pendingPins(names []string) []distribution.CheckPin {
	pins := map[string]string{}
	for _, name := range names {
		pins[name] = pending[name].pin
	}
	return pinList(pins)
}

// noteFailures records the info note of each check that could not be fetched.
func noteFailures(failed map[string]error) {
	for name, why := range failed {
		fetchFailures[name] = fmt.Sprintf("check %s applies to this run's files but could not be fetched, so it judged nothing: %v", name, why)
	}
}

// takeFetched moves the named checks from pending to the run's programs.
func takeFetched(names []string, programs map[string]string) {
	for _, name := range names {
		fetchedPrograms = append(fetchedPrograms, programs[name])
		delete(pending, name)
	}
}

// describeFetched runs the handshake of each fetched check, keeping each
// for the run's later stages.
func describeFetched(names []string, programs map[string]string) []resolved {
	progs := make([]checkkit.Program, len(names))
	for i, name := range names {
		progs[i] = checkkit.Program{Path: programs[name]}
	}
	fresh := describeAll(names, progs)
	retryOverran(fresh)
	for _, r := range fresh {
		fetchedResolved[r.name] = r
	}
	return fresh
}

// withoutFailedFetches keeps out of the run each check whose fetch failed,
// returning the rest and the info note of each one kept out.
func withoutFailedFetches(checks []resolved) ([]resolved, map[string]string) {
	notes := map[string]string{}
	var kept []resolved
	for _, c := range checks {
		if note := fetchFailures[c.name]; c.pending && note != "" {
			notes[c.name] = note
			continue
		}
		kept = append(kept, c)
	}
	return kept, notes
}
