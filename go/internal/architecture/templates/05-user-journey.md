---
fitnessFunctions: ['architecture-contract', 'architecture-journey']
---

# User Journey

## Legend

<!-- One flowchart sample of each node type the journey uses, styled by the same classes, with its own callout table. -->

```mermaid
flowchart TB
    subgraph types["Node types"]
        persona((1 Persona)):::persona
        screen[2 Screen]:::screen
        action(3 Action):::action
        system[(4 System)]:::system
    end

    classDef persona fill:#eef2ff,stroke:#6366f1,color:#312e81,stroke-width:2px
    classDef screen fill:#f8fafc,stroke:#64748b,color:#1e293b,stroke-width:2px
    classDef action fill:#fff7ed,stroke:#f97316,color:#9a3412,stroke-width:2px
    classDef system fill:#ecfdf5,stroke:#10b981,color:#064e3b,stroke-width:2px
```

Numbers match the callout table.

| #   | Description | Why                                      |
| --- | ----------- | ---------------------------------------- |
| 1   | Persona     | A person the system context declares.    |
| 2   | Screen      | What the person sees.                    |
| 3   | Action      | What the person does.                    |
| 4   | System      | What the product does in reply.          |

## Journey

<!-- One flowchart TB, the only form this level takes: the person's steps from start to result, each node styled by one of the four classes above and no other style. Each label ends with the requirement docs the step fulfils, by ID, and every requirement doc appears on some step. -->

```mermaid
flowchart TB
    user((1 User)):::persona
    open(2 Open the app<br/>Req 0001):::action
    saved[3 Record saved<br/>Req 0001]:::screen

    user --> open
    open --> saved

    classDef persona fill:#eef2ff,stroke:#6366f1,color:#312e81,stroke-width:2px
    classDef screen fill:#f8fafc,stroke:#64748b,color:#1e293b,stroke-width:2px
    classDef action fill:#fff7ed,stroke:#f97316,color:#9a3412,stroke-width:2px
```

Numbers match the callout table.

| #   | Description          | Type    | Why                                   |
| --- | -------------------- | ------- | ------------------------------------- |
| 1   | The user.            | Persona | Same person as the system context.    |
| 2   | Opens the app.       | Action  | The first step of every visit.        |
| 3   | The record is saved. | Screen  | What the user came to do.             |
