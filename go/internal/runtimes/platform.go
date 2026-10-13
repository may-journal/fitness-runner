package runtimes

import "regexp"

// platformImport matches an import of a framework backed by a platform
// service, which the containers view must show as an external system.
var platformImport = regexp.MustCompile(`^\s*(?:@preconcurrency\s+)?import\s+(CloudKit|StoreKit|HealthKit|WeatherKit)\b`)

// platformServices records the first product line importing each
// platform framework.
func (f *finder) platformServices(file string) {
	lines, ok := f.lines(file)
	if !ok {
		return
	}
	for i, line := range lines {
		if match := platformImport.FindStringSubmatch(line); match != nil && !f.service(match[1]) {
			f.Services = append(f.Services, Service{Name: match[1], At: Location{file, i + 1}})
		}
	}
}

func (f *finder) service(name string) bool {
	for _, s := range f.Services {
		if s.Name == name {
			return true
		}
	}
	return false
}
