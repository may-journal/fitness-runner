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
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// pushPlansEnv names the file through which the pre-push gate hands
// local-branch the Plans it read.
const pushPlansEnv = "FITNESS_PUSH_PLANS_FILE"

// approvalRe matches a comment whose first line starts with `I approve`, in
// any case, so a mere mention of approval, such as "not approved", is none.
var approvalRe = regexp.MustCompile(`(?i)^[ \t]*I approve\b`)

// defaultApprovers are the associations GitHub gives the repo's owner and
// its org's members, who may approve when planApprovers is unset.
var defaultApprovers = []string{"OWNER", "MEMBER"}

// planReadTimeout bounds one Plan read, as local-branch bounds its own.
const planReadTimeout = 30 * time.Second

// PushPlans are the Plans one push read successfully, each read once. A
// failed read is not kept, so a later ask tries GitHub again.
type PushPlans struct {
	reads map[string]planRead
}

// planRead is one successful read and the issue number it read.
type planRead struct {
	number int
	view   aftereffect.IssueView
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
	return &PushPlans{reads: map[string]planRead{}}
}

// Approved reports whether the Plan ref names carries the exact `Plan` label
// and a comment whose first line starts with `I approve` by an allowed
// approver, reading it from GitHub until a read succeeds. A qualified ref is
// read from the repo it names; a bare one from this repo. Approvers lists the
// logins allowed; with none, the repo's owner and its org's members are.
func (p *PushPlans) Approved(ref gitx.PlanRef, approvers []string) bool {
	key := strings.ToLower(ref.String())
	r, ok := p.reads[key]
	if !ok {
		r, ok = p.read(key, ref)
	}
	return ok && slices.Contains(r.view.Labels, "Plan") && slices.ContainsFunc(r.view.Comments, func(c aftereffect.IssueComment) bool {
		return approvalRe.MatchString(c.Body) && allowed(c, approvers)
	})
}

// allowed reports whether c's author may approve: a login approvers lists,
// compared case-insensitively, or with no list the repo's owner or an org
// member.
func allowed(c aftereffect.IssueComment, approvers []string) bool {
	if len(approvers) == 0 {
		return slices.Contains(defaultApprovers, c.Association)
	}
	return slices.ContainsFunc(approvers, func(login string) bool { return strings.EqualFold(login, c.Author) })
}

// read asks GitHub for the Plan ref names and keeps a successful read.
func (p *PushPlans) read(key string, ref gitx.PlanRef) (planRead, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), planReadTimeout)
	defer cancel()
	v, err := aftereffect.ViewIssue(ctx, ref.Slug(), ref.Number)
	num, numErr := strconv.Atoi(ref.Number)
	if err = cmp.Or(err, numErr); err != nil {
		return planRead{}, false
	}
	r := planRead{number: num, view: v}
	p.reads[key] = r
	return r, true
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
	for _, r := range p.reads {
		if v := r.view; v.Repo() != "" {
			plans = append(plans, pushPlan{Repo: v.Repo(), Number: r.number, Body: v.Body, Labels: v.Labels})
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
