package metadata

import (
	"net/url"
	"os"
	"sync"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// GitHubAPI is the address of GitHub's API: GITHUB_API_URL, else api.github.com
// for github.com, else GitHubHost itself, as GitHub Enterprise Server serves it.
func GitHubAPI() *url.URL {
	if u, err := url.Parse(os.Getenv("GITHUB_API_URL")); err == nil && u.Host != "" {
		return u
	}
	host := GitHubHost()
	if host == "github.com" {
		host = "api.github.com"
	}
	return &url.URL{Scheme: "https", Host: host}
}

// probeGitHub probes GitHubAPI once per process.
var probeGitHub = sync.OnceValue(func() string {
	api := GitHubAPI()
	if checkkit.Probe(api) != nil {
		return api.Host
	}
	return ""
})

// UnreachableGitHub probes GitHub's API host once per process and returns it
// when silent, sharing one answer with the gate and runner.
func UnreachableGitHub() string {
	return probeGitHub()
}
