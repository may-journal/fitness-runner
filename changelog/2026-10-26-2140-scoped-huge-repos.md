### 2026.10.26.2140

- Fix: `jscpd` on a scoped run reads the changed files and the tracked files sharing their endings, as jscpd compares one format.
- Fix: the six `architecture-*` checks pass with zero files when a scoped run changes no view, requirement doc, source or manifest.
- Fix: the architecture inventory classes only sources and manifests, so the generated and dev-script passes skip the rest.
- Test: requirements 0032.13 and 0076.1 prove both scoped rules with a hooked repo, and 0032.12 merges the two minimums.
- Docs: `FITNESS_TREE_SIZES` times a test and a product commit, and each Scope section states its scoped reading; relatedConfigurations .fitnessrc.json goes.
