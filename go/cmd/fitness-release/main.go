// Command fitness-release builds, publishes, and verifies release assets.
// cspell:ignore releasemeta
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/may-journal/fitness-runner/go/internal/release"
	"github.com/may-journal/fitness-runner/go/internal/releasemeta"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

func main() { os.Exit(command(os.Args[1:])) }

func command(args []string) int {
	operation := "fitness-release"
	if len(args) > 0 {
		operation += " " + args[0]
	}
	if err := run(args); err != nil {
		reportFailure(operation, err)
		return 1
	}
	report.Outcome(operation, "Completed successfully.")
	return 0
}

func reportFailure(operation string, err error) {
	if report.Enabled() {
		report.Error(operation, err)
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("use bundle-hashes, assemble, smoke, verify-download, verify-tag, promote, release-pr, version, or tag")
	}
	flags := flag.NewFlagSet("fitness-release", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	config, err := release.New(*root)
	if err != nil {
		return err
	}
	return dispatch(args[0], config, *root)
}

func dispatch(command string, config release.Config, root string) error {
	handlers := map[string]func() error{
		"bundle-hashes": func() error {
			hashes, err := config.BundleHashes()
			if err == nil {
				fmt.Println(hashes)
			}
			return err
		},
		"assemble":        config.Assemble,
		"promote":         func() error { return config.Promote(context.Background()) },
		"release-pr":      func() error { return updateReleasePR(root) },
		"smoke":           func() error { return config.Smoke(context.Background(), false) },
		"verify-download": func() error { return config.Smoke(context.Background(), true) },
		"verify-tag":      func() error { return config.VerifyTag(os.Getenv("GITHUB_REF_NAME")) },
		"version": func() error {
			version, err := config.Version()
			if err == nil {
				fmt.Println(version)
			}
			return err
		},
		"tag": func() error {
			version, err := config.Version()
			if err == nil {
				fmt.Println("v" + version)
			}
			return err
		},
	}
	handler, ok := handlers[command]
	if !ok {
		return fmt.Errorf("unknown release command %q", command)
	}
	return handler()
}

func updateReleasePR(root string) error {
	var value struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal([]byte(os.Getenv("RELEASE_PR")), &value); err != nil {
		return err
	}
	client := releasemeta.Client{Root: root, Repo: os.Getenv("GITHUB_REPOSITORY")}
	return client.UpdateReleasePR(context.Background(), value.Number)
}
