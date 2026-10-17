package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/lang"
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
	if err := loadLocalChecks(cfg, root); err != nil {
		return nil, nil, err
	}
	checks, err := resolvePolicy(cfg, options, spec)
	return checks, passthrough, err
}

// commitChecks excludes branch metadata from a pre-commit run, including
// installer hooks. It also excludes semantic-commit when a commit-msg hook
// will judge the new message. Without one, as with the installer's hook,
// semantic-commit judges the branch's earlier commits. Both stay enabled,
// so docs may still name them.
func commitChecks(checks []resolved) []resolved {
	if os.Getenv("FITNESS_INTERNAL_HOOK") != "pre-commit" {
		return checks
	}
	messageHook := commitMsgHook()
	var local []resolved
	for _, check := range checks {
		if !skipsAtCommit(check.name, messageHook) {
			local = append(local, check)
		}
	}
	return local
}

// skipsAtCommit reports whether pre-commit leaves check out, given whether a
// commit-msg hook judges the new message.
func skipsAtCommit(check string, messageHook bool) bool {
	return check == "local-branch" || messageHook && check == "semantic-commit"
}

// commitMsgHook reports whether the repository has an executable commit-msg
// hook, in core.hooksPath or the git directory.
func commitMsgHook() bool {
	out, err := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-path", "hooks/commit-msg").Output()
	if err != nil {
		return false
	}
	info, err := os.Stat(strings.TrimSpace(string(out)))
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
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
		checks, err := resolveList(names)
		return withBudgets(cfg, checks), err
	}
	return prepareChecks(cfg, spec)
}
