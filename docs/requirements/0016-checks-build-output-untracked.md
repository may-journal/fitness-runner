---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0016 Checks Build Output Untracked

## Why

I never commit compiled build output or import it from source, so every change is reviewed as real source.

## Measurement

repos with build output tracked, not ignored, or imported reported red
-
repos with build output tracked, not ignored, or imported

## Requirements

- 0016.1
    - Given Developer has a tracked TypeScript test file importing from `dist` three ways
        - When fitness runs build-output-untracked
            - Then it fails naming each import's file, line, and path
- 0016.2
    - Given Developer has a `dist` folder that `.gitignore` does not ignore
        - When fitness runs build-output-untracked
            - Then it fails asking to add `dist/` to `.gitignore`
- 0016.3
    - Given Developer has ignored `dist` folders that still have tracked files
        - When fitness runs build-output-untracked
            - Then it fails listing them with a `git rm --cached` hint
- 0016.4
    - Given Developer has an ignored `dist` and TypeScript with only local imports
        - When fitness runs build-output-untracked
            - Then it passes, counting each source file plus one
- 0016.5
    - Given Developer has a repo with no TypeScript and no `dist`
        - When fitness runs build-output-untracked
            - Then it passes with 0 files checked
- 0016.6
    - Given Developer has an untracked TypeScript file importing from `dist`
        - When fitness runs build-output-untracked
            - Then it passes, since only tracked source is judged
- 0016.7
    - Given Developer has tracked Maven and Gradle output folders and compiled classes
        - When fitness runs build-output-untracked
            - Then it fails listing them with a `git rm --cached` hint
- 0016.8
    - Given Developer has a tracked `build` folder and no Java build
        - When fitness runs build-output-untracked
            - Then it passes, since only a language's own output is judged
