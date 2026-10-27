# 0132 Paired Checks

## Why

Each architecture decision and requirement in my repo holds only while a check proves it, so the doc and its check change together.

## Measurement

ADRs and requirement docs with a check that reads the files they govern
-
ADRs and requirement docs in repos with a fitness folder

## Requirements

- 0132.1
    - Given Developer has an ADR as the README of its check in `fitness/adr-NNNN/`
        - When the suite runs
            - Then adr-checks passes, and the ADR's check reports its verdict
- 0132.2
    - Given Developer has a fitness folder and an ADR kept apart from its check, in a folder named `adr`
        - When the suite runs
            - Then adr-checks reports red naming the ADR and the check README to move it into, and exits 1
- 0132.3
    - Given Developer has a `fitness/adr-NNNN/` README that lacks an ADR's Context, Decision or Consequences section
        - When the suite runs
            - Then adr-checks reports red naming the README and each section it lacks, and exits 1
- 0132.4
    - Given Developer has an ADR whose status says it is superseded, with no check program beside it
        - When the suite runs
            - Then adr-checks passes
- 0132.5
    - Given Developer changes an ADR's decision without its check's code
        - When fitness judges the commit or the pull request
            - Then adr-checks reports red naming the check, and exits 1
- 0132.6
    - Given Developer has a requirement doc proven by a check in `fitness/req-NNNN/` and no test
        - When the full suite runs
            - Then the requirements check passes, and the requirement's check reports its verdict
- 0132.7
    - Given Developer has a requirement check whose doc is missing, or changes the check without its doc
        - When the requirements check runs
            - Then it reports red naming the check and the doc, and exits 1
- 0132.8
    - Given Developer has an ADR or requirement check that passes having judged no files
        - When the check runs
            - Then it reports red saying its scan is broken, and exits 1
