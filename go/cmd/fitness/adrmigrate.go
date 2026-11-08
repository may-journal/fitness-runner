package main

import (
	"flag"
	"io"
	"os"

	"github.com/may-journal/fitness-runner/go/internal/adr"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// runADRMigrate runs `fitness adr-migrate`: it moves each ADR file of the
// repo in the working folder into an `ADR` issue, as adr-as-issue asks.
func runADRMigrate(args []string) int {
	m, err := adrMigration(args)
	if err == nil {
		var files []string
		if files, err = walkfs.AllFiles(m.Root); err == nil {
			err = m.Run(files)
		}
	}
	if err != nil {
		report.Error("fitness adr-migrate", err)
		return 1
	}
	return 0
}

// adrMigration reads the command's flags into a Migration, asking gh for
// the repo when --repo names none.
func adrMigration(args []string) (adr.Migration, error) {
	fs := flag.NewFlagSet("adr-migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dry := fs.Bool("dry-run", false, "print the plan and change nothing")
	repo := fs.String("repo", "", "owner/name the issues live in")
	branch := fs.String("branch", "main", "branch issue bodies link other files on")
	if err := fs.Parse(args); err != nil {
		return adr.Migration{}, err
	}
	root, err := os.Getwd()
	if err == nil && *repo == "" {
		*repo, err = adr.CurrentRepo()
	}
	return adr.Migration{Root: root, Branch: *branch, GitHub: adr.GitHub{Repo: *repo}, DryRun: *dry, Out: os.Stdout}, err
}
