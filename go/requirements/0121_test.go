package requirements

import (
	"strings"
	"testing"
)

// actionsJob is one job as the run's job list prints it: conclusion, start
// and end, tab-separated.
func actionsJob(conclusion, start, end string) string {
	return conclusion + "\t" + start + "\t" + end + "\n"
}

// minutesRun runs `fitness-release minutes` with args, gh answering saved.
func minutesRun(t *testing.T, saved []response, args ...string) (string, int, replays) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := releaseToolRun(t, releaseToolRepo(t, nil), rp.env, append([]string{"minutes"}, args...)...)
	return out, code, rp
}

func Test0121_1(t *testing.T) {
	t.Parallel()
	out, code, rp := minutesRun(t, []response{
		{Match: []string{"orgs/acme/repos?type=private"}, Stdout: "acme/app\n"},
		{Match: []string{"actions/runs/1/jobs"}, Stdout: actionsJob("success", "2026-10-20T10:00:00Z", "2026-10-20T10:00:08Z") +
			actionsJob("failure", "2026-10-20T10:01:00Z", "2026-10-20T10:01:12Z")},
		{Match: []string{"actions/runs/2/jobs"}, Stdout: actionsJob("success", "2026-10-20T11:00:00Z", "2026-10-20T11:01:01Z")},
		{Match: []string{"repos/acme/app/actions/runs?per_page=100&created=>="}, Stdout: "1\tComment check\n2\tFitness suite\n"},
	}, "--org", "acme", "--days", "4")
	sees(t, out, code, 0, "Actions jobs in 1 repos over the last 4 days",
		"| Comment check | 2 | 2 | 12 |", "| Fitness suite | 1 | 2 | 61 |", "| Total | 3 | 4 | |")
	if !rp.called("gh", "api", "repos/acme/app/actions/runs?per_page=100&created=>=") {
		t.Errorf("want the runs of acme/app read, got gh calls %q", rp.calls("gh"))
	}
}

func Test0121_2(t *testing.T) {
	t.Parallel()
	out, code, _ := minutesRun(t, []response{
		{Match: []string{"actions/runs/3/jobs"}, Stdout: actionsJob("skipped", "2026-10-20T12:00:00Z", "2026-10-20T12:00:00Z")},
		{Match: []string{"actions/runs/4/jobs"}, Stdout: actionsJob("success", "2026-10-20T12:00:00Z", "2026-10-20T12:00:09Z")},
		{Match: []string{"repos/acme/app/actions/runs?"}, Stdout: "3\tAuto-merge\n4\tPR check\n"},
	}, "--repos", "acme/app")
	sees(t, out, code, 0, "| PR check | 1 | 1 | 9 |", "| Total | 1 | 1 | |")
	if strings.Contains(out, "| Auto-merge |") {
		t.Errorf("want the skipped job left out, got:\n%s", out)
	}
}
