package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/may-journal/fitness-runner/go/internal/boxes"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// runBoxes runs `fitness boxes <report.json> <page.html>`: it draws a
// `--json` report as an HTML box diagram, one box per check.
func runBoxes(args []string) int {
	if err := writeBoxes(args); err != nil {
		report.Error("fitness boxes", err)
		return 1
	}
	return 0
}

// writeBoxes reads the report and writes the page.
func writeBoxes(args []string) error {
	if len(args) != 2 {
		return errors.New("use fitness boxes <report.json> <page.html>")
	}
	r, err := readReport(args[0])
	if err != nil {
		return err
	}
	return writePage(args[1], r)
}

// readReport loads and checks a `--json` report.
func readReport(path string) (boxes.Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return boxes.Report{}, err
	}
	r, err := boxes.Parse(data)
	if err != nil {
		return boxes.Report{}, fmt.Errorf("%s: %w", path, err)
	}
	return r, nil
}

// writePage renders the diagram in memory first, so a failed render leaves
// no half-written page, then writes it, creating its folder.
func writePage(path string, r boxes.Report) error {
	var page bytes.Buffer
	if err := boxes.Write(&page, r); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, page.Bytes(), 0o644)
}
