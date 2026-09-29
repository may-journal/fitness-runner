package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// runPlanCheck validates the triggering Plan issue, or every open Plan on
// workflow_dispatch, and comments the result once per body version. The plan
// check workflows call this in one step.
func runPlanCheck() int {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ev, err := readEvent()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness plan-check: "+err.Error())
		return 1
	}
	gh, err := newGHClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness plan-check: "+err.Error())
		return 1
	}
	return planCheck(ev, gh, newBodyChecker(root))
}

// planCheck is runPlanCheck past the environment: it validates each target
// Plan and comments unless this body version already has a comment.
func planCheck(ev ghEvent, gh githubAPI, c bodyChecker) int {
	targets, err := planTargets(ev, gh)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness plan-check: "+err.Error())
		return 1
	}
	failed, err := checkPlans(targets, gh, c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness plan-check: "+err.Error())
		return 1
	}
	if failed {
		fmt.Fprintln(os.Stderr, "One or more plan issues have violations.")
		return 1
	}
	return 0
}

// checkPlans validates and comments on each target, reporting whether any
// failed.
func checkPlans(targets []target, gh githubAPI, c bodyChecker) (failed bool, err error) {
	for _, t := range targets {
		errs := validatePlan(t, c)
		failed = failed || len(errs) > 0
		if err := commentOnce(t, errs, gh); err != nil {
			return failed, err
		}
	}
	return failed, nil
}

// planTargets is the triggering issue, or every open Plan on
// workflow_dispatch.
func planTargets(ev ghEvent, gh githubAPI) ([]target, error) {
	if ev.name == "workflow_dispatch" {
		return gh.openPlans()
	}
	if ev.Issue == nil {
		return nil, fmt.Errorf("no issue in the %q event", ev.name)
	}
	return []target{*ev.Issue}, nil
}

// validatePlan runs the plan-structure check and the prose checks on a body.
func validatePlan(t target, c bodyChecker) []string {
	errs := c.errorsOf("plan-structure", nil, t.Body)
	bodyFile, err := writeBody(t.Body)
	if err != nil {
		return append(errs, err.Error())
	}
	defer os.Remove(bodyFile)
	return append(errs, c.docErrors(bodyFile)...)
}

// commentOnce posts the result unless a comment already carries this body
// version's marker, so re-runs and unrelated edits stay quiet.
func commentOnce(t target, errs []string, gh githubAPI) error {
	mark := planMarker(t.Body)
	seen, err := hasMarker(t.Number, mark, gh)
	if err != nil || seen {
		return err
	}
	return gh.comment(t.Number, planComment(errs, mark))
}

// hasMarker reports whether a comment on issue n already carries mark.
func hasMarker(n int, mark string, gh githubAPI) (bool, error) {
	existing, err := gh.commentBodies(n)
	if err != nil {
		return false, err
	}
	for _, body := range existing {
		if strings.Contains(body, mark) {
			return true, nil
		}
	}
	return false, nil
}

// planMarker tags a comment with a hash of the body it judged; the format
// matches the earlier JavaScript, so existing comments still count.
func planMarker(body string) string {
	sum := sha256.Sum256([]byte(body))
	return "<!-- fitness:plan-structure:" + hex.EncodeToString(sum[:])[:12] + " -->"
}

// planComment is the verdict comment for a Plan body.
func planComment(errs []string, mark string) string {
	if len(errs) == 0 {
		return "✅ Plan looks good.\n\n" + mark
	}
	return "❌ Plan needs work:\n" + strings.TrimSuffix(bulletList(errs), "\n") + "\n\n" + mark
}
