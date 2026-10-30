---
name: fitness
description: Fix any failing fitness check by reading its own docs. Use when a check fails or you need to know what a check requires.
---

# Fix a failing fitness check

Every check keeps its truth in docs that already exist. Read them in this order, and do not guess from the error alone.

## Read the failure

The error names the check, such as `doc-template`. Find it in [the checks index](../../docs/checks.md).

## Read the check

Open `go/cmd/fitness-check-<name>/README.md`. It states the rule, the exact errors and how a pass looks.

A repo's own check lives in its `fitness/<name>/` folder instead, with its README beside its Go source. To add one, write it in Go on the check SDK with unit tests, as the [local checks guide](../../docs/local-checks.md) shows.

## Read the requirements

Open [docs/requirements](../../docs/requirements) and search for the check name. Each requirement is proven by a test that runs the real product. The engine's own rules are there too, such as [engine front matter](../../docs/requirements/0096-engine-strip-front-matter.md).

## Read the architecture

Open [docs/architecture](../../docs/architecture) for where the check sits. [The check protocol](../../docs/check-protocol.md) explains how checks report.

## Fix and rerun

Fix the code or doc the error points at, never the check. Run `fitness <name>` from the repo root. Exit 0 means it passes.
