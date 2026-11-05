# 0144 Engine Stops Under Load

## Why

A check past its budget stops on time with every tool it started, even on a busy machine. A slow check never holds up my commit or leaves processes behind.

## Measurement

Timed-out checks that end, with their tools, within 1s past the budget
-
Timed-out checks

## Requirements

- 0144.1
    - Given Developer has a check past its budget, whose tool starts children or ignores the stop signal, while every CPU is busy
        - When fitness runs
            - Then the check ends within 1s past its budget, and none of its tools outlive the run
