package runtimes

import (
	"path"
	"strings"
)

// devRun collects what the manifests of a repository run and publish
// while DevScripts reads them.
type devRun struct {
	tracked, dev, start map[string]bool
	published           []string
}

// DevScripts lists tracked files that are tools by how the repo runs them,
// not files the start script runs or the package publishes.
func DevScripts(root string, files []string, readers []Reader) map[string]bool {
	r := newDevRun(files)
	for _, file := range files {
		for _, reader := range readers {
			if scripts := reader.Scripts[path.Base(file)]; scripts != nil {
				r.add(scripts(root, file))
			}
		}
	}
	return r.tools()
}

// newDevRun starts with every file under the top-level scripts folder.
func newDevRun(files []string) *devRun {
	r := &devRun{tracked: map[string]bool{}, dev: map[string]bool{}, start: map[string]bool{}}
	for _, file := range files {
		r.tracked[file] = true
		r.dev[file] = strings.HasPrefix(file, "scripts/")
	}
	return r
}

// tools lists the dev files the start script never runs and the package
// never publishes.
func (r *devRun) tools() map[string]bool {
	tools := map[string]bool{}
	for file, dev := range r.dev {
		tools[file] = dev && !r.start[file] && !published(file, r.published)
	}
	return tools
}

// add keeps one manifest's runs: the tracked files its start script and
// its other scripts run, and its published entries.
func (r *devRun) add(runs Runs) {
	r.mark(runs.Start, r.start)
	r.mark(runs.Dev, r.dev)
	r.published = append(r.published, runs.Published...)
}

// mark adds each tracked path to into.
func (r *devRun) mark(paths []string, into map[string]bool) {
	for _, p := range paths {
		if r.tracked[p] {
			into[p] = true
		}
	}
}

// published reports whether an entry names the file, directly, by folder, or
// by an exports pattern up to its star.
func published(file string, entries []string) bool {
	for _, entry := range entries {
		if star := strings.Index(entry, "*"); star >= 0 {
			entry = path.Dir(entry[:star] + "x")
		}
		if file == entry || strings.HasPrefix(file, entry+"/") {
			return true
		}
	}
	return false
}
