# coverage

Every coverage check judges by one goal, whichever language it measures: every counted line, branch or statement covered.

A check compares exact counts, so a total that rounds to the goal still falls short. The goal lives in go/pkg/goal.

## Percent

Percent is the coverage goal, as a percentage.

## Met

Met reports whether covered of total meets the goal. Nothing to cover meets it.

## Text

Text is the goal as findings print it, such as "100%".
