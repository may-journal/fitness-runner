package checkkit

import (
	"context"
	"slices"
	"strings"
)

// MarkerPrefix starts every marker fitness writes into a comment.
const MarkerPrefix = "<!-- fitness:"

// ghEnvText is gh output blaming its environment, not the request: missing
// login, a rejected token, or an unreachable network.
var ghEnvText = []string{
	"gh auth login", "http 401", "bad credentials", "authentication required",
	"dial tcp", "no such host", "could not resolve host", "connection refused",
	"network is unreachable", "i/o timeout", "tls handshake timeout", "error connecting to",
}

// GHEnvironment reports whether a failed gh run blames its environment: no
// start, a signal, or an auth or network failure.
func GHEnvironment(o Outcome) bool {
	if o.EnvErr("gh") != nil {
		return true
	}
	text := strings.ToLower(o.Stdout + "\n" + o.Stderr)
	return slices.ContainsFunc(ghEnvText, func(s string) bool { return strings.Contains(text, s) })
}

// GHErr returns a gh run's error: nil on success, Internal for environment
// failures, and the run's own error otherwise.
func GHErr(ctx context.Context, o Outcome) error {
	switch {
	case o.Err == nil:
		return nil
	case ctx.Err() != nil:
		return Internal(ctx.Err())
	case o.SpawnErr != nil:
		return ToolMissing("gh")
	case GHEnvironment(o):
		return Internal(o.Err)
	}
	return o.Err
}
