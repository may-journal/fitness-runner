---
fitnessFunctions: ['architecture-contract', 'architecture-containers']
---

# Containers

<!-- Copy to docs/architecture/02-containers.md. One C4Container diagram: each unit that runs or ships, inside the product's boundary, named for the folder it is built from. Reuse the people and IDs of the system context. -->

```mermaid
C4Container
    title Product

    Person(user, "1 User")

    System_Boundary(product, "2 Product") {
        Container(app, "3 App", "Language", "src/app")
    }

    System_Ext(service, "4 Outside service", "What it lends the product.")

    Rel(user, app, "5")
    Rel(app, service, "6")
```

<!-- The legend is a second C4Container diagram titled Legend, with one sample of each element kind above. -->

```mermaid
C4Container
    title Legend

    Person(p, "Person", "Human or agent actor")
    System_Boundary(b, "System boundary") {
        Container(c, "Container", "Tech", "Runnable or deployable unit")
    }
    System_Ext(e, "External", "Outside our boundary")
```

Numbers on nodes and arrows match the callout table.

| #   | Description                              | Why                                     |
| --- | ---------------------------------------- | --------------------------------------- |
| 1   | Same person as the system context.       | Every run starts with this actor.       |
| 2   | The product's boundary.                  | Everything inside it ships from here.   |
| 3   | The app, built from `src/app`.           | One unit per runtime or shipped binary. |
| 4   | Same outside service as the context.     | The product does not own it.            |
| 5   | The user runs the app.                   | The main way in.                        |
| 6   | The app calls the outside service.       | The one dependency it does not own.     |
