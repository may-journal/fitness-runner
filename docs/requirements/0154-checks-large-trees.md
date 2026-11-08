# 0154 Checks Large Trees

## Why

When my repo tracks tens of thousands of files, the build output and components checks still finish in seconds. A large tree never spends the fast total on them.

## Measurement

large trees whose build output and components checks each pass within five seconds

-

large trees

## Requirements

- 0154.1
  - Given Developer has a Java project, TypeScript, an ignored dist folder and a components view among 50000 tracked files
    - When build-output-untracked and architecture-components judge the whole repo
      - Then each passes within five seconds
