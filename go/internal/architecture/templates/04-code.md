# Code

<!-- Start docs/architecture/04-code.md from this template. One `classDiagram`, the only form this level takes: the types and functions a reader needs, each labeled with its number and its name in the code, and each arrow a call or reference the code makes, labeled with a letter. Each class carries the one violet style line below, with white text; `classDef` and `:::` color nothing here. -->

```mermaid
classDiagram
    class Record["1 Record"] {
        +ID
    }
    class save["2 save(Record) error"]

    save --> Record : A

    style Record fill:#6d28d9,stroke:#4c1d95,color:#ffffff
    style save fill:#6d28d9,stroke:#4c1d95,color:#ffffff
```

<!-- The legend is a second classDiagram with no numbers, showing each class and arrow kind above in the same violet. -->

```mermaid
classDiagram
    class Example {
        +property
        +method()
    }
    class Parent
    class Child
    Parent <|-- Child : inherits
    Parent --> Child : uses

    style Example fill:#6d28d9,stroke:#4c1d95,color:#ffffff
    style Parent fill:#6d28d9,stroke:#4c1d95,color:#ffffff
    style Child fill:#6d28d9,stroke:#4c1d95,color:#ffffff
```

Numbers on classes and letters on relationships match the callout table.

| #   | Description                           | Why                                  |
| --- | ------------------------------------- | ------------------------------------ |
| 1   | One saved record.                     | The type every store call shares.    |
| 2   | Writes one record.                    | The only write path.                 |
| A   | Save takes the record it writes.      | The call the code makes.             |
