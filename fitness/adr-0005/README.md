# ADR 0005 check

Keeps [ADR 0005](../../docs/architecture/adr/0005-swift-references-from-a-go-parser.md): fitness reads Swift references with its own Go parser.

- Fails a `require` line in `go/go.mod`, since the parser takes no dependency.
- Fails a file in `go/internal/lang/swift/views` that imports cgo or starts a Swift toolchain.
- Fix: parse what the view needs in Go.
