---
name: architecture-contract
description: Fix a failing architecture-contract check on the architecture views. Use when a view is missing, malformed, or names a path or requirement that does not exist.
---

# Keep the five architecture views well formed

A repo with product code has all five views under `docs/architecture/` or `architecture/`. Each view's diagram resolves to real IDs, paths, and requirements. [The check README](../../go/cmd/fitness-check-architecture-contract/README.md) lists the model. Start each view from its [template](../../docs/architecture/templates/01-system-context.md); the other four sit beside it.

## Read the failure

- `docs/architecture/ has no views, but product code such as go/main.go needs them; add the five views ...`: copy the five templates and fill them in.
- `unresolved source path go/old.go`: the claimed path is gone. Update it to the real path.
- `unresolved evidence go/a.go#L3-L9`: fix the link or line range.
- `unresolved requirement 0099 in docs/requirements`: name an existing requirement doc or acceptance.
- `missing architecture view 03-components.md`: add the view from its template.
- `view has no diagram outside its legends`: draw the view's real diagram.
- `container X needs an enclosing boundary`: put the container or component inside a boundary.
- `duplicate ID x; first at <path>:<line>`, `unresolved ID x; declare it in this diagram`, and `unsupported Mermaid form` name the view line: fix the diagram text there.

## Check it first

Run `fitness architecture-contract` from the repo root. Exit 0 means the model loads without errors.
