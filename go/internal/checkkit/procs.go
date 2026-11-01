package checkkit

import (
	"os"
	"runtime"
)

// shareCPUs gives a check its share of the cores, at least two, when run beside
// others, changing only this process.
func shareCPUs() {
	if os.Getenv("GOMAXPROCS") != "" {
		return
	}
	cpus := runtime.GOMAXPROCS(0)
	running := min(len(EnabledChecks()), cpus)
	if running > 1 {
		runtime.GOMAXPROCS(max(2, cpus/running))
	}
}
