package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// targetReport keeps issue-body diagnostics without source locations and caps the whole sweep.
type targetReport struct {
	operation, kind string
	summary         strings.Builder
	diagnostics     []string
	targets         []int
}

func newTargetReport(operation, kind string) *targetReport {
	return &targetReport{operation: operation, kind: kind}
}

func (r *targetReport) add(number int, errs []string) {
	r.targets = append(r.targets, number)
	fmt.Fprintf(&r.summary, "### %s #%d\n\n", r.operation, number)
	if len(errs) == 0 {
		r.summary.WriteString("Passed.\n\n")
	}
	r.summary.WriteString(aftereffect.BulletList(errs))
	for _, message := range errs {
		r.diagnostics = append(r.diagnostics, fmt.Sprintf("#%d: %s", number, message))
	}
}

func (r *targetReport) links() string {
	var b strings.Builder
	for _, number := range r.targets {
		b.WriteString(targetLink(r.kind, number))
	}
	return b.String()
}

func (r *targetReport) emit() {
	report.EmitAnnotations(r.operation, r.diagnostics, false)
}

func (r *targetReport) finish() {
	fmt.Fprintf(&r.summary, "\n%d targets checked; %d findings.\n", len(r.targets), len(r.diagnostics))
	fmt.Print(r.summary.String())
	report.WriteSummary(r.summary.String() + r.links())
	r.emit()
}

// targetURL uses environment metadata, never temporary body-file locations.
func targetURL(kind string, number int) string {
	server := os.Getenv("GITHUB_SERVER_URL")
	if server == "" {
		server = "https://github.com"
	}
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return ""
	}
	link, err := url.JoinPath(server, repo, kind, fmt.Sprint(number))
	if err != nil {
		return ""
	}
	return strings.NewReplacer("(", "%28", ")", "%29").Replace(link)
}

// targetLink is shared by successful and failed target operations.
func targetLink(kind string, number int) string {
	if link := targetURL(kind, number); link != "" {
		return fmt.Sprintf("\n[View #%d](%s)\n", number, link)
	}
	return ""
}
