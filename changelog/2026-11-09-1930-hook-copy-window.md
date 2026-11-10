### 2026.11.09.1930

- Fix: the Claude hook's second copy of an event passes even when the first copy judged for longer than 5s.
- Fix: the 5s window now runs from when the first copy finishes judging.
- Test: requirement 0142.7 judges with a local check that takes 6s.
