---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0074 Architecture Actors

## Why

I name every role a requirement uses in the system context, so a new role cannot slip into a requirement unnoticed.

## Measurement

Given clauses that start with a declared actor
-
Given clauses

Source: #286

## Requirements

- 0074.1
    - Given Developer has requirement docs and no architecture
        - When I check the actors
            - Then it passes with zero files
- 0074.2
    - Given Developer has Givens that start with an actor's full name or its ID
        - When I check the actors
            - Then they pass, and a partial or undeclared name fails naming the Given line and the context diagram
- 0074.3
    - Given Developer has a Given that names only a state
        - When I check the actors
            - Then it fails naming the Given line and the context diagram
- 0074.4
    - Given Developer has a Given naming an actor the context no longer declares
        - When I check an unrelated change
            - Then it fails naming the Given line and the context diagram
- 0074.5
    - Given Developer has an actor whose ID is another actor's name
        - When I check a Given that uses it
            - Then it fails naming both context lines
- 0074.6
    - Given Developer has several actors joined by and before the state
        - When I check the actors
            - Then they pass, and an undeclared one among them fails
- 0074.8
    - Given Developer has an acceptance the requirements check rejects
        - When I check the actors
            - Then it reports nothing for that acceptance
- 0074.9
    - Given Developer has a Given that starts with a system, a legend node, or a flowchart node classed person
        - When I check the actors
            - Then it fails naming the Given line and the context diagram, though the flowchart node only warns until opted in
