---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0072 Architecture Containers

## Why

My container view matches what I ship. A new server, a missing native target, or a layer drawn as a runtime fails.

## Measurement

container views whose containers match build targets, entry points, and deploy files
-
container views checked

Source: #287

## Requirements

- 0072.1
    - Given a native app whose view declares its app and extension targets beside Node tools
        - When I check its containers
            - Then it passes
- 0072.2
    - Given a Go command, Node service, or deploy file the view does not declare
        - When I check its containers
            - Then the error names the entry point
- 0072.3
    - Given a container whose build target no longer exists
        - When I check its containers
            - Then the error names the container line
- 0072.4
    - Given a container whose technology is not its product's language
        - When I check its containers
            - Then the error names the container and the evidence
- 0072.5
    - Given a container whose paths hold only source layers
        - When I check its containers
            - Then the error says to show them in the component view
- 0072.6
    - Given an owned store naming no path, or a platform service no external system names
        - When I check its containers
            - Then the error names the store line or the import line
- 0072.7
    - Given a target or package manifest the check cannot read
        - When I check its containers
            - Then the error names the manifest
- 0072.8
    - Given unchanged container drift
        - When I check unrelated changes
            - Then the same findings remain
