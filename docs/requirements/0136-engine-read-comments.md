# 0136 Read Code Comments

## Why

Every check reads my code comments the same way, so I can list them and see what a check will judge. Text inside a string, a tool directive, or a license is no comment.

## Measurement

Comment blocks listed at the line they start on
-
Comment blocks in the files

## Requirements

- 0136.1
    - Given Developer has Go, Swift, TypeScript, JavaScript, Java, Kotlin and Rust files with `//` inside a string, template or raw string
        - When `fitness comments` runs
            - Then it lists each file's comments and no text from a string
- 0136.2
    - Given Developer has block, doc and inline comments in Go, Swift and TypeScript
        - When `fitness comments` runs
            - Then it lists each block at the line it starts on
- 0136.3
    - Given Developer has shell scripts, one without an extension, and YAML files with `#` comments and a quoted `#`
        - When `fitness comments` runs
            - Then it lists each comment at its line and no quoted text
- 0136.4
    - Given Developer has tool directives, a license header and a generated file
        - When `fitness comments` runs
            - Then it lists none of them
- 0136.5
    - Given Developer names a folder and the absolute path of a file
        - When `fitness comments <paths>` runs
            - Then it lists the comments of the tracked files under them
