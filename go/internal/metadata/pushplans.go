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
	"time"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
)

// pushPlansEnv names the file through which the pre-push gate hands
// local-branch the Plans it read.
const pushPlansEnv = "FITNESS_PUSH_PLANS_FILE"

var approveRe = regexp.MustCompile(`(?i)approve`)

// planReadTimeout bounds one Plan read, as local-branch bounds its own.
const planReadTimeout = 30 * time.Second

// PushPlans are the Plans one push read successfully, each read once. A
// failed read is not kept, so a later ask tries GitHub again.
type PushPlans struct {
	reads map[string]aftereffect.IssueView
}

// handoff is what the gate hands local-branch: its own process ID, which
// must be local-branch's parent, and the Plans it read.
type handoff struct {
	Gate  int        `json:"gate"`
	Plans []pushPlan `json:"plans"`
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
	return &PushPlans{reads: map[string]aftereffect.IssueView{}}
}

// Approved reports whether issue n carries the exact `Plan` label and an
// approval comment, reading it from GitHub until a read succeeds.
func (p *PushPlans) Approved(n string) bool {
	v, ok := p.reads[n]
	if !ok {
		v, ok = p.read(n)
	}
	return ok && slices.Contains(v.Labels, "Plan") && slices.ContainsFunc(v.Comments, approveRe.MatchString)
}

// read asks GitHub for issue n and keeps a successful read.
func (p *PushPlans) read(n string) (aftereffect.IssueView, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), planReadTimeout)
	defer cancel()
	v, err := aftereffect.ViewIssue(ctx, n)
	if err != nil {
		return v, false
	}
	p.reads[n] = v
	return v, true
}

// Handoff writes the reads to a file only this user can read or write and
// names it for local-branch. The returned func removes the file.
func (p *PushPlans) Handoff() (func(), error) {
	data, err := json.Marshal(handoff{Gate: os.Getpid(), Plans: p.plans()})
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
		if v.Repo() != "" && err == nil {
			plans = append(plans, pushPlan{Repo: v.Repo(), Number: num, Body: v.Body, Labels: v.Labels})
		}
	}
	return plans
}

// seedPushIssues gives gh the pre-push gate's reads of repo's issues, so
// local-branch reads none of them again. Without a trusted handoff it adds
// nothing, and the issues are read from GitHub as usual.
func seedPushIssues(gh aftereffect.Client, repo string) {
	for _, plan := range handedPlans() {
		if strings.EqualFold(plan.Repo, repo) {
			gh.Seed(plan.Number, plan.Body, plan.Labels)
		}
	}
}

// handedPlans reads the gate's handoff inside pre-push. It trusts only a
// private file written by this process's parent, so a file set up by hand
// for a plain local-branch run seeds nothing.
func handedPlans() []pushPlan {
	path := os.Getenv(pushPlansEnv)
	if os.Getenv("FITNESS_INTERNAL_HOOK") != "pre-push" || !privateFile(path) {
		return nil
	}
	if h := readHandoff(path); h.Gate == os.Getppid() {
		return h.Plans
	}
	return nil
}

// readHandoff decodes the handoff at path; an unreadable one is empty.
func readHandoff(path string) handoff {
	var h handoff
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &h) != nil {
		return handoff{}
	}
	return h
}

// privateFile reports whether path is a regular file no other user can
// read or write.
func privateFile(path string) bool {
	info, err := os.Lstat(path)
	return path != "" && err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o077 == 0
}
