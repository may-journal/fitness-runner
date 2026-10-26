---
fitnessFunctions: ['architecture-contract', 'architecture-actors']
---

# System Context

<!-- Start docs/architecture/01-system-context.md from this template. One `C4Context` diagram, the only form this level takes: each person and outside system the product meets, and the product as one System. Its kinds are Person, System and `System_Ext`, with their Db and Queue forms. Each element carries the one `UpdateElementStyle` line of its kind below: slate people, blue systems, grey externals, white text. A number opens each label and a letter labels each arrow, A to Z, then AA to ZZ. Each has a row in the callout table. -->

```mermaid
C4Context
    title Product

    Person(user, "1 User")
    System(product, "2 Product", "What it does for the user.")
    System_Ext(service, "3 Outside service", "What it lends the product.")

    Rel(user, product, "A")
    Rel(product, service, "B")

    UpdateElementStyle(user, $bgColor="#334155", $borderColor="#1e293b", $fontColor="#ffffff")
    UpdateElementStyle(product, $bgColor="#1d4ed8", $borderColor="#1e3a8a", $fontColor="#ffffff")
    UpdateElementStyle(service, $bgColor="#64748b", $borderColor="#475569", $fontColor="#ffffff")
```

<!-- The legend is a second C4Context diagram titled Legend, with one sample of each element kind above in its own colors. -->

```mermaid
C4Context
    title Legend

    Person(p, "Person", "Human or agent actor")
    System(s, "System", "Software we own")
    System_Ext(e, "External", "Outside our boundary")

    UpdateElementStyle(p, $bgColor="#334155", $borderColor="#1e293b", $fontColor="#ffffff")
    UpdateElementStyle(s, $bgColor="#1d4ed8", $borderColor="#1e3a8a", $fontColor="#ffffff")
    UpdateElementStyle(e, $bgColor="#64748b", $borderColor="#475569", $fontColor="#ffffff")
```

Numbers on nodes and letters on arrows match the callout table.

| #   | Description                         | Why                                       |
| --- | ----------------------------------- | ----------------------------------------- |
| 1   | The person who uses the product.    | Requirements name this actor in a Given.  |
| 2   | The product this repo builds.       | The boundary the other views break down.  |
| 3   | A system the product relies on.     | It sits outside the product's boundary.   |
| A   | The user works in the product.      | The main way in.                          |
| B   | The product calls the outside service. | The one dependency it does not own.    |
