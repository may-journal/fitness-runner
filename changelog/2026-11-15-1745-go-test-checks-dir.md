### 2026.11.15.1745

> A Go test that runs fitness finds the installed checks again, since the test runner keeps `FITNESS_CHECKS_DIR`.

- Fix: `toolchain.CleanEnv` keeps `FITNESS_CHECKS_DIR`, so tests that start fitness no longer print "no result from the check".
- Fix: `CleanEnv` still drops every other `FITNESS_` variable, so those tests inherit no scope.
- Test: requirement 0162.5 builds `go-test` on this engine and runs a Go test that starts fitness.
