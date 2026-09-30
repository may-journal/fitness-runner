package main

import (
	"os"
	"runtime/debug"
	"strings"
)

// runnerRepo is where the fitness code lives, for links to the build.
const runnerRepo = "https://github.com/may-journal/fitness-runner"

// buildInfo reads the running binary's build details; tests pin it.
var buildInfo = debug.ReadBuildInfo

// attribution is the verdict footer naming the fitness-runner code that
// judged the body and the Actions run that ran it.
func attribution() string {
	parts := []string{"Checked by fitness-runner " + codeLink()}
	if url := runURL(); url != "" {
		parts = append(parts, "[workflow run]("+url+")")
	}
	return "<sub>" + strings.Join(parts, " · ") + "</sub>"
}

// codeLink links the commit or tag the binary was built from.
func codeLink() string {
	info, ok := buildInfo()
	if !ok {
		return "(unknown build)"
	}
	if rev := vcsRevision(info); rev != "" {
		return commitLink(rev)
	}
	return versionLink(info.Main.Version)
}

// vcsRevision is the commit a from-source build records, or "".
func vcsRevision(info *debug.BuildInfo) string {
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return ""
}

// versionLink links a `go install`ed module version: a pseudo-version ends
// in the commit hash, and anything else is a tag.
func versionLink(v string) string {
	if v == "" || v == "(devel)" {
		return "(unknown build)"
	}
	parts := strings.Split(v, "-")
	if len(parts) >= 3 {
		return commitLink(parts[len(parts)-1])
	}
	return "[" + v + "](" + runnerRepo + "/releases/tag/" + v + ")"
}

func commitLink(rev string) string {
	short := rev
	if len(short) > 7 {
		short = short[:7]
	}
	return "[" + short + "](" + runnerRepo + "/commit/" + rev + ")"
}

// runURL is the Actions run executing this check, or "" outside Actions.
func runURL() string {
	server, repo, id := os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY"), os.Getenv("GITHUB_RUN_ID")
	if server == "" || repo == "" || id == "" {
		return ""
	}
	return server + "/" + repo + "/actions/runs/" + id
}
