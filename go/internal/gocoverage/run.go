package gocoverage

import (
	"context"
	"fmt"
	"github.com/may-journal/fitness-runner/go/internal/toolchain"
	"os"
	"os/exec"
	"path/filepath"
)

type Options struct {
	Entries []string
	Audit   bool
}

func Run(ctx context.Context, root string, options Options) (Report, error) {
	mods := toolchain.ScopedModules(root)
	if len(mods) == 0 {
		return Report{}, nil
	}
	bin, err := exec.LookPath("go")
	if err != nil {
		return Report{}, fmt.Errorf("%s", toolchain.NotInstalled)
	}
	temp, err := os.MkdirTemp("", "fitness-go-coverage-")
	if err != nil {
		return Report{}, err
	}
	defer os.RemoveAll(temp)
	return runModules(ctx, root, bin, temp, mods, options)
}
func runModules(ctx context.Context, root, bin, temp string, mods []string, options Options) (Report, error) {
	var result Report
	found := map[string]bool{}
	for _, mod := range mods {
		r := runner{ctx: ctx, dir: filepath.Join(root, mod), bin: bin, temp: temp}
		report, err := r.module(options, found)
		if err != nil {
			return result, fmt.Errorf("%s: %w", mod, err)
		}
		result.append(report)
	}
	for _, entry := range options.Entries {
		if !found[entry] {
			return result, fmt.Errorf("unknown entry package %q", entry)
		}
	}
	result.sort()
	return result, nil
}
func (r runner) module(options Options, found map[string]bool) (Report, error) {
	packages, err := r.inventory()
	if err != nil {
		return Report{}, err
	}
	selected := localEntries(packages, options.Entries, found)
	entry, err := entries(packages, selected)
	if err != nil {
		return Report{}, err
	}
	p, err := r.measure("./...", "")
	if err != nil {
		return Report{}, err
	}
	if err := validateInventory(packages, p); err != nil {
		return Report{}, err
	}
	return r.report(packages, p, entry, options.Audit)
}
func (r runner) report(packages []Package, p Profile, entry map[string]bool, audit bool) (Report, error) {
	report, err := summarize(packages, p, entry)
	if err != nil {
		return report, err
	}
	if audit {
		report.Findings, err = r.audit(packages)
	}
	return report, err
}
func localEntries(packages []Package, requested []string, found map[string]bool) []string {
	known := map[string]bool{}
	for _, pkg := range packages {
		known[pkg.ImportPath] = true
	}
	var local []string
	for _, entry := range requested {
		if known[entry] {
			local = append(local, entry)
			found[entry] = true
		}
	}
	return local
}
func (r *Report) append(other Report) {
	r.Rows = append(r.Rows, other.Rows...)
	r.Gaps = append(r.Gaps, other.Gaps...)
	r.Findings = append(r.Findings, other.Findings...)
	r.Failures = append(r.Failures, other.Failures...)
}
