// Package selection resolves explicit external policy without changing org defaults.
package selection

import (
	"fmt"
	"slices"
	"strings"
)

const External = "external"
const Org = "org"

type Options struct {
	Policy               string
	Checks               []string
	policySet, checksSet bool
}

// Parse removes policy options before the legacy single-check parser runs.
func Parse(args []string) (Options, []string, error) {
	var options Options
	var remaining []string
	for _, arg := range args {
		handled, err := options.consume(arg)
		if err != nil {
			return options, nil, err
		}
		if !handled {
			remaining = append(remaining, arg)
		}
	}
	return options, remaining, nil
}

func (o *Options) consume(arg string) (bool, error) {
	key, value, found := strings.Cut(arg, "=")
	switch key {
	case "--policy":
		return true, o.setPolicy(value, found)
	case "--checks":
		return true, o.setChecks(value, found)
	default:
		return false, nil
	}
}

func (o *Options) setPolicy(value string, found bool) error {
	if !found || o.policySet {
		return fmt.Errorf("use --policy=org or --policy=external once")
	}
	o.Policy, o.policySet = value, true
	return validPolicy(value)
}

func (o *Options) setChecks(value string, found bool) error {
	if !found || o.checksSet {
		return fmt.Errorf("use --checks=name,name once")
	}
	o.Checks, o.checksSet = strings.Split(value, ","), true
	return nil
}

func validPolicy(policy string) error {
	if policy != Org && policy != External {
		return fmt.Errorf("unknown policy %q; use org or external", policy)
	}
	return nil
}

// Resolve returns nil names for the existing org selection behavior.
func (o Options) Resolve(policy string, configured []string, single string, known []string, optional ...string) (string, []string, error) {
	if o.policySet {
		policy = o.Policy
	}
	if policy == "" {
		policy = Org
	}
	if err := validPolicy(policy); err != nil {
		return "", nil, err
	}
	names, err := o.names(policy, configured, single)
	if err != nil {
		return policy, nil, err
	}
	return resolvedNames(policy, names, known, optional...)
}

func (o Options) names(policy string, configured []string, single string) ([]string, error) {
	if policy == Org {
		return nil, o.orgChecks()
	}
	if o.checksSet {
		return o.explicitNames(single)
	}
	if single != "" {
		return []string{single}, nil
	}
	return configured, nil
}

func (o Options) orgChecks() error {
	if o.checksSet {
		return fmt.Errorf("--checks requires --policy=external or policy external in config")
	}
	return nil
}

func (o Options) explicitNames(single string) ([]string, error) {
	if single != "" {
		return nil, fmt.Errorf("choose --checks or a single check, not both")
	}
	return o.Checks, nil
}

func validateNames(policy string, names, known []string) error {
	if policy == Org {
		return nil
	}
	if len(names) == 0 {
		return fmt.Errorf("external policy requires a nonempty checks list")
	}
	available := make(map[string]bool, len(known))
	for _, name := range known {
		available[name] = true
	}
	return validateList(names, available)
}

func validateList(names []string, known map[string]bool) error {
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if !known[name] {
			return fmt.Errorf("unknown or empty check %q", name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate check %q", name)
		}
		seen[name] = true
	}
	return nil
}

func resolvedNames(policy string, names, known []string, optional ...string) (string, []string, error) {
	if policy == External && names == nil {
		names = append([]string{}, known...)
	}
	return policy, names, validateNames(policy, names, append(append([]string{}, known...), optional...))
}

// AllowReplacements applies ADR 0008 to local checks that share a built-in
// check's name: org policy is the org's shared bar, so it refuses them;
// external policy lets a repo own its bar and allows them.
func AllowReplacements(policy string, replaced []string) error {
	if len(replaced) == 0 || policy == External {
		return nil
	}
	return fmt.Errorf("local check %s replaces a built-in check, and org policy runs the built-in checks as the shared bar (ADR 0008); rename it, or set \"policy\": \"external\"",
		strings.Join(replaced, ", "))
}

// Disable applies ADR 0008 to disabledChecks: org policy refuses it, as it
// would lower the shared bar; external policy drops each named check from
// names. Every disabled name must be a known check, listed once, and at
// least one check must remain.
func Disable(policy string, names, disabled, known []string) ([]string, error) {
	if len(disabled) == 0 {
		return names, nil
	}
	if policy != External {
		return nil, fmt.Errorf("disabledChecks turns off %s, and org policy runs every check as the shared bar (ADR 0008); fix the findings, or set \"policy\": \"external\"",
			strings.Join(disabled, ", "))
	}
	available := make(map[string]bool, len(known))
	for _, name := range known {
		available[name] = true
	}
	if err := validateList(disabled, available); err != nil {
		return nil, fmt.Errorf("disabledChecks: %w", err)
	}
	return remaining(names, disabled)
}

// remaining is names without the disabled ones, failing when none is left.
func remaining(names, disabled []string) ([]string, error) {
	var kept []string
	for _, name := range names {
		if !slices.Contains(disabled, name) {
			kept = append(kept, name)
		}
	}
	if len(kept) == 0 {
		return nil, fmt.Errorf("disabledChecks turns off every selected check; run at least one")
	}
	return kept, nil
}
