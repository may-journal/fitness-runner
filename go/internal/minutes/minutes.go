// Package minutes totals what GitHub Actions bills per workflow, rounding each
// job up to a whole minute as private repos are billed.
package minutes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/par"
)

// Row is one workflow's jobs over the window.
type Row struct {
	Workflow string
	Jobs     int
	Billed   int
	// MedianSeconds is the middle job's wall time, the upper middle for an
	// even count.
	MedianSeconds int
}

// Job is one job that ran, named by its run's workflow.
type Job struct {
	Workflow string
	Seconds  int
}

// Billed is the whole minutes GitHub bills for a job that ran seconds,
// never fewer than one.
func Billed(seconds int) int {
	return max(1, (seconds+59)/60)
}

// Table groups jobs by workflow, most billed minutes first, then by name.
func Table(jobs []Job) []Row {
	seconds := map[string][]int{}
	for _, j := range jobs {
		seconds[j.Workflow] = append(seconds[j.Workflow], j.Seconds)
	}
	rows := make([]Row, 0, len(seconds))
	for name, s := range seconds {
		slices.Sort(s)
		row := Row{Workflow: name, Jobs: len(s), MedianSeconds: s[len(s)/2]}
		for _, v := range s {
			row.Billed += Billed(v)
		}
		rows = append(rows, row)
	}
	slices.SortFunc(rows, func(a, b Row) int {
		if a.Billed != b.Billed {
			return b.Billed - a.Billed
		}
		return strings.Compare(a.Workflow, b.Workflow)
	})
	return rows
}

// Write prints rows as a Markdown table with a total row.
func Write(w io.Writer, rows []Row, days int, repos int) {
	fmt.Fprintf(w, "Actions jobs in %d repos over the last %d days\n\n", repos, days)
	fmt.Fprintln(w, "| Workflow | Jobs | Billed min | Median job s |")
	fmt.Fprintln(w, "| --- | --- | --- | --- |")
	jobs, billed := 0, 0
	for _, r := range rows {
		fmt.Fprintf(w, "| %s | %d | %d | %d |\n", r.Workflow, r.Jobs, r.Billed, r.MedianSeconds)
		jobs, billed = jobs+r.Jobs, billed+r.Billed
	}
	fmt.Fprintf(w, "| Total | %d | %d | |\n", jobs, billed)
}

// Client reads Actions runs and jobs through gh.
type Client struct{}

// Repos lists org's private repos that are not archived, the ones GitHub bills.
func (Client) Repos(ctx context.Context, org string) ([]string, error) {
	out, err := gh(ctx, "--paginate", "--jq", `.[] | select(.archived | not) | .full_name`,
		"orgs/"+org+"/repos?type=private&per_page=100")
	return lines(out), err
}

// Jobs lists every job that ran in repo's workflow runs created since since.
func (Client) Jobs(ctx context.Context, repo string, since time.Time) ([]Job, error) {
	out, err := gh(ctx, "--paginate", "--jq", `.workflow_runs[] | "\(.id)\t\(.name)"`,
		"repos/"+repo+"/actions/runs?per_page=100&created=>="+since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	runs := lines(out)
	found, err := par.MapErr(ctx, len(runs), 8, func(ctx context.Context, i int) ([]Job, error) {
		id, name, _ := strings.Cut(runs[i], "\t")
		return runJobs(ctx, repo, id, name)
	})
	return slices.Concat(found...), err
}

// runJobs lists the jobs of run id that started and finished; a skipped job
// started no runner, so it bills nothing.
func runJobs(ctx context.Context, repo, id, workflow string) ([]Job, error) {
	out, err := gh(ctx, "--paginate", "--jq",
		`.jobs[] | "\(.conclusion // "")\t\(.started_at // "")\t\(.completed_at // "")"`,
		"repos/"+repo+"/actions/runs/"+id+"/jobs?per_page=100&filter=all")
	if err != nil {
		return nil, err
	}
	var jobs []Job
	for _, line := range lines(out) {
		seconds, ran, err := jobSeconds(line)
		if err != nil {
			return nil, fmt.Errorf("job %q in run %s: %w", line, id, err)
		}
		if ran {
			jobs = append(jobs, Job{Workflow: workflow, Seconds: seconds})
		}
	}
	return jobs, nil
}

// jobSeconds reads one tab-separated job line and reports its wall time and
// whether it ran, since skipped or unfinished jobs did not.
func jobSeconds(line string) (int, bool, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != 3 {
		return 0, false, fmt.Errorf("want conclusion, start and end")
	}
	if fields[0] == "skipped" || fields[1] == "" || fields[2] == "" {
		return 0, false, nil
	}
	seconds, err := wallSeconds(fields[1], fields[2])
	return seconds, err == nil, err
}

// wallSeconds is the whole seconds from start to end, two RFC 3339 times.
func wallSeconds(start, end string) (int, error) {
	from, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return 0, err
	}
	to, err := time.Parse(time.RFC3339, end)
	return int(to.Sub(from).Seconds()), err
}

// chosen is repos, or org's private repos when repos names none.
func (c Client) chosen(ctx context.Context, org string, repos []string) ([]string, error) {
	switch {
	case len(repos) > 0:
		return repos, nil
	case org == "":
		return nil, fmt.Errorf("name the repos with --repos, or their org with --org")
	}
	return c.Repos(ctx, org)
}

// gh runs `gh api` with args and returns its output.
func gh(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", append([]string{"api"}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w: %s", args[len(args)-1], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// lines splits gh output into its non-blank lines, keeping their tabs.
func lines(out []byte) []string {
	var found []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			found = append(found, line)
		}
	}
	return found
}

// Print writes to w the billed table for repos, or for org's private repos
// when repos is empty, over the last days days.
func Print(ctx context.Context, w io.Writer, org string, repos []string, days int) error {
	if days <= 0 {
		return fmt.Errorf("--days must be a positive number of days, not %d", days)
	}
	var c Client
	repos, err := c.chosen(ctx, org, repos)
	if err != nil {
		return err
	}
	since := time.Now().AddDate(0, 0, -days)
	var all []Job
	for _, repo := range repos {
		jobs, err := c.Jobs(ctx, repo, since)
		if err != nil {
			return err
		}
		all = append(all, jobs...)
	}
	Write(w, Table(all), days, len(repos))
	return nil
}
