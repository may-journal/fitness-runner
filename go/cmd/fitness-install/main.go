// Command fitness-install fetches verified Fitness binaries and runs them.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
)

// Release publishing embeds the exact version and platform bundle hashes.
var version = "0.20261004.900"
var bundleHashes string

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	options, err := distribution.ParseOptions(args, version, os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		return failure(err)
	}
	installer, err := distribution.New(version, bundleHashes, os.Stderr)
	if err != nil {
		return failure(err)
	}
	return execute(installer, options)
}

func execute(installer *distribution.Installer, options distribution.Options) int {
	bin, err := installer.Install(context.Background(), options.Version)
	if err != nil {
		return failure(err)
	}
	if options.InstallOnly {
		fmt.Println(bin)
		return 0
	}
	runner := filepath.Join(bin, "fitness")
	return failure(syscall.Exec(runner, append([]string{runner}, options.Args...), os.Environ()))
}

func failure(err error) int { fmt.Fprintln(os.Stderr, "fitness:", err); return 1 }
