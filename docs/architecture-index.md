---
relatedConfigurations: ['../.fitnessrc.json']
---

# Architecture

C4 model for fitness-runner. Read [architecture/](architecture/) in order — system context, then containers, components, and code.

| Level          | Doc                                                                    |
| -------------- | ---------------------------------------------------------------------- |
| System context | [architecture/01-system-context.md](architecture/01-system-context.md) |
| Containers     | [architecture/02-containers.md](architecture/02-containers.md)         |
| Components     | [architecture/03-components.md](architecture/03-components.md)         |
| Code           | [architecture/04-code.md](architecture/04-code.md)                     |

Strategic maps: [wardley.md](./wardley.md) · [competition.md](./competition.md)

Research: [research/0001-prose-cognitive-complexity.md](./research/0001-prose-cognitive-complexity.md) — the study behind the `text-readability` check.

Implementation history now lives as GitHub Issues under the `Plan` label: the Go rewrite (#51), removing the npm toolchain (#52), and publishing releases and `go install` (#49). Earlier TypeScript-era plans are closed `Plan` issues (#53–#58).
