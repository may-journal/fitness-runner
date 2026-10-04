package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const summaryLimit = 1024 * 1024

// WriteSummary appends a bounded summary, preserving overflow in a local report.
// The report path is disclosed in both the summary and the job log for upload.
func WriteSummary(md string) {
	if !Enabled() {
		return
	}
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		summaryFallback(md, fmt.Errorf("GITHUB_STEP_SUMMARY is unset"))
		return
	}
	if err := appendSummary(path, md); err != nil {
		summaryFallback(md, err)
	}
}

func appendSummary(path, md string) error {
	complete, err := existingSummary(path)
	if err != nil {
		return err
	}
	complete = append(complete, md...)
	if len(complete) <= summaryLimit {
		return os.WriteFile(path, complete, 0o644)
	}
	return overflowSummary(path, complete)
}

func existingSummary(path string) ([]byte, error) {
	complete, err := os.ReadFile(path + ".fitness-report.md")
	if err == nil {
		return complete, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	complete, err = os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return complete, err
}

func overflowSummary(path string, complete []byte) error {
	report := path + ".fitness-report.md"
	if err := os.WriteFile(report, complete, 0o644); err != nil {
		return err
	}
	publishReport(path, complete)
	notice := "\n\n**Summary truncated.** Complete report saved on the runner at " + EscapeMarkdown(report) + ". The complete report is retained for workflow artifact upload when available; check the workflow run for uploaded artifacts.\n"
	fmt.Fprintf(os.Stderr, "Fitness summary exceeded 1 MiB; complete report saved at %s\n", report)
	budget := max(0, summaryLimit-len(notice))
	prefix := summaryPrefix(complete, budget)
	return os.WriteFile(path, append(prefix, notice...), 0o644)
}

func summaryFallback(md string, err error) {
	fmt.Fprintf(os.Stderr, "Could not write GitHub job summary: %v\n", err)
	// Prefix lines so diagnostic text cannot become executable workflow commands.
	fmt.Fprintln(os.Stderr, "Fitness report: "+strings.ReplaceAll(strings.ReplaceAll(md, "\r", "\n"), "\n", "\nFitness report: "))
}

func publishReport(path string, complete []byte) {
	root := os.Getenv("RUNNER_TEMP")
	if root == "" {
		summaryFallback(string(complete), fmt.Errorf("RUNNER_TEMP unavailable for artifact report"))
		return
	}
	destination := filepath.Join(root, "fitness-reports")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		summaryFallback(string(complete), err)
		return
	}
	report := filepath.Join(destination, filepath.Base(path)+".md")
	if err := os.WriteFile(report, complete, 0o644); err != nil {
		summaryFallback(string(complete), err)
		return
	}
	fmt.Fprintf(os.Stderr, "Complete Fitness report available for artifact upload: %s\n", report)
}

func summaryPrefix(complete []byte, budget int) []byte {
	normalized := strings.ToValidUTF8(string(complete), "�")
	end := min(budget, len(normalized))
	for end > 0 && end < len(normalized) && !utf8.RuneStart(normalized[end]) {
		end--
	}
	return []byte(normalized[:end])
}
