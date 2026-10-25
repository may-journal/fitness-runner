---
name: architecture-containers
description: Fix a failing architecture-containers check on the containers view. Use when a shipped product has no container or a container matches nothing shipped.
---

# Match the containers view to what ships

Each product the repo ships has one container in the [containers view](../../docs/architecture/02-containers.md), and each container matches a product. [The check README](../../go/cmd/fitness-check-architecture-containers/README.md) lists the evidence a product needs.

## Read the failure

- `<kind> <name> has no container in the containers view`: add a container whose path covers the product folder, or whose name is the product's name.
- `<kind> <name> is claimed by containers A at 02-containers.md:10 and B at 02-containers.md:11`: keep one claim.
- `container A says "Go" but <kind> <name> at go/cmd/x is <technology>`: write the product's real technology in the container.
- `container A matches no build target, entry point, or deploy file by name`: remove it, or point its paths at the product folder.
- `container A claims only source layers (...): no build target, entry point, or deploy file is under them; show them in the component view`: draw those layers as components.
- `store S names no schema, configuration, or source path`: add the path that defines the store.
- `<service> is used here but no external system in the containers view names it`: add an external system naming it.

## Check it first

Run `fitness architecture-containers` from the repo root. Exit 0 means containers and products match.
