---
fitnessFunctions: ['architecture-containers']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# architecture-containers

The containers view matches what the repository ships. A new server, a missing native target, a wrong technology, or a source layer drawn as a runtime fails. The view comes from the model `architecture-contract` reads; this check adds no fields or columns.

## What ships

Each product is proved by a build target, an entry point, or a deploy file. A dependency manifest alone proves nothing, so a native app may keep Node packages for lint and build tools. Build manifests such as `project.pbxproj` are read even when `.gitattributes` marks them `linguist-generated`.

| Evidence | Product | Technology |
| --- | --- | --- |
| A product `.go` file declaring `package main` without a `go:build ignore` constraint | Go command, one per folder | Go |
| An XcodeGen `project.yml` target of type `application`, `*extension`, or `tool`, its own or from `targetTemplates` | App, extension, or tool target | Swift |
| A `project.pbxproj` native target with the same product types | App, extension, or tool target | Swift |
| A `package.json` with a `start` script or `bin`, or a deploy file beside it | Node service | Node |
| Any other `package.json` with `main`, `module`, `exports`, `types`, or `files` | Node library | Node |

A template source may name `${target_name}`; other template attributes are not substituted. Deploy files are `Dockerfile`, `Procfile`, `fly.toml`, `app.yaml`, `serverless.yml`, `vercel.json`, and `wrangler.toml`. Xcode frameworks, libraries, and test bundles, and files under test or tool folders, are source types, never products.

A product `.swift` file importing CloudKit, StoreKit, HealthKit, or WeatherKit uses a platform service.

## What the view says

An owned container is a `Container`, `ContainerDb`, or `ContainerQueue` inside a boundary. A container owns a product when a path it names covers the product's folder, or when its name is the product's name. A trailing `*` in a path, such as `go/cmd/fitness-check-*`, names every sibling folder it matches.

A container also owns an npm package when its text names the package or a path covering an entry point. Entry points are `main`, `module`, `types`, `files`, each `bin`, and each `exports` path, so `dist/esm` names `dist/esm/index.js` though nobody tracks it.

External systems, `System_Ext` and `Container_Ext`, need no evidence. A platform service the code uses needs an external system whose name, technology, or description names it.

## Failures

Every error names the view line or the evidence line; an error about both names both.

- A product no owned container claims, naming its entry point, target, or deploy file.
- A product two containers claim.
- A `Container` that matches no product: by name, or through paths that hold only source layers.
- A container whose technology does not name its product's language.
- An owned store that names no schema, configuration, or source path.
- A platform service the code uses with no external system naming it.
- A target without a type, a target source folder that does not exist, or a `package.json` the check cannot parse.

## Runs

No architecture views means a pass with zero files; `filesChecked` counts the container views. Scoped runs read every view and every product, so a change on either side still meets the other.

## Requirements

[Requirement 0073](../../../docs/requirements/0073-checks-architecture-containers.md) has one installed-command test per acceptance.
