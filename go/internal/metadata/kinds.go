package metadata

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/epic"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
)

// kindRule is what a kind adds beyond its template's shape: checks that
// judge the body, and links, which judge what the body points at on GitHub.
type kindRule struct {
	checks []string
	links  func(t target, gh aftereffect.Client) []string
}

// kindRules are the rules each kind label adds. A kind with no entry is
// judged by its template's shape and the doc checks alone, so a new kind is
// one template file.
var kindRules = map[string]kindRule{
	"Plan": {checks: []string{"plan-structure"}},
	"Epic": {checks: []string{"epic-structure"}, links: subIssueErrors},
}

// planChecks are the checks a Plan body gets, for branch-check.
var planChecks = slices.Concat(kindRules["Plan"].checks, docChecks)

// issueKind is one kind of labeled issue: the label that marks it, the shape
// its template gives when no check of its own judges that, and its rule.
type issueKind struct {
	label string
	spec  *mdtemplate.Spec
	rule  kindRule
}

// discoverKinds reads the kinds root's issue templates name, keyed by each
// template's first label, beside every kind in kindRules a repo has no
// template for. A kind whose rule names checks leaves its template's shape to
// them, as plan-structure and epic-structure read the same template, so one
// source sets the sections and no two specs disagree.
func discoverKinds(root string) map[string]issueKind {
	kinds := map[string]issueKind{}
	for label, rule := range kindRules {
		kinds[label] = issueKind{label: label, rule: rule}
	}
	for label, template := range mdtemplate.IssueTemplates(root) {
		k := issueKind{label: label, rule: kindRules[label]}
		if len(k.rule.checks) == 0 {
			spec := mdtemplate.KindSpec(template, strings.ToLower(label))
			k.spec = &spec
		}
		kinds[label] = k
	}
	return kinds
}

// onlyKind limits discoverKinds to one label, for the plan-check alias.
func onlyKind(label string) func(root string) map[string]issueKind {
	return func(root string) map[string]issueKind {
		return map[string]issueKind{label: discoverKinds(root)[label]}
	}
}

// classify picks t's kind from its labels. An issue with no kind label does
// not apply; one with two judges by the first and reports the second.
func classify(t target, kinds map[string]issueKind) (issueKind, []string, bool) {
	var found []string
	for _, l := range t.Labels {
		if _, ok := kinds[l]; ok {
			found = append(found, l)
		}
	}
	switch len(found) {
	case 0:
		return issueKind{}, nil, false
	case 1:
		return kinds[found[0]], nil, true
	}
	return kinds[found[0]], []string{fmt.Sprintf("the issue has %d kind labels (%s); keep the one label that names its kind", len(found), strings.Join(found, ", "))}, true
}

// judge runs the kind's template shape, its checks and the doc checks on
// t's body, then its links, reporting each finding once.
func (k issueKind) judge(t target, gh aftereffect.Client, b blobChecker) []string {
	var errs []string
	if k.spec != nil {
		errs = mdtemplate.Validate(t.Body, *k.spec)
	}
	errs = append(errs, b.judge(slices.Concat(k.rule.checks, docChecks), t.Body)...)
	if k.rule.links != nil {
		errs = append(errs, k.rule.links(t, gh)...)
	}
	return unique(errs)
}

// unique drops repeated findings, keeping the first of each.
func unique(errs []string) []string {
	var out []string
	for _, e := range errs {
		if !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out
}

// issueTargets is the triggering issue with the labels its event carries,
// or every open issue with a kind label on workflow_dispatch.
func issueTargets(ev ghEvent, gh aftereffect.Client, b blobChecker) ([]target, error) {
	if ev.name == "workflow_dispatch" {
		return openOfKinds(gh, discoverKinds(b.root))
	}
	ts, err := eventTarget(ev.name, "issue", ev.Issue)
	if err != nil {
		return nil, err
	}
	return ts, eventLabels(&ts[0], gh)
}

// eventLabels sets t's labels from its event payload, reading them from
// GitHub only for a payload that carries none. A read that finds the issue
// gone marks it so, and it is noted and skipped, not failed.
func eventLabels(t *target, gh aftereffect.Client) error {
	if t.EventLabels != nil {
		for _, l := range t.EventLabels {
			t.Labels = append(t.Labels, l.Name)
		}
		return nil
	}
	_, labels, err := gh.Issue(t.Number)
	t.Labels, t.Gone = labels, errors.Is(err, aftereffect.ErrIssueGone)
	if t.Gone {
		return nil
	}
	return err
}

// openOfKinds lists the open issues of every kind once each, by number.
func openOfKinds(gh aftereffect.Client, kinds map[string]issueKind) ([]target, error) {
	byNumber := map[int]target{}
	for _, l := range slices.Sorted(maps.Keys(kinds)) {
		ts, err := gh.OpenLabeled(l)
		if err != nil {
			return nil, err
		}
		for _, t := range ts {
			byNumber[t.Number] = t
		}
	}
	out := slices.Collect(maps.Values(byNumber))
	slices.SortFunc(out, func(a, b target) int { return a.Number - b.Number })
	return out, nil
}

// planTargets is the triggering issue, or every open Plan on
// workflow_dispatch, each judged as a Plan.
func planTargets(ev ghEvent, gh aftereffect.Client, _ blobChecker) ([]target, error) {
	var ts []target
	var err error
	if ev.name == "workflow_dispatch" {
		ts, err = gh.OpenLabeled("Plan")
	} else {
		ts, err = eventTarget(ev.name, "issue", ev.Issue)
	}
	for i := range ts {
		ts[i].Labels = []string{"Plan"}
	}
	return ts, err
}

// subIssueErrors compares an epic's listed issues with its GitHub
// sub-issues: every listed issue must be a sub-issue and exist, and every
// sub-issue must be listed.
func subIssueErrors(t target, gh aftereffect.Client) []string {
	subs, err := gh.SubIssues(t.Number)
	if err != nil {
		return []string{fmt.Sprintf("cannot read the sub-issues of #%d: %v; check gh authentication and network access, then retry", t.Number, err)}
	}
	refs := epic.Refs(t.Body)
	return append(listedErrors(refs, subs, gh), unlistedErrors(refs, subs, gh.Repo())...)
}

// listedErrors names each listed reference that is not a sub-issue.
func listedErrors(refs, subs []string, gh aftereffect.Client) []string {
	var errs []string
	for _, ref := range refs {
		full := epic.Key(ref, gh.Repo())
		if !slices.ContainsFunc(subs, func(s string) bool { return epic.Key(s, "") == full }) {
			errs = append(errs, notSubIssue(ref, full, gh.Repo(), gh))
		}
	}
	return errs
}

// unlistedErrors names each sub-issue the list leaves out.
func unlistedErrors(refs, subs []string, repo string) []string {
	var errs []string
	for _, s := range subs {
		if !slices.ContainsFunc(refs, func(r string) bool { return epic.Key(r, repo) == epic.Key(s, repo) }) {
			errs = append(errs, fmt.Sprintf("sub-issue %s is not listed under `## %s`; list it, or remove it as a sub-issue", shortRef(s, repo), epic.Section))
		}
	}
	return errs
}

// notSubIssue is the finding for a listed issue that is not a sub-issue:
// one in this repository that cannot be read is missing.
func notSubIssue(ref, full, repo string, gh aftereffect.Client) string {
	if n, same := sameRepoNumber(full, repo); same {
		if _, _, err := gh.Issue(n); err != nil {
			return fmt.Sprintf("%s is listed but cannot be read (%v); list only issues that exist", ref, err)
		}
	}
	return fmt.Sprintf("%s is listed but is not a sub-issue of this epic; add it as a sub-issue", ref)
}

// shortRef is a full reference as #number when it is in repo.
func shortRef(full, repo string) string {
	if _, same := sameRepoNumber(strings.ToLower(full), repo); same {
		return full[strings.LastIndex(full, "#"):]
	}
	return full
}

// sameRepoNumber is the number of a lower-case full reference in repo.
func sameRepoNumber(full, repo string) (int, bool) {
	name, num, _ := strings.Cut(full, "#")
	n, err := strconv.Atoi(num)
	return n, err == nil && strings.EqualFold(name, repo)
}
