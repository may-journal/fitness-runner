// Package usage warns about flags a subcommand was given but does not
// read, so a flag never silently does nothing.
package usage

import (
	"flag"
	"fmt"
	"io"
	"slices"
)

// WarnUnhandled writes a warning to w for each flag set on the command line
// that command does not list in handled.
func WarnUnhandled(w io.Writer, command string, flags *flag.FlagSet, handled []string) {
	flags.Visit(func(f *flag.Flag) {
		if !slices.Contains(handled, f.Name) {
			fmt.Fprintf(w, "%s: warning: --%s not handled, ignoring it\n", command, f.Name)
		}
	})
}
