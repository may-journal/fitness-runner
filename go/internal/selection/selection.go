// Package selection resolves explicit external policy without changing org defaults.
package selection

import (
	"fmt"
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
func (o Options) Resolve(policy string, configured []string, single string, known []string) (string, []string, error) {
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
	return resolvedNames(policy, names, known)
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

func resolvedNames(policy string, names, known []string) (string, []string, error) {
	if policy == External && names == nil {
		names = append([]string{}, known...)
	}
	return policy, names, validateNames(policy, names, known)
}
