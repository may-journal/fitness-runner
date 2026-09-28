package main

import (
	"reflect"
	"slices"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/conf"
)

func TestConfiguredNamesDefaultsWithoutConfig(t *testing.T) {
	names, fromConfig := configuredNames(nil)
	if fromConfig || !reflect.DeepEqual(names, defaultChecks) {
		t.Errorf("configuredNames(nil) = %v, %v; want the defaults", names, fromConfig)
	}
}

func TestConfiguredNamesChecksListOverrides(t *testing.T) {
	cfg := &conf.Config{Checks: []string{"cspell"}, EnableChecks: []string{"commit-attribution"}}
	names, fromConfig := configuredNames(cfg)
	if !fromConfig || !reflect.DeepEqual(names, []string{"cspell"}) {
		t.Errorf("configuredNames = %v, %v; want only the checks list", names, fromConfig)
	}
}

func TestConfiguredNamesEnableChecksAppends(t *testing.T) {
	cfg := &conf.Config{EnableChecks: []string{"commit-attribution"}}
	names, fromConfig := configuredNames(cfg)
	if fromConfig || len(names) != len(defaultChecks)+1 || names[len(names)-1] != "commit-attribution" {
		t.Errorf("configuredNames = %v; want the defaults plus commit-attribution", names)
	}
	if slices.Contains(defaultChecks, "commit-attribution") {
		t.Error("appending must not modify the default list")
	}
}

func TestDefaultChecksIncludeGoAndBuildChecks(t *testing.T) {
	for _, name := range []string{"go-vet", "go-test", "gofmt", "changelog-bullets", "build-output-untracked"} {
		if !slices.Contains(defaultChecks, name) {
			t.Errorf("default list is missing %s", name)
		}
	}
}
