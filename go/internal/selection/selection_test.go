package selection

import (
	"reflect"
	"testing"
)

var knownChecks = []string{"alpha", "beta", "new-check"}

func TestSelection(t *testing.T) {
	cases := []struct {
		name, policy, single   string
		configured, args, want []string
		mode                   string
	}{
		{name: "org default", mode: Org},
		{name: "external defaults to all", policy: External, want: knownChecks, mode: External},
		{name: "org ignores retired list", configured: []string{"old"}, mode: Org},
		{name: "config", policy: External, configured: []string{"beta", "alpha"}, want: []string{"beta", "alpha"}, mode: External},
		{name: "CLI list wins", policy: External, configured: []string{"old"}, args: []string{"--checks=alpha"}, want: []string{"alpha"}, mode: External},
		{name: "CLI mode wins", policy: "bad", args: []string{"--policy=external", "--checks=beta,alpha"}, want: []string{"beta", "alpha"}, mode: External},
		{name: "single wins", policy: External, configured: []string{"alpha"}, single: "beta", want: []string{"beta"}, mode: External},
		{name: "org override", policy: External, configured: []string{"alpha"}, args: []string{"--policy=org"}, mode: Org},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			options, _, err := Parse(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			policy, names, err := options.Resolve(tc.policy, tc.configured, tc.single, knownChecks)
			assertSelection(t, policy, names, err, tc.mode, tc.want)
		})
	}
}

func assertSelection(t *testing.T, policy string, names []string, err error, wantPolicy string, want []string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if policy != wantPolicy || !reflect.DeepEqual(names, want) {
		t.Fatalf("got %s %v; want %s %v", policy, names, wantPolicy, want)
	}
}

func TestInvalidCLI(t *testing.T) {
	cases := [][]string{{"--policy"}, {"--policy="}, {"--policy=other"}, {"--policy=org", "--policy=external"}, {"--checks"}, {"--checks=alpha", "--checks=beta"}}
	for _, args := range cases {
		if _, _, err := Parse(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestInvalidSelections(t *testing.T) {
	cases := []struct {
		policy, single   string
		configured, args []string
	}{
		{policy: "bad"},
		{args: []string{"--checks=alpha"}},
		{policy: External, configured: []string{}},
		{policy: External, configured: []string{""}},
		{policy: External, configured: []string{"missing"}},
		{policy: External, configured: []string{"alpha", "alpha"}},
		{policy: External, args: []string{"--checks="}},
		{policy: External, args: []string{"--checks=alpha,"}},
		{policy: External, args: []string{"--checks=alpha, beta"}},
		{policy: External, single: "alpha", args: []string{"--checks=beta"}},
	}
	for _, tc := range cases {
		options, _, _ := Parse(tc.args)
		if _, _, err := options.Resolve(tc.policy, tc.configured, tc.single, knownChecks); err == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
}

func TestRemainingArguments(t *testing.T) {
	_, args, err := Parse([]string{"--policy=external", "--all", "--checks=alpha", "--jobs=2"})
	if err != nil || !reflect.DeepEqual(args, []string{"--all", "--jobs=2"}) {
		t.Fatalf("remaining %v: %v", args, err)
	}
}

func TestOptionalCheckSelection(t *testing.T) {
	options, _, err := Parse([]string{"--policy=external", "--checks=go-test-coverage"})
	if err != nil {
		t.Fatal(err)
	}
	_, names, err := options.Resolve("", nil, "", []string{"go-test"}, "go-test-coverage")
	if err != nil || len(names) != 1 || names[0] != "go-test-coverage" {
		t.Fatalf("%v %v", names, err)
	}
}
func TestOptionalCheckDoesNotChangeDefault(t *testing.T) {
	options, _, err := Parse([]string{"--policy=external"})
	if err != nil {
		t.Fatal(err)
	}
	_, names, err := options.Resolve("", nil, "", []string{"go-test"}, "go-test-coverage")
	if err != nil || len(names) != 1 || names[0] != "go-test" {
		t.Fatalf("%v %v", names, err)
	}
}
