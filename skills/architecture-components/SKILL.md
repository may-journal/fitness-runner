---
name: architecture-components
description: Fix a failing architecture-components check on the components view. Use when a source has no owner or a reference has no drawn edge.
---

# Give every source an owner and draw its references

Each product source belongs to one component in the [components view](../../docs/architecture/03-components.md), and every reference between components is a drawn edge. [The check README](../../go/cmd/fitness-check-architecture-components/README.md) defines owners, references, and runtimes.

## Read the failure

- `go/internal/store/store.go:1: no component in the components view names this source`: name its path in a component's callout row or diagram text. A trailing `*` covers sibling folders.
- `go/a.go:1: components A at 03-components.md:20 and B at 03-components.md:21 both name this source`: keep the claim on one component.
- `Parser imports Store, owned by Store, but 03-components.md draws no Parser -> Store edge`: draw the edge, or remove the reference from the code. The real line names the reference kind and location.
- `evidence <link> names nothing <component> owns`: link a source the first component owns whose lines import or name the second component.
- `evidence <link> is owned by <owner>, not <component>`: link a source the edge's first component owns.
- `<kind> <name> reaches <unit> here, but <unit> is under container <id> at <path>:<line>, not <runtime>; ...`: move the shared component outside every container boundary.

## Check it first

Run `fitness architecture-components` from the repo root. Exit 0 means every source has one owner and every reference is drawn.
