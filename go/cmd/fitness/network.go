package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	issueLinkOnceCheck "github.com/may-journal/fitness-runner/go/cmd/fitness-check-issue-link-once"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/lang/node"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
)

// networkEnv probes GitHub's API host only when a network check will run, and
// returns env entries saying it did not answer.
//
// GOPROXY=off makes a Go check missing a module fail at once, and one probe
// stands for npm too, since one proxy blocks both.
func networkEnv(root string, checks []resolved, msg string) []string {
	if !slices.ContainsFunc(checks, func(c resolved) bool { return readsNetwork(root, c, msg) }) {
		return nil
	}
	host := metadata.UnreachableGitHub()
	if host == "" {
		return nil
	}
	return []string{checkkit.UnreachableEnv(host), "GOPROXY=off"}
}

// readsNetwork reports a GitHub metadata check outside pre-commit,
// dependency-currency with a root package.json, or issue-link-once on a link.
func readsNetwork(root string, check resolved, msg string) bool {
	switch check.name {
	case "dependency-currency":
		_, err := os.Stat(filepath.Join(root, node.Manifest))
		return err == nil
	case "issue-link-once":
		return len(issueLinkOnceCheck.MessageLinks(msg, gitx.OriginSlug(root))) > 0
	}
	return check.desc.Scope == checkkit.ScopeGitHub && !atCommit(check.name)
}

// atCommit reports whether this run is pre-commit's and leaves check out.
func atCommit(check string) bool {
	return os.Getenv("FITNESS_INTERNAL_HOOK") == "pre-commit" && skipsAtCommit(check, false)
}

// gateOnline reports whether the pre-push gate reads its Plans: it is on,
// and GitHub's API host answered the probe. When the host did not, the gate
// prints one info line naming the host and the proxy and lets the push go.
func gateOnline(gated bool) bool {
	if !gated {
		return false
	}
	host := metadata.UnreachableGitHub()
	if host == "" {
		return true
	}
	fmt.Fprintln(os.Stderr, "pre-push: info: "+checkkit.OfflineNote(host, "the Plan gate"))
	return false
}
