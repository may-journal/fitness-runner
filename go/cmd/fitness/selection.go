package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/selection"
	"github.com/may-journal/fitness-runner/go/pkg/conf"
	"github.com/may-journal/fitness-runner/go/pkg/gitx"
	"github.com/may-journal/fitness-runner/go/pkg/lang"
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
	if err := loadLocalChecks(cfg, root); err != nil {
		return nil, nil, err
	}
	checks, err := resolvePolicy(cfg, options, spec)
	if err == nil {
		err = newEnough(checks)
	}
	return checks, passthrough, markRequired(checks, cfg, options, err)
}

// markRequired marks each check that requiredChecks or --require names, so
// its internal error fails the run. It returns failed, else an error for a
// name that is no check.
func markRequired(checks []resolved, cfg *conf.Config, options selection.Options, failed error) error {
	if failed != nil {
		return failed
	}
	var configured []string
	if cfg != nil {
		configured = cfg.RequiredChecks
	}
	names, err := options.Require(configured, selectableChecks())
	for i := range checks {
		checks[i].required = slices.Contains(names, checks[i].name)
	}
	if err != nil {
		return err
	}
	return errors.Join(outsideRun(options.Required, checks), turnedOffRequired(cfg))
}

// outsideRun is an error for a check --require names that the run leaves
// out, since it could never prove that check ran.
func outsideRun(flagged []string, checks []resolved) error {
	for _, name := range flagged {
		if !slices.ContainsFunc(checks, func(c resolved) bool { return c.name == name }) {
			return fmt.Errorf("--require names %s, which this run leaves out; add it to the run or drop it from --require", name)
		}
	}
	return nil
}

// turnedOffRequired is an error for a check that requiredChecks names and
// disabledChecks turns off.
func turnedOffRequired(cfg *conf.Config) error {
	if cfg == nil {
		return nil
	}
	for _, name := range cfg.RequiredChecks {
		if slices.Contains(cfg.DisabledChecks, name) {
			return fmt.Errorf("requiredChecks names %s, which disabledChecks turns off; remove it from one of them", name)
		}
	}
	return nil
}

// commitChecks excludes branch metadata from a pre-commit run, and also
// semantic-commit and requirements-changed when a commit-msg hook judges the message.
func commitChecks(checks []resolved) ([]resolved, error) {
	if os.Getenv("FITNESS_INTERNAL_HOOK") != "pre-commit" {
		return checks, nil
	}
	messageHook, err := commitMsgHook()
	var local []resolved
	for _, check := range checks {
		if !skipsAtCommit(check.name, messageHook) {
			local = append(local, check)
		}
	}
	return local, err
}

// skipsAtCommit reports whether pre-commit leaves check out, given whether a
// commit-msg hook judges the new message.
func skipsAtCommit(check string, messageHook bool) bool {
	return check == "local-branch" || messageHook && slices.Contains(messageChecks, check)
}

// messageChecks are the checks pre-commit leaves to a commit-msg hook.
var messageChecks = []string{"semantic-commit", "requirements-changed"}

// commitMsgHook reports whether the repository has an executable commit-msg
// hook, in core.hooksPath or the git directory. A stalled git call returns
// its error.
func commitMsgHook() (bool, error) {
	path, err := gitx.Repo(".").GitPath("hooks/commit-msg")
	if err != nil {
		return stalledOnly(false, err)
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0, nil
}

func resolvePolicy(cfg *conf.Config, options selection.Options, spec string) ([]resolved, error) {
	var policy string
	var configured []string
	if cfg != nil {
		policy, configured = cfg.Policy, cfg.LegacyChecks
	}
	policy, names, err := options.Resolve(policy, configured, spec, checkCatalog(), lang.Ignored()...)
	if err == nil {
		names, err = repoChecks(cfg, policy, names)
	}
	if err != nil {
		return nil, err
	}
	if policy == selection.External {
		fmt.Fprintln(os.Stderr, "fitness: policy=external checks="+strings.Join(names, ","))
		return resolveList(names)
	}
	return prepareChecks(cfg, spec)
}
