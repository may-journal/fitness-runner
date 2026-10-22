# 0100 Single Folder

## Why

I check one folder of a big repo again after a fix without waiting for the rest of it.

## Measurement

runs that judge only the files under the folder they were given
-
runs given a folder

Source: `go/cmd/fitness/dir.go:20`

## Requirements

- 0100.1
    - Given Developer has bad files in two folders
        - When fitness runs a check with `--dir` naming one folder
            - Then it reports only that folder's files
- 0100.2
    - Given Developer names a folder that is missing, outside the repo, or holds no files
        - When fitness runs with `--dir`
            - Then it stops with an error naming the folder, not a scan of the whole repo
