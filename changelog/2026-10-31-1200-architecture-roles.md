### 2026.10.31.1200

- Feat: every Go package names its container and component, with a reason, in `//fitness:container` and `//fitness:component` doc comment lines.
- Feat: the generated check fails a package that names neither, or names a container or component the views do not draw.
- Feat: the generated check fails a component line that gives no reason after its ID.
