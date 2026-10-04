// Command fitness-check-go-test-coverage measures entry coverage and test overlap.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/gocoverage"
)

func main() {
	checkkit.Main(checkkit.Check{Describe: checkkit.Describe{Name: "go-test-coverage", TimeoutMs: 900000}, Run: run})
}
func run(root string, args []string) (checkkit.Result, error) {
	options, err := parse(root, args)
	if err != nil {
		return checkkit.Fail(0, err.Error()), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	report, err := gocoverage.Run(ctx, root, options)
	if err != nil {
		return checkkit.Fail(0, err.Error()), nil
	}
	return finish(root, options.AuditMap, report)
}
func finish(root, name string, report gocoverage.Report) (checkkit.Result, error) {
	if err := report.WriteAudit(root, name); err != nil {
		return checkkit.Fail(0, err.Error()), nil
	}
	for _, line := range report.Lines() {
		fmt.Fprintln(os.Stderr, line)
	}
	if len(report.Failures) > 0 {
		return checkkit.Fail(report.FileCount(), report.Failures...), nil
	}
	return checkkit.Pass(report.FileCount()), nil
}
func parse(root string, args []string) (gocoverage.Options, error) {
	var options gocoverage.Options
	config, err := conf.Load(root)
	if err != nil {
		return options, err
	}
	if config != nil {
		options.Entries = config.GoTestCoverage.Entries
	}
	return parseFlags(options, args)
}
func parseFlags(options gocoverage.Options, args []string) (gocoverage.Options, error) {
	flags := flag.NewFlagSet("go-test-coverage", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	entries := flags.String("entry", strings.Join(options.Entries, ","), "comma-separated library entry import paths")
	flags.StringVar(&options.AuditMap, "audit-map", "", "write a JSON source map to a new file outside the repository")
	flags.BoolVar(&options.Audit, "audit", false, "audit test coverage overlap")
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() > 0 {
		return options, fmt.Errorf("unexpected coverage arguments: %v", flags.Args())
	}
	if err := validateExplicitEntries(flags, *entries, options.Entries); err != nil {
		return options, err
	}
	parsed, err := parseEntries(*entries)
	options.Entries = parsed
	return options, err
}
func parseEntries(value string) ([]string, error) {
	if value == "" {
		return nil, nil
	}
	entries := strings.Split(value, ",")
	seen := map[string]bool{}
	for _, entry := range entries {
		if invalidEntry(entry) || seen[entry] {
			return nil, fmt.Errorf("empty, repeated, or invalid entry package %q", entry)
		}
		seen[entry] = true
	}
	return entries, nil
}

func invalidEntry(entry string) bool { return strings.TrimSpace(entry) != entry || entry == "" }

func validateExplicitEntries(flags *flag.FlagSet, value string, configured []string) error {
	explicit := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "entry" {
			explicit = true
		}
	})
	if value == "" && (explicit || configured != nil) {
		return fmt.Errorf("entry selection must not be empty")
	}
	return nil
}
