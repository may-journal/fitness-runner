---
name: fitness
description: Fix any failing fitness check by reading its own docs. Use when a check fails or you need to know what a check requires.
hooks:
  PostToolUse:
    - matcher: "Edit|Write|MultiEdit|NotebookEdit"
      hooks:
        - type: command
          command: fitness claude-hook
          timeout: 30
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: fitness claude-hook
          timeout: 30
  Stop:
    - hooks:
        - type: command
          command: fitness claude-hook
          timeout: 900
  SubagentStop:
    - hooks:
        - type: command
          command: fitness claude-hook
          timeout: 300
  SessionStart:
    - hooks:
        - type: command
          command: fitness claude-hook
          timeout: 30
  SessionEnd:
    - hooks:
        - type: command
          command: fitness claude-hook
          timeout: 30
---

# Fix a failing fitness check

Every check keeps its truth in docs that already exist. Read them in this order, and do not guess from the error alone.

## Read the failure

The error names the check, such as `doc-template`. Find it in [the checks index](../../docs/checks.md).

## Read the check

Open `go/cmd/fitness-check-<name>/README.md`. It states the rule, the exact errors and how a pass looks.

A repo's own check lives in its `fitness/<name>/` folder instead, with its README beside its Go source. To add one, write it in Go on the check SDK with unit tests, as the [local checks guide](../../docs/local-checks.md) shows.

## Read the requirements

Open [docs/requirements](../../docs/requirements) and search for the check name. Each requirement is proven by a test that runs the real product. The engine's own rules are there too, such as [engine front matter](../../docs/requirements/0096-engine-strip-front-matter.md).

## Read the architecture

Open [docs/architecture](../../docs/architecture) for where the check sits. [The check protocol](../../docs/check-protocol.md) explains how checks report.

## Fix and rerun

Fix the code or doc the error points at, never the check. Run `fitness <name>` from the repo root. Exit 0 means it passes.

## Hooks in Claude Code

`fitness claude-hook` judges Claude's work as it goes and hands any findings back to fix in the same turn:

- After an edit tool writes a file, it runs the file checks on that one file.
- Only checks within the default budget run there. Tests, vet, linters and network checks wait for the stop.
- Before a `gh` issue or PR call posts a body, it judges that body as `fitness --blob` does.
- A red body denies the call: create, edit, comment or review.
- When Claude stops, it runs every file and git check on the uncommitted files, and blocks the same findings once per session.
- When a subagent stops, it runs those within four minutes on the files that subagent edited, and blocks each subagent once.
- When a session on a feature branch starts, resumes, clears or compacts, it adds the edit-check findings on that branch's changed files.
- Each judged event adds a line to `claude-hooks.jsonl` in the fitness cache folder, and SessionEnd adds the session's total per event.

Commit and push stay with the git hooks. A repo with no `.fitnessrc.json` and no fitness git hooks passes silently.

This skill's front matter declares the hooks for the session that uses it. `fitness-install --claude-settings ~/.claude/settings.json` adds them to every session, run by the installed fitness; run it again after an upgrade. Requirements [0142](../../docs/requirements/0142-hooks-claude-hooks.md) and [0143](../../docs/requirements/0143-hooks-claude-session-hooks.md) state each rule.
