package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/selection"
	"github.com/may-journal/fitness-runner/go/pkg/conf"
	"github.com/may-journal/fitness-runner/go/pkg/lang"
	"github.com/may-journal/fitness-runner/go/pkg/lang/langs"
)

// configView is what `fitness config` shows: the repo's resolved settings
// and, for each check, whether the suite runs it and why.
type configView struct {
	// File is the config file the repo has, or "" when it has none and
	// every key takes its default.
	File string `json:"file"`
	// Policy is the policy a run uses: org or external.
	Policy string `json:"policy"`
	// PolicyFrom says what set Policy: --policy, the config file, or the
	// default.
	PolicyFrom string `json:"policyFrom"`
	// Settings lists each key with its value, where that came from, and
	// what it does.
	Settings []conf.Setting `json:"settings"`
	// Checks lists every check the suite knows, in run order.
	Checks []checkState `json:"checks"`
	// Error says why a run with these settings would stop before any check
	// runs, or is empty.
	Error string `json:"error,omitempty"`
}

// checkState is one check's place in the suite.
type checkState struct {
	// Name is the check's name.
	Name string `json:"name"`
	// Enabled says whether the full suite runs the check.
	Enabled bool `json:"enabled"`
	// Reason says why the check is enabled or skipped.
	Reason string `json:"reason"`
	// Origin says where the check's program comes from: built-in, local,
	// or a local check that replaces a built-in one.
	Origin string `json:"origin"`
	// Language names the language module that claims the check, if any.
	Language string `json:"language,omitempty"`
}

// runConfig handles `fitness config [--json] [--policy=..] [--checks=..]`,
// which shows the resolved settings and each check's state, and `fitness
// config schema`, which prints a JSON Schema for the config file. It exits
// 1 when the settings would stop a run, and 2 on a bad argument.
func runConfig(args []string) int {
	if len(args) > 0 && args[0] == "schema" {
		return printSchema(args[1:])
	}
	view, asJSON, code := configFor(args)
	if view == nil {
		return code
	}
	printConfig(os.Stdout, *view, asJSON)
	return code
}

// configFor reads the arguments and resolves the view, or reports why it
// cannot and returns no view. code is the exit code: 2 for a bad argument,
// 1 when settings would stop a run, else 0.
func configFor(args []string) (view *configView, asJSON bool, code int) {
	asJSON, options, err := configArgs(args)
	if err != nil {
		report.Error("fitness config", err)
		return nil, false, 2
	}
	resolved, err := resolveConfig(options)
	if err != nil {
		report.Error("fitness config", err)
		return nil, false, 1
	}
	return &resolved, asJSON, stopCode(resolved)
}

// stopCode is 1 when the view's settings would stop a run, else 0.
func stopCode(view configView) int {
	if view.Error != "" {
		return 1
	}
	return 0
}

// configArgs reads `--json` and the selection flags; anything else fails.
func configArgs(args []string) (asJSON bool, options selection.Options, err error) {
	options, rest, err := selection.Parse(args)
	if err != nil {
		return false, options, err
	}
	for _, a := range rest {
		if a != "--json" {
			return false, options, fmt.Errorf("unknown argument %q; use fitness config [--json] [--policy=..] [--checks=..], or fitness config schema", a)
		}
		asJSON = true
	}
	return asJSON, options, nil
}

// printSchema prints the config file's JSON Schema, built from the code.
func printSchema(args []string) int {
	if len(args) > 0 {
		report.Error("fitness config schema", errors.New("takes no arguments"))
		return 2
	}
	declared, err := repoDeclaredKeys()
	var schema map[string]any
	if err == nil {
		schema, err = conf.Schema(declared...)
	}
	if err == nil {
		err = printIndented(os.Stdout, schema)
	}
	if err != nil {
		report.Error("fitness config schema", err)
		return 1
	}
	return 0
}

// resolveConfig reads the repo's settings and works out each check's
// state as a full run with options would.
func resolveConfig(options selection.Options) (configView, error) {
	root, view, cfg, err := readSettings()
	if err != nil {
		return view, err
	}
	view.Policy, view.PolicyFrom = policyOf(cfg, options)
	if err := loadLocalChecks(cfg, root); err != nil {
		view.Error = err.Error()
		view.Settings, err = conf.Settings(root)
		return view, err
	}
	enabled, err := suiteChecks(cfg, options)
	declared, declErr := declaredKeys(enabled)
	if err = cmp.Or(err, declErr); err != nil {
		view.Error = err.Error()
	}
	if view.Settings, err = conf.Settings(root, declared...); err != nil {
		return view, err
	}
	view.Checks = checkStates(root, cfg, options, enabled)
	return view, nil
}

// readSettings reads the repo's config file from the working directory:
// the config itself, nil without a file, and a view naming the file.
func readSettings() (root string, view configView, cfg *conf.Config, err error) {
	root, err = os.Getwd()
	if err == nil {
		cfg, err = conf.Load(root)
	}
	if err != nil {
		return root, view, nil, err
	}
	if cfg != nil {
		view.File = conf.FileName
	}
	return root, view, cfg, nil
}

// repoDeclaredKeys lists every check's declared keys, opt-in checks' too, so
// a repo may set them before naming the check. It skips the local checks
// when the config file fails to load, so the schema still helps fix it.
func repoDeclaredKeys() ([]conf.Key, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if cfg, err := conf.Load(root); err == nil {
		if err := loadLocalChecks(cfg, root); err != nil {
			return nil, err
		}
	}
	return declaredKeys(append(checkCatalog(), lang.Ignored()...))
}

// declaredKeys lists the config keys that the named checks declare in their
// `--describe`. A local check that fails to build is an error, not a silent
// gap.
func declaredKeys(names []string) ([]conf.Key, error) {
	found, missing := installedChecks(names)
	checks, err := resolveList(found)
	if err != nil {
		return nil, err
	}
	keys, err := keysOfChecks(checks)
	return keys, errors.Join(err, brokenLocal(missing))
}

// brokenLocal joins the build errors of the local checks among missing,
// whose keys no `--describe` could report.
func brokenLocal(missing map[string]string) error {
	var errs []error
	for _, name := range slices.Sorted(maps.Keys(missing)) {
		if _, local := localBins[name]; local {
			errs = append(errs, fmt.Errorf("local check %s cannot declare its config keys: %s", name, missing[name]))
		}
	}
	return errors.Join(errs...)
}

// keysOfChecks lists each check's declared keys in check order, failing on
// the first key a check may not claim.
func keysOfChecks(checks []resolved) ([]conf.Key, error) {
	owner := map[string]string{}
	var keys []conf.Key
	for _, c := range checks {
		for _, k := range c.desc.Config {
			if err := claimKey(owner, c.name, k); err != nil {
				return nil, err
			}
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// claimKey records check as key k's owner. A malformed key fails, as does
// one another check declared, or an engine key whose check tag names
// another reader.
func claimKey(owner map[string]string, check string, k conf.Key) error {
	if err := validKey(k, ""); err != nil {
		return fmt.Errorf("check %s declares a bad config key: %w", check, err)
	}
	if other, taken := owner[k.Path]; taken {
		return fmt.Errorf("checks %s and %s both declare the %q config key; one check owns each key", other, check, k.Path)
	}
	if reader, engine := conf.KeyOwner(k.Path); engine && reader != check {
		return fmt.Errorf("check %s declares the engine's %q config key, which only %s may declare", check, k.Path, cmp.Or(reader, "the engine"))
	}
	owner[k.Path] = check
	return nil
}

// keyTypes are the JSON types a declared key may take.
var keyTypes = []string{"string", "integer", "number", "boolean", "array", "object"}

// validKey reports a declared key whose path is not one name under prefix
// or whose type is not a JSON type. It checks the key's own keys the same way.
func validKey(k conf.Key, prefix string) error {
	if err := keyShape(k, prefix); err != nil {
		return err
	}
	for _, sub := range k.Keys {
		if err := validKey(sub, k.Path+"."); err != nil {
			return err
		}
	}
	return nil
}

// keyShape reports key k alone breaking validKey's rules, not its own keys.
func keyShape(k conf.Key, prefix string) error {
	if name, under := strings.CutPrefix(k.Path, prefix); !under || !oneName(name) {
		return fmt.Errorf("%q must be one name under %q", k.Path, strings.TrimSuffix(prefix, "."))
	}
	if !slices.Contains(keyTypes, k.Type) {
		return fmt.Errorf("%q has type %q, not one of %s", k.Path, k.Type, strings.Join(keyTypes, ", "))
	}
	return nil
}

// oneName reports whether name is a single key name, not empty or dotted.
func oneName(name string) bool {
	return name != "" && !strings.Contains(name, ".")
}

// policyOf is the policy a run uses and what set it.
func policyOf(cfg *conf.Config, options selection.Options) (policy, from string) {
	switch {
	case options.Policy != "":
		return options.Policy, "--policy"
	case cfg != nil && cfg.Policy != "":
		return cfg.Policy, conf.FileName
	}
	return conf.Defaults.Policy, "default"
}

// suiteChecks is the checks a full run with options runs, as
// selectedChecks resolves them, or why it would stop.
func suiteChecks(cfg *conf.Config, options selection.Options) ([]string, error) {
	var policy string
	var configured []string
	if cfg != nil {
		policy, configured = cfg.Policy, cfg.LegacyChecks
	}
	policy, names, err := options.Resolve(policy, configured, "", checkCatalog(), lang.Ignored()...)
	if err != nil {
		return nil, err
	}
	if names == nil {
		names = checkCatalog()
	}
	return repoChecks(cfg, policy, names)
}

// checkStates lists every known check, the ignored ones last, with whether
// the suite runs it and why.
func checkStates(root string, cfg *conf.Config, options selection.Options, enabled []string) []checkState {
	claims := languages(root)
	names := append(checkCatalog(), lang.Ignored()...)
	states := make([]checkState, len(names))
	for i, name := range names {
		on := slices.Contains(enabled, name)
		states[i] = checkState{Name: name, Enabled: on, Reason: reason(cfg, options, name, on), Origin: origin(name)}
		if m, ok := claims[name]; ok {
			states[i].Language = m.name
			states[i].Reason += m.note(on)
		}
	}
	return states
}

// claim is the language module that claims a check, and whether this repo
// uses its language.
type claim struct {
	name    string
	present bool
}

// note adds to an enabled check's reason when its language is absent.
func (c claim) note(enabled bool) string {
	if !enabled || c.present {
		return ""
	}
	return "; no " + c.name + " project here, so it passes with 0 files"
}

// languages maps each check a language module claims to that module, with
// whether the repo at root uses the language.
func languages(root string) map[string]claim {
	claims := map[string]claim{}
	for _, m := range langs.All {
		present := m.Detect(root)
		for _, check := range m.Checks {
			claims[check] = claim{m.Name, present}
		}
	}
	return claims
}

// reason says why the suite runs a check or skips it.
func reason(cfg *conf.Config, options selection.Options, name string, enabled bool) string {
	if why := lang.IgnoredWhy(name); why != "" && !enabled {
		return why
	}
	if policy, _ := policyOf(cfg, options); policy != selection.External {
		return "org policy runs every check"
	}
	return externalReason(cfg, options, name, enabled)
}

// externalReason says why an external policy runs a check or skips it.
func externalReason(cfg *conf.Config, options selection.Options, name string, enabled bool) string {
	if cfg != nil && slices.Contains(cfg.DisabledChecks, name) {
		return "disabledChecks in " + conf.FileName + " turns it off"
	}
	return listReason(cfg, options, enabled)
}

// listReason says how an external policy's check list picks a check.
func listReason(cfg *conf.Config, options selection.Options, enabled bool) string {
	list := ""
	switch {
	case options.Checks != nil:
		list = "--checks"
	case cfg != nil && len(cfg.LegacyChecks) > 0:
		list = "the checks list in " + conf.FileName
	default:
		return "external policy runs every check"
	}
	if enabled {
		return list + " names it"
	}
	return list + " leaves it out"
}

// origin says where a check's program comes from.
func origin(name string) string {
	if _, local := localBins[name]; !local {
		return "built-in"
	}
	if builtIn(name) {
		return "local, replacing the built-in check"
	}
	return "local"
}

// printConfig writes the view as indented JSON, or as text for people.
func printConfig(w io.Writer, view configView, asJSON bool) {
	if asJSON {
		_ = printIndented(w, view)
		return
	}
	file := view.File
	if file == "" {
		file = "none, so every key takes its default"
	}
	fmt.Fprintf(w, "Config file: %s\nPolicy: %s, from %s\n\nSettings:\n", file, view.Policy, view.PolicyFrom)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, s := range view.Settings {
		value, _ := json.Marshal(s.Value)
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", s.Key, value, s.Source, s.Description)
	}
	_ = tw.Flush()
	printChecks(w, view)
}

// printChecks writes each check's state, then why a run would stop.
func printChecks(w io.Writer, view configView) {
	fmt.Fprintln(w, "\nChecks:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, c := range view.Checks {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", c.Name, map[bool]string{true: "enabled", false: "skipped"}[c.Enabled], c.Reason)
	}
	_ = tw.Flush()
	if view.Error != "" {
		fmt.Fprintln(w, "\nA run stops before any check: "+strings.TrimSpace(view.Error))
	}
}

// printIndented writes v as indented JSON.
func printIndented(w io.Writer, v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(out))
	return err
}
