package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
	"github.com/may-journal/fitness-runner/go/pkg/gitx"
	"github.com/may-journal/fitness-runner/go/pkg/lang/node"
)

// networkEnv probes GitHub's API host when a check that reads the network
// will run. It returns env entries (checkkit.NetworkEnv, GOPROXY=off) when the
// host did not answer, and a run without such a check makes no connection.
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

// readsNetwork reports whether check reads the network in this run, such as
// live GitHub metadata or a package.json lookup.
func readsNetwork(root string, check resolved, msg string) bool {
	switch check.name {
	case "dependency-currency":
		_, err := os.Stat(filepath.Join(root, node.Manifest))
		return err == nil
	case "issue-link-once":
		return len(gitx.MessageLinks(msg, gitx.OriginSlug(root))) > 0
	}
	return readsOnline(check.desc) && !atCommit(check.name)
}

// readsOnline reports whether desc reads GitHub or requires the network.
func readsOnline(desc checkkit.Describe) bool {
	return onlineRequirement(desc) != ""
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
