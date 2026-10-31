# 0134 Architecture Roles

## Why

Each Go package names its container and component in its own doc comment, so the views can be drawn from the code.

## Measurement

Go packages that name a container and a component the views draw, with a reason

-

Go packages in the fitness repo

## Requirements

- 0134.1
  - Given AI agent reads the fitness repo's Go packages
    - When it runs the generated check
      - Then the check passes, since each package names a container and component the views draw
- 0134.2
  - Given Developer drops the component line from a package's doc comment
    - When they run the generated check
      - Then it fails naming the package and the lines its doc comment needs
- 0134.3
  - Given Developer names a component the components view does not draw
    - When they run the generated check
      - Then it fails naming the component and the view
- 0134.4
  - Given Developer names a container the containers view does not draw
    - When they run the generated check
      - Then it fails naming the container and the view
- 0134.5
  - Given Developer names a component without a reason
    - When they run the generated check
      - Then it fails asking for a reason after the ID
