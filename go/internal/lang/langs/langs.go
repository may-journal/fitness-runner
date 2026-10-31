// Package langs holds the language modules, one file each, and the registry
// that orders them.
//
//fitness:container cli
//fitness:component resolve Its registry lists every check a run resolves.
package langs

import "github.com/may-journal/fitness-runner/go/internal/lang"

// All is the registry: each module, in the order its check rows print.
var All = []lang.Module{Go, Swift, Node, Java}
