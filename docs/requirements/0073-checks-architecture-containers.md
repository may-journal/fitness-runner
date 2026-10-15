---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0073 Architecture Containers

## Why

My container view matches what I ship. A new server, a missing native target, or a layer drawn as a runtime fails.

## Measurement

container views whose containers match build targets, entry points, and deploy files
-
container views checked

Source: #287

## Requirements

- 0073.1
    - Given Developer has a native app whose view declares its app and extension targets beside Node tools
        - When I check its containers
            - Then it passes
- 0073.2
    - Given Developer has a Go command, Node service, or deploy file the view does not declare
        - When I check its containers
            - Then the error names the entry point
- 0073.4
    - Given Developer has a container whose technology is not its product's language
        - When I check its containers
            - Then the error names the container and the evidence
- 0073.6
    - Given Developer has an owned store naming no path, or a platform service no external system names
        - When I check its containers
            - Then the error names the store line or the import line
- 0073.7
    - Given Developer has a target or package manifest the check cannot read
        - When I check its containers
            - Then the error names the manifest
- 0073.9
    - Given Developer has a target manifest `.gitattributes` marks `linguist-generated`
        - When I check its containers
            - Then its targets still match containers by name
- 0073.10
    - Given Developer has a container whose build target no longer exists, or whose paths hold only source layers
        - When I check its containers
            - Then the error names the container line and says to show source layers in the component view
- 0073.11
    - Given Developer has an npm library that publishes `main`, `module`, `exports`, `types`, or `files`
        - When I check its containers
            - Then a container naming its package or an entry point passes, and naming neither fails
