package gocoverage

import (
	"path/filepath"
)

func sourceFiles(module string, packages []Package) (map[string]string, error) {
	root, err := canonicalDirectory(module)
	if err != nil {
		return nil, err
	}
	sources := map[string]string{}
	for _, pkg := range packages {
		if err := packageSources(sources, root, pkg); err != nil {
			return nil, err
		}
	}
	return sources, nil
}
func packageSources(sources map[string]string, root string, pkg Package) error {
	directory, err := filepath.EvalSymlinks(pkg.Dir)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, directory)
	if err != nil {
		return err
	}
	for _, file := range append(pkg.GoFiles, pkg.CgoFiles...) {
		sources[pkg.ImportPath+"/"+file] = filepath.ToSlash(filepath.Join(relative, file))
	}
	return nil
}

func canonicalDirectory(directory string) (string, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}
