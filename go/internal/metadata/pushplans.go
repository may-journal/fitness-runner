package metadata

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
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

// approvalRe matches a comment whose first line starts with `I approve` in any
// case, so a mere mention such as "not approved" is none.
var approvalRe = regexp.MustCompile(`(?i)^[ \t]*I approve\b`)

// defaultApprovers are the associations GitHub gives the repo's owner and
// its org's members, who may approve when planApprovers is unset.
var defaultApprovers = []string{"OWNER", "MEMBER"}

// PushDeadline bounds every GitHub read one push makes, the gate's and
// local-branch's together, and any one GitHub read outside a push.
const PushDeadline = 30 * time.Second

// PushContext opens the deadline every GitHub read of one push shares.
func PushContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), PushDeadline)
}

// GitHubHost is the host gh reads from: GH_HOST, else github.com.
func GitHubHost() string {
	return cmp.Or(os.Getenv("GH_HOST"), "github.com")
}

// PushPlans are the Plans one push read successfully, each read once. A
// failed read is not kept, so a later ask tries GitHub again while the
// push's deadline lasts.
type PushPlans struct {
	ctx   context.Context
	reads map[string]planRead
}

// planRead is one successful read and the issue number it read.
type planRead struct {
	number int
	view   aftereffect.IssueView
}

// handoff is what the gate hands local-branch: its process ID, which must be
// local-branch's parent, the push's deadline, and the Plans it read.
type handoff struct {
	Gate     int        `json:"gate"`
	Deadline time.Time  `json:"deadline"`
	Plans    []pushPlan `json:"plans"`
}

// pushPlan is one handed-off read: the issue's repo, number, body and labels.
type pushPlan struct {
	Repo   string   `json:"repo"`
	Number int      `json:"number"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

// NewPushPlans starts a push with no Plans read, whose reads all end when
// ctx, the push's deadline from PushContext, does.
func NewPushPlans(ctx context.Context) *PushPlans {
	return &PushPlans{ctx: ctx, reads: map[string]planRead{}}
}

// TimedOut is one line naming the GitHub host and the deadline once the
// push's deadline has passed, else empty.
func (p *PushPlans) TimedOut() string {
	if p.ctx.Err() == nil {
		return ""
	}
	return fmt.Sprintf("%s did not answer within %s", GitHubHost(), PushDeadline)
}

// Approved reports whether the Plan ref names has the `Plan` label and an
// `I approve` comment by an allowed approver.
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

// allowed reports whether c's author may approve, as a listed login or, with no
// list, the repo's owner or an org member.
func allowed(c aftereffect.IssueComment, approvers []string) bool {
	if len(approvers) == 0 {
		return slices.Contains(defaultApprovers, c.Association)
	}
	return slices.ContainsFunc(approvers, func(login string) bool { return strings.EqualFold(login, c.Author) })
}

// read asks GitHub for the Plan ref names and keeps a successful read.
func (p *PushPlans) read(key string, ref gitx.PlanRef) (planRead, bool) {
	v, err := aftereffect.ViewIssue(p.ctx, ref.Slug(), ref.Number)
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
	deadline, _ := p.ctx.Deadline()
	data, err := json.Marshal(handoff{Gate: os.Getpid(), Deadline: deadline, Plans: p.plans()})
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

// handedPlans are the Plans the gate read, from a trusted handoff.
func handedPlans() []pushPlan {
	return trustedHandoff().Plans
}

// pushDeadline is when local-branch's GitHub reads must end: what is left
// of the gate's deadline inside pre-push, else PushDeadline from now.
func pushDeadline() time.Time {
	own := time.Now().Add(PushDeadline)
	if handed := trustedHandoff().Deadline; !handed.IsZero() && handed.Before(own) {
		return handed
	}
	return own
}

// trustedHandoff reads the gate's handoff inside pre-push, trusting only a
// private file written by this process's parent.
func trustedHandoff() handoff {
	path := os.Getenv(pushPlansEnv)
	if os.Getenv("FITNESS_INTERNAL_HOOK") != "pre-push" || !privateFile(path) {
		return handoff{}
	}
	if h := readHandoff(path); h.Gate == os.Getppid() {
		return h
	}
	return handoff{}
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
