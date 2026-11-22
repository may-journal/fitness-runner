package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

// pullRequest is the PR a step reads from PR, in the run's repository.
type pullRequest struct {
	number string
}

// stepPR is the PR the step's PR variable names.
func stepPR() pullRequest { return pullRequest{env("PR")} }

// path is the PR's REST path.
func (p pullRequest) path() string { return actions.RepoPath() + "/pulls/" + p.number }

// prView is the part of `gh pr view --json` the steps read.
type prView struct {
	Title  string `json:"title"`
	State  string `json:"state"`
	Draft  bool   `json:"isDraft"`
	Head   string `json:"headRefOid"`
	Base   string `json:"baseRefOid"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

// view reads fields of the PR.
func (p pullRequest) view(fields string) (prView, error) {
	var v prView
	return v, actions.GhJSON(&v, "pr", "view", p.number, "--json", fields)
}

// field reads one field of the PR as text.
func (p pullRequest) field(name string) (string, error) {
	return actions.Gh("pr", "view", p.number, "--json", name, "--jq", "."+name)
}

// gate says why the PR gets no approval, or "" to go on.
type gate func() (string, error)

// passes runs each gate in order and reports whether all passed. The first
// that refuses is noted as `PR #<n> <why>`.
func (p pullRequest) passes(gates ...gate) (bool, error) {
	for _, g := range gates {
		why, err := g()
		if err != nil {
			return false, err
		}
		if why != "" {
			return false, actions.Skip("PR #" + p.number + " " + why)
		}
	}
	return true, nil
}

// openAndReady refuses a closed or draft PR.
func openAndReady(v prView) gate {
	return func() (string, error) {
		if v.State == "OPEN" && !v.Draft {
			return "", nil
		}
		return "is not open and ready, so it gets no approval.", nil
	}
}

// changedFile is one file a PR changes, with its old name when renamed.
type changedFile struct {
	Filename string `json:"filename"`
	Previous string `json:"previous_filename"`
}

// onlyChanges refuses a PR that changes no file, or one beyond allowed; a
// rename counts both its names.
func (p pullRequest) onlyChanges(what string, allowed ...string) gate {
	return func() (string, error) {
		files, err := actions.GhPages[changedFile]("api", p.path()+"/files", "--paginate")
		var names []string
		for _, f := range files {
			names = append(names, f.Filename, f.Previous)
		}
		names = slices.DeleteFunc(names, func(n string) bool { return n == "" })
		others := slices.DeleteFunc(slices.Clone(names), func(n string) bool { return slices.Contains(allowed, n) })
		if err != nil || len(names) > 0 && len(others) == 0 {
			return "", err
		}
		return fmt.Sprintf("changes more than %s, so a person approves it: %s", what, strings.Join(others, ", ")), nil
	}
}
