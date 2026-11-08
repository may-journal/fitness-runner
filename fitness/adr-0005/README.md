# 0005 — Swift references come from a parser in the check

Fitness reads Swift references with its own Go parser.

https://github.com/may-journal/fitness-runner/issues/857

## Check

- Fails a `require` line in `go/go.mod`, since the parser takes no dependency.
- Fails a file in `go/internal/lang/swift/views` that imports cgo or starts a Swift toolchain.
- Fix: parse what the view needs in Go.
