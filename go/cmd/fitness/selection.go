package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/selection"
)

func selectedChecks(root string, argv []string) ([]resolved, int, []string, error) {
	options, args, err := selection.Parse(argv)
	if err != nil {
		return nil, 0, nil, err
	}
	spec, _, jobs, passthrough := parseArgv(args)
	cfg, err := loadConfig(root, spec)
	if err != nil {
		return nil, 0, nil, err
	}
	checks, err := resolvePolicy(cfg, options, spec)
	return checks, jobs, passthrough, err
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
		return resolveList(names, nil)
	}
	warnLegacyChecks(cfg)
	return prepareChecks(cfg, spec)
}
