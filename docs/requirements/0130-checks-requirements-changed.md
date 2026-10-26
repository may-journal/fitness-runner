# 0130 Checks Requirements Changed

## Why

When I change product code, I change its test and requirement doc with it, or I say why not in a commit trailer.

## Measurement

changes to product code that also change a test and a requirement doc, or carry the trailer

-

changes to product code

## Requirements

- 0130.1
  - Given Developer has changed product code without a test or requirement doc
    - When requirements-changed checks the commit or its pull request
      - Then it fails naming the file and the missing half
- 0130.2
  - Given Developer has staged product code with a changed test and requirement doc
    - When requirements-changed checks the staged commit
      - Then it passes
- 0130.3
  - Given Developer has staged only docs, config, changelog, a rename, or a whitespace edit
    - When requirements-changed checks the staged commit
      - Then it passes
- 0130.4
  - Given Developer has a merge in progress that replays a product code change
    - When requirements-changed checks the staged commit
      - Then it passes
- 0130.5
  - Given Developer has staged the deletion of product code no requirement doc names
    - When requirements-changed checks the staged commit
      - Then it passes
- 0130.6
  - Given Developer has staged the deletion of product code its requirement doc still names
    - When requirements-changed checks the staged commit
      - Then it fails naming the doc and tests left behind
- 0130.7
  - Given Developer has a repo with no requirement docs yet
    - When requirements-changed checks staged product code
      - Then it passes
- 0130.8
  - Given Developer has the hooks installed and staged product code alone
    - When I commit with a `Requirements: none, <reason>` trailer
      - Then the commit lands
