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

Implementation history: [plans/archive/01-go-rewrite.md](./plans/archive/01-go-rewrite.md) (the Go rewrite), [plans/archive/02-npm-free.md](./plans/archive/02-npm-free.md) (removing the npm toolchain), and [plans/03-publish.md](./plans/03-publish.md) (releases and `go install`; the brew tap is tracked in issue #47). Earlier TypeScript-era plans remain in [plans/archive/](./plans/archive/).
