package swift

import (
	"regexp"

	"github.com/may-journal/fitness-runner/go/internal/runtimes"
)

// platformImport matches an import of a framework backed by a platform
// service, which the containers view must show as an external system.
var platformImport = regexp.MustCompile(`^\s*(?:@preconcurrency\s+)?import\s+(CloudKit|StoreKit|HealthKit|WeatherKit)\b`)

// platformServices records the first product line importing each
// platform framework.
func platformServices(f *runtimes.Finder, file string) {
	lines, ok := f.Lines(file)
	if !ok {
		return
	}
	for i, line := range lines {
		if match := platformImport.FindStringSubmatch(line); match != nil && !service(f, match[1]) {
			f.Services = append(f.Services, runtimes.Service{Name: match[1], At: runtimes.Location{Path: file, Line: i + 1}})
		}
	}
}

func service(f *runtimes.Finder, name string) bool {
	for _, s := range f.Services {
		if s.Name == name {
			return true
		}
	}
	return false
}
