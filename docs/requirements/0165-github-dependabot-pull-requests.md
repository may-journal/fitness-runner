# 0165 GitHub Dependabot pull requests

## Why

PR check judges every pull request alike, and Dependabot writes none of what it asks for. So a workflow completes the PR, and no check is excluded.

## Measurement

Dependabot pull requests that pass pr-check unchanged
-
Dependabot pull requests

## Requirements

- 0165.1
    - Given Developer has a recorded Dependabot payload whose title is not a semantic commit
        - When dependabot-pr completes it and pr-check judges the result
            - Then the title, description, bump issue and changelog entry pass, and the release notes stay in the description
- 0165.2
    - Given Developer has an open issue for the same bump
        - When dependabot-pr completes the pull request
            - Then the description closes that issue and no second issue opens
- 0165.3
    - Given Developer has a pull request that a person opened
        - When dependabot-pr runs on it
            - Then it writes nothing to GitHub
- 0165.4
    - Given Developer has a Dependabot pull request that is already complete
        - When dependabot-pr runs on it again
            - Then it writes nothing to GitHub
