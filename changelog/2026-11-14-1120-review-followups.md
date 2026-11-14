### 2026.11.14.1120

- Fix: a build holding `local-branch` without the engine reports that it could not judge, with exit 3, instead of a failure.
- Test: requirement 0160 finds engine packages in any folder and reads the Go version from `go.mod`.
- Test: requirement 0160 ignores `GOWORK` and names the offending check and package.
- Test: requirement 0112 accepts only a package under `go/pkg` or `go/internal`.
