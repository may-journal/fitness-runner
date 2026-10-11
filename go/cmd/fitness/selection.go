package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/selection"
)

func selectedChecks(root string, argv []string) ([]resolved, []string, error) {
	options, args, err := selection.Parse(argv)
	if err != nil {
		return nil, nil, err
	}
	spec, passthrough := parseArgv(args)
	cfg, err := loadConfig(root)
	if err != nil {
		return nil, nil, err
	}
	checks, err := resolvePolicy(cfg, options, spec)
	return commitChecks(checks), passthrough, err
}

// commitChecks excludes branch metadata from pre-commit, including installer hooks.
func commitChecks(checks []resolved) []resolved {
	if os.Getenv("FITNESS_INTERNAL_HOOK") != "pre-commit" {
		return checks
	}
	var local []resolved
	for _, check := range checks {
		if check.name != "local-branch" {
			local = append(local, check)
		}
	}
	return local
}

func resolvePolicy(cfg *conf.Config, options selection.Options, spec string) ([]resolved, error) {
	var policy string
	var configured []string
	if cfg != nil {
		policy, configured = cfg.Policy, cfg.LegacyChecks
	}
	policy, names, err := options.Resolve(policy, configured, spec, allChecks, "go-test-coverage")
	if err != nil {
		return nil, err
	}
	if policy == selection.External {
		fmt.Fprintln(os.Stderr, "fitness: policy=external checks="+strings.Join(names, ","))
		checks, err := resolveList(names)
		return withBudgets(cfg, checks), err
	}
	return prepareChecks(cfg, spec)
}
