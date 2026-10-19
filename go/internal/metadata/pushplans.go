package metadata

import (
	"cmp"
	"context"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
)

// pushPlansEnv names the file through which the pre-push gate hands
// local-branch the Plans it read.
const pushPlansEnv = "FITNESS_PUSH_PLANS_FILE"

var approveRe = regexp.MustCompile(`(?i)approve`)

// PushPlans are the Plans one push reads, each once. A nil read is a read
// that failed.
type PushPlans struct {
	reads map[string]*aftereffect.IssueView
}

// pushPlan is one handed-off read: the issue's repo, number, body and labels.
type pushPlan struct {
	Repo   string   `json:"repo"`
	Number int      `json:"number"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

// NewPushPlans starts a push with no Plans read.
func NewPushPlans() *PushPlans {
	return &PushPlans{reads: map[string]*aftereffect.IssueView{}}
}

// Approved reports whether issue n carries the exact `Plan` label and an
// approval comment, reading it from GitHub the first time it is asked.
func (p *PushPlans) Approved(n string) bool {
	v, seen := p.reads[n]
	if !seen {
		if read, err := aftereffect.ViewIssue(context.Background(), n); err == nil {
			v = &read
		}
		p.reads[n] = v
	}
	return v != nil && slices.Contains(v.Labels, "Plan") && slices.ContainsFunc(v.Comments, approveRe.MatchString)
}

// Handoff writes the reads to a file only this user can read or write and
// names it for local-branch. The returned func removes the file.
func (p *PushPlans) Handoff() (func(), error) {
	data, err := json.Marshal(p.plans())
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp("", "fitness-push-plans-*.json")
	if err != nil {
		return nil, err
	}
	cleanup := func() {
		_ = os.Remove(f.Name())
		_ = os.Unsetenv(pushPlansEnv)
	}
	_, err = f.Write(data)
	if err = cmp.Or(err, f.Close()); err != nil {
		cleanup()
		return nil, err
	}
	_ = os.Setenv(pushPlansEnv, f.Name())
	return cleanup, nil
}

// plans lists the successful reads that name their repo.
func (p *PushPlans) plans() []pushPlan {
	var plans []pushPlan
	for n, v := range p.reads {
		num, err := strconv.Atoi(n)
		if v != nil && v.Repo() != "" && err == nil {
			plans = append(plans, pushPlan{Repo: v.Repo(), Number: num, Body: v.Body, Labels: v.Labels})
		}
	}
	return plans
}

// seedPushIssues gives gh the pre-push gate's reads of repo's issues, so
// local-branch reads none of them again. It trusts only a regular file that
// no other user can read or write; anything else adds nothing, and the
// issues are read from GitHub as usual.
func seedPushIssues(gh aftereffect.Client, repo string) {
	for _, plan := range handedPlans() {
		if strings.EqualFold(plan.Repo, repo) {
			gh.Seed(plan.Number, plan.Body, plan.Labels)
		}
	}
}

// handedPlans reads the gate's handoff file inside pre-push; nil elsewhere.
func handedPlans() []pushPlan {
	path := os.Getenv(pushPlansEnv)
	if os.Getenv("FITNESS_INTERNAL_HOOK") != "pre-push" || !privateFile(path) {
		return nil
	}
	data, err := os.ReadFile(path)
	var plans []pushPlan
	if err != nil || json.Unmarshal(data, &plans) != nil {
		return nil
	}
	return plans
}

// privateFile reports whether path is a regular file no other user can
// read or write.
func privateFile(path string) bool {
	info, err := os.Lstat(path)
	return path != "" && err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o077 == 0
}
