package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/render"
)

// jsonFlag names the file that receives the run's performance report.
const jsonFlag = "--json"

// perfReport is the `--json` record of one run: how long it took, how big
// the runner is, and each check's cost. CI keeps it as an artifact, so the
// trail of runs shows which checks slow down or grow.
type perfReport struct {
	ElapsedMs   int64       `json:"elapsedMs"`
	RunnerBytes int64       `json:"runnerBytes"`
	Checks      []perfCheck `json:"checks"`
}

// perfCheck is one check's cost: wall time against its budget, files judged,
// and the size of its binary.
type perfCheck struct {
	Name         string `json:"name"`
	Ok           bool   `json:"ok"`
	Ms           int64  `json:"ms"`
	TimeoutMs    int64  `json:"timeoutMs"`
	TimedOut     bool   `json:"timedOut"`
	FilesChecked int    `json:"filesChecked"`
	BinaryBytes  int64  `json:"binaryBytes"`
}

// jsonPath pulls `--json=<file>` out of argv, so neither check selection nor
// a single check's passthrough sees it.
func jsonPath(argv []string) (path string, rest []string, err error) {
	var flags []string
	for _, a := range argv {
		if a == jsonFlag || strings.HasPrefix(a, jsonFlag+"=") {
			flags = append(flags, a)
			continue
		}
		rest = append(rest, a)
	}
	return onePath(flags, rest)
}

// onePath accepts no `--json` flag, or exactly one that names a file.
func onePath(flags, rest []string) (string, []string, error) {
	if len(flags) == 0 {
		return "", rest, nil
	}
	path := strings.TrimPrefix(flags[0], jsonFlag+"=")
	if len(flags) > 1 || path == "" || path == jsonFlag {
		return "", nil, errors.New("use --json=<file> once")
	}
	return path, rest, nil
}

// buildPerfReport pairs each check with its row, in dispatch order.
func buildPerfReport(checks []resolved, outcomes []outcome, rows []render.Row, elapsedMs int64) perfReport {
	runner, _ := os.Executable()
	r := perfReport{ElapsedMs: elapsedMs, RunnerBytes: fileBytes(runner), Checks: make([]perfCheck, len(checks))}
	for i, c := range checks {
		r.Checks[i] = perfCheck{
			Name:         c.name,
			Ok:           rows[i].Ok,
			Ms:           rows[i].Ms,
			TimeoutMs:    timeoutFor(c).Milliseconds(),
			TimedOut:     outcomes[i].timedOut,
			FilesChecked: rows[i].FilesChecked,
			BinaryBytes:  fileBytes(c.bin),
		}
	}
	return r
}

// writePerfReport writes the report as indented JSON, creating its folder.
func writePerfReport(path string, r perfReport) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// fileBytes returns a file's size, or 0 when it cannot be read.
func fileBytes(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
