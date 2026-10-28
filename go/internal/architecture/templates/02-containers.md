# Containers

<!-- Start docs/architecture/02-containers.md from this template. One `C4Container` diagram, the only form this level takes: each unit that runs or ships, inside the product's `System_Boundary`, named for the folder it is built from, with its technology. Its kinds are Person, `System_Boundary`, Container and `System_Ext`, with their Db and Queue forms. Reuse the people and IDs of the system context. Each element carries the one `UpdateElementStyle` line of its kind below: slate people, teal containers, grey externals, white text, and no style on the boundary. -->

```mermaid
C4Container
    title Product

    Person(user, "1 User")

    System_Boundary(product, "2 Product") {
        Container(app, "3 App", "Language", "src/app")
    }

    System_Ext(service, "4 Outside service", "What it lends the product.")

    Rel(user, app, "A")
    Rel(app, service, "B")

    UpdateElementStyle(user, $bgColor="#334155", $borderColor="#1e293b", $fontColor="#ffffff")
    UpdateElementStyle(app, $bgColor="#0f766e", $borderColor="#115e59", $fontColor="#ffffff")
    UpdateElementStyle(service, $bgColor="#64748b", $borderColor="#475569", $fontColor="#ffffff")
```

<!-- The legend is a second C4Container diagram titled Legend, with one sample of each element kind above in its own colors. -->

```mermaid
C4Container
    title Legend

    Person(p, "Person", "Human or agent actor")
    System_Boundary(b, "System boundary") {
        Container(c, "Container", "Tech", "Runnable or deployable unit")
    }
    System_Ext(e, "External", "Outside our boundary")

    UpdateElementStyle(p, $bgColor="#334155", $borderColor="#1e293b", $fontColor="#ffffff")
    UpdateElementStyle(c, $bgColor="#0f766e", $borderColor="#115e59", $fontColor="#ffffff")
    UpdateElementStyle(e, $bgColor="#64748b", $borderColor="#475569", $fontColor="#ffffff")
```

Numbers on nodes and letters on arrows match the callout table.

| #   | Description                              | Why                                     |
| --- | ---------------------------------------- | --------------------------------------- |
| 1   | The person from the system context.      | Every run starts with this actor.       |
| 2   | The product boundary.                    | Everything inside it ships from here.   |
| 3   | The app, built from `src/app`.           | One unit per runtime or shipped binary. |
| 4   | The outside service from the context.    | The product does not own it.            |
| A   | The user runs the app.                   | The main way in.                        |
| B   | The app calls the outside service.       | The one dependency it does not own.     |
