package main

import (
	"fmt"
	"os"

	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
	"github.com/may-journal/fitness-runner/go/pkg/gitx"
	"github.com/may-journal/fitness-runner/go/pkg/render"
)

// skipGroup is one reason a run leaves checks out, with their names in order.
type skipGroup struct {
	why   string
	names []string
}

// skipGroups collects skipped checks under their reasons, in first-seen order.
type skipGroups []skipGroup

// add records name under why.
func (g *skipGroups) add(why, name string) {
	for i := range *g {
		if (*g)[i].why == why {
			(*g)[i].names = append((*g)[i].names, name)
			return
		}
	}
	*g = append(*g, skipGroup{why: why, names: []string{name}})
}

// list prints each group as one Skipped line on out.
func (g skipGroups) list(out *os.File) {
	for _, group := range g {
		listSkipped(out, group.why, group.names)
	}
}

// reasons maps each skipped check's name to why the run left it out.
func (g skipGroups) reasons() map[string]string {
	why := map[string]string{}
	for _, group := range g {
		for _, name := range group.names {
			why[name] = group.why
		}
	}
	return why
}

// withSkipped puts a passing row for each skipped check back in its place
// among all, and tallies it in totals. Rows and findings of run checks stay.
func withSkipped(all []resolved, rows []render.Row, found [][]checkkit.Finding, skipped skipGroups, totals *render.Totals) ([]render.Row, [][]checkkit.Finding) {
	why := skipped.reasons()
	var outRows []render.Row
	var outFound [][]checkkit.Finding
	ran := 0
	for _, c := range all {
		if reason, ok := why[c.name]; ok {
			row := render.Row{Name: c.name, Ok: true, FilesChecked: -1, Skipped: reason}
			totals.Add(row)
			outRows, outFound = append(outRows, row), append(outFound, []checkkit.Finding{})
			continue
		}
		outRows, outFound = append(outRows, rows[ran]), append(outFound, found[ran])
		ran++
	}
	return outRows, outFound
}

// Why a run leaves a check out, as its Skipped line says.
const (
	noMatchingFiles = "no matching files"
	needsNoFile     = "the repo tracks no file its needs pattern names"
)

// applicable keeps the checks this run reaches and groups the rest by why
// skipReason leaves them out. shared is the run's shared tracked list,
// which needs patterns read before git is asked for one.
func applicable(root string, checks []resolved, env, shared []string) ([]resolved, skipGroups) {
	paths := scopedFiles(env)
	tracked := trackedList{root: root, paths: shared, listed: shared != nil}
	var kept []resolved
	var skipped skipGroups
	for _, c := range checks {
		if why := skipReason(c, paths, &tracked); why != "" {
			skipped.add(why, c.name)
			continue
		}
		kept = append(kept, c)
	}
	return kept, skipped
}

// skipReason says why the run leaves c out, or "" to run it. Needs must
// match a tracked file, and a scoped run's paths must reach the check.
func skipReason(c resolved, paths []string, tracked *trackedList) string {
	if r := checkkit.UnknownRequirement(c.desc); r != "" {
		return unknownWhy(r)
	}
	p, err := checkkit.PatternsOf(c.desc)
	switch {
	case err != nil:
		return ""
	case !tracked.meets(p):
		return needsNoFile
	case !p.ReachedBy(paths):
		return noMatchingFiles
	}
	return ""
}

// trackedList holds root's tracked files: the run's shared list, or one
// git lists once, the first time a check's needs pattern asks.
type trackedList struct {
	root   string
	paths  []string
	listed bool
	failed bool
}

// meets reports whether a tracked file matches p's needs pattern, listing
// files only for a check that has one. A failed git call meets every pattern.
func (t *trackedList) meets(p checkkit.Patterns) bool {
	if !p.HasNeeds() {
		return true
	}
	if !t.listed {
		paths, err := gitx.Repo(t.root).TrackedFiles()
		t.paths, t.failed, t.listed = paths, err != nil, true
	}
	return t.failed || p.Met(t.paths)
}

// unhonored says why a check whose pattern does not compile cannot run,
// naming the fix, or returns "" when its patterns compile.
func unhonored(name string, desc checkkit.Describe) string {
	_, err := checkkit.PatternsOf(desc)
	if err == nil {
		return ""
	}
	fix := "reinstall the check: " + installCommand(name)
	if _, local := localBins[name]; local {
		fix = "fix the pattern in the repo's own check"
	}
	return fmt.Sprintf("check %s %v; %s", name, err, fix)
}

// unknownWhy is the Skipped reason for a requirement this fitness does not
// know, which a newer fitness may.
func unknownWhy(r checkkit.Requirement) string {
	return fmt.Sprintf("requires %q, which this fitness does not know; upgrade fitness", r)
}

// blobWhy says why a text alone cannot reach a check, or "" when it can.
// The check may require the repo, the diff or the history.
func blobWhy(desc checkkit.Describe) string {
	if r := checkkit.UnknownRequirement(desc); r != "" {
		return unknownWhy(r)
	}
	for _, r := range []checkkit.Requirement{checkkit.RequiresRepo, checkkit.RequiresDiff, checkkit.RequiresHistory} {
		if checkkit.Requiring(desc, r) {
			return "requires " + string(r)
		}
	}
	return ""
}
