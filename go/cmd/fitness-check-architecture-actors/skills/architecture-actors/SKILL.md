---
name: architecture-actors
description: Fix a failing architecture-actors check on a requirement Given. Use when a Given names a role the system context does not declare.
---

# Start each Given with a declared actor

A requirement's Given starts with one actor, or several joined by `and`, that the [system context](../../../../../docs/architecture/01-system-context.md) declares as a person. [The check README](../../README.md) lists the grammar.

## Read the failure

- `Given "Admin has joined" starts with no actor; actors declared at ... are Developer`: start the Given with a declared name or ID. Add the person to the context diagram if the role is real.
- `Given "Host and" names no actor at ""`: the spelling after `and` is empty or unknown. Name a second actor there.
- `no actor is declared at docs/architecture/01-system-context.md:1`: the context view declares no person. Add each role as a `Person` or a node classed `person`.
- `Given actor "Host" is ambiguous: <id> at <path>:<line> and <id> at <path>:<line>`: one actor's ID is another actor's name. Give the two actors distinct names and IDs.

A spelling matches as a whole word with the diagram's case, so `Hosts` does not match `Host`.

## Check it first

Run `fitness architecture-actors` from the repo root. Exit 0 means every Given starts with a declared actor.
