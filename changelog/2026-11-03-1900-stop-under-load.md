### 2026.11.03.1900

- Test: requirement 0144.1 proves that, with every CPU busy, a check past its budget ends within 1s past it.
- Test: 0144.1 also proves no tool the check started, nor any child of one, outlives that second.
- Test: requirement 0001.14 no longer fails when a timed-out tool never wrote its process ID.
- Test: 0001.14 and the prompt tests still fail if any process a fixture tool started runs on.
