### 2026.11.27.0115

> A check that reads a Vitest or Jest suite sees each requirement test's named cases.

- Feat: the Node module lists each requirement test's case rows with a fixed `name`, their `file:line` and step ID.
- Fix: only array elements count as case table rows.
- Docs: requirement 0009.4 states the case behavior as a numbered acceptance.
