package metadata

import (
	"net/url"
	"os"
	"sync"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// GitHubAPI is the address of GitHub's API: GITHUB_API_URL, which Actions
// and the release lookup use, else api.github.com for github.com, else
// GitHubHost itself, where GitHub Enterprise Server serves its API.
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

// UnreachableGitHub probes GitHub's API host, once per process however
// often it is asked, and returns the host when it did not answer, else
// empty. The pre-push gate and the runner share the one answer.
func UnreachableGitHub() string {
	return probeGitHub()
}
