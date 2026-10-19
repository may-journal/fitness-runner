package checkkit

import (
	"os"
	"runtime"
)

// shareCPUs gives a check its share of the cores when the runner starts it
// beside others. The runner runs up to one check per core, so each of the
// enabled checks gets the cores over the checks that run at once, and at
// least two. A check run alone, or with GOMAXPROCS set, keeps Go's default.
//
// Only this process changes: the environment is untouched, so the `go test`
// and `go vet` children a check starts still use every core.
func shareCPUs() {
	if os.Getenv("GOMAXPROCS") != "" {
		return
	}
	cpus := runtime.NumCPU()
	running := min(len(EnabledChecks()), cpus)
	if running > 1 {
		runtime.GOMAXPROCS(max(2, cpus/running))
	}
}
