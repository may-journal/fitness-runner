# 0001 Install and Update Fitness

## Why

I get the exact, verified fitness and its skills with one installer, and my old files move to the current shape.

## Measurement

installs that run the verified release they asked for

-

installs

## Requirements

- 0001.1
  - Given Developer has any folder
    - When I run `fitness-install` with its options
      - Then it runs the verified release, or says why it cannot and exits 1
- 0001.2
  - Given Developer has a GitHub job
    - When it runs `fitness-install --github-action`
      - Then it writes the folder to the job outputs and summarizes the result
- 0001.3
  - Given Developer asks for the latest release
    - When `fitness-install` looks it up
      - Then it sends my token, asks once per 10 minutes, and survives a refusal
- 0001.4
  - Given Developer has an agent's skills folder
    - When I run `fitness-install --link-skills` with it
      - Then every skill in the release links there, readable, and my own skills stay
- 0001.5
  - Given Developer has the onboarding skill linked
    - When I follow it
      - Then it covers adopting fitness, and every doc it links exists
- 0001.6
  - Given Developer has a Claude Code settings file of my own
    - When I run `fitness-install --claude-settings` with it, twice
      - Then it holds each fitness hook once, and my settings stay
- 0001.7
  - Given Developer has docs with old fitness front matter
    - When `fitness strip-front-matter` runs
      - Then only the fitness keys go, and every body stays
- 0001.8
  - Given Developer has ADR files
    - When `fitness adr-migrate` runs
      - Then each becomes one `ADR` issue, none lost or opened twice
