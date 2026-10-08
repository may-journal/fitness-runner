---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0006 Requirement Docs Read the Same

## Why

Anyone can open any requirement and find its reason, its measure, and its proof in the same place.

## Measurement

requirement docs with no shape errors
-
requirement docs

Source: `go/cmd/fitness-check-requirements/doc.go:86`

## Requirements

- 0006.1
    - Given a doc whose file name and title carry different IDs
        - When fitness runs
            - Then it fails
- 0006.2
    - Given two docs with the same ID
        - When fitness runs
            - Then it fails
- 0006.3
    - Given a doc with sections other than Why, Measurement, and Requirements
        - When fitness runs
            - Then it fails
- 0006.4
    - Given a Measurement that is not a ratio with a Source
        - When fitness runs
            - Then it fails
- 0006.5
    - Given a Source citing neither one existing code line nor one issue
        - When fitness runs
            - Then it fails
- 0006.6
    - Given an acceptance with a second Given, When, or Then
        - When fitness runs
            - Then it fails
- 0006.7
    - Given an acceptance chain out of order or not nested
        - When fitness runs
            - Then it fails
- 0006.8
    - Given an acceptance ID that is repeated or names another requirement
        - When fitness runs
            - Then it fails
