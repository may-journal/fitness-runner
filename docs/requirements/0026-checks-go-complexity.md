---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0026 Checks Go Complexity

## Why

I keep every Go function small enough to read and test, the same way the house lint rule gates JavaScript.

## Measurement

Go functions within the complexity ceiling
-
Go functions scored

Source: `go/cmd/fitness-check-go-complexity/main.go:90`

## Requirements

- 0026.1
    - Given Developer has a Go function with more than 5 branches
        - When go-complexity checks the repo
            - Then it fails naming the file, line, function, score, and ceiling
- 0026.2
    - Given Developer has a function with `switch` and `select` cases, `default`, `for`, `range`, `&&`, and `||`
        - When go-complexity scores it
            - Then each adds one point except `default`
- 0026.3
    - Given Developer has a function literal with many branches inside a simple function
        - When go-complexity checks the repo
            - Then it fails naming only the function literal
- 0026.4
    - Given Developer has a method with too many branches
        - When go-complexity checks the repo
            - Then it fails naming the method
- 0026.5
    - Given Developer has a test file with too many branches
        - When go-complexity checks the repo
            - Then it fails like any other Go file
- 0026.6
    - Given Developer has a repo that sets `goComplexity.max` in `.fitnessrc.json`
        - When go-complexity checks the repo
            - Then it judges against that ceiling
- 0026.7
    - Given Developer has a Go file that does not parse
        - When go-complexity checks the repo
            - Then it fails with the parser's error
- 0026.8
    - Given Developer has a repo with no Go files
        - When fitness runs go-complexity
            - Then it passes with 0 files
