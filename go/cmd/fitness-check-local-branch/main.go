// Command fitness-check-local-branch runs PR and Plan judgments locally.
package main

import (
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/metadata"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "local-branch", TimeoutMs: 35000},
		Run:      metadata.LocalBranch,
	})
}
