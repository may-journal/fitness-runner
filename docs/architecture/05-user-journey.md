# User Journey

## Legend

```mermaid
flowchart TB
    subgraph types["Node types"]
        persona((1 Persona)):::persona
        screen[2 Screen]:::screen
        action(3 Action):::action
        system[(4 System)]:::system
        blocked[5 Blocked]:::blocked
    end

    classDef persona fill:#eef2ff,stroke:#6366f1,color:#312e81,stroke-width:2px
    classDef screen fill:#f8fafc,stroke:#64748b,color:#1e293b,stroke-width:2px
    classDef action fill:#fff7ed,stroke:#f97316,color:#9a3412,stroke-width:2px
    classDef system fill:#ecfdf5,stroke:#10b981,color:#064e3b,stroke-width:2px
    classDef blocked fill:#fef2f2,stroke:#ef4444,color:#991b1b,stroke-width:2px

    style types fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
```

Numbers match the callout table. "Screen" is what the person reads: terminal output or a GitHub page.

| #   | Description | Why                                                        |
| --- | ----------- | ---------------------------------------------------------- |
| 1   | Persona     | Developer or AI agent — same CLI, same exit-code contract. |
| 2   | Screen      | What the person reads in the terminal or on GitHub.        |
| 3   | Action      | A command, edit or post the person makes.                  |
| 4   | System      | A step fitness, a hook or a workflow takes for them.       |
| 5   | Blocked     | A red result that stops the person until they fix it.      |

## Journey

Each subgroup is one requirement doc, titled with its ID, in order from top to bottom. The `architecture-journey` check reads those titles, so every step inside names its requirement.

```mermaid
flowchart TB
    dev((1 Developer or agent)):::persona

    subgraph r1["Req 0001 Install and update fitness"]
        install(2 Run fitness-install):::action
        verify[(3 Verify the release checksum)]:::system
        link(4 Link skills and Claude settings):::action
    end

    subgraph r2["Req 0002 Get the checks a run needs"]
        pins[(5 Read the pinned check versions)]:::system
        download[(6 Download only the checks the run reaches)]:::system
        cached[(7 Run cached checks offline)]:::system
    end

    subgraph r3["Req 0003 Run checks and read results"]
        run(8 Run fitness on the repo, a folder or a text):::action
        results[9 Results table, JSON or annotations]:::screen
        red[10 Red row names the fix]:::blocked
        explain(11 Explain the check, then edit):::action
    end

    subgraph r4["Req 0004 Choose and configure checks"]
        choose(12 Turn checks on, off or add one):::action
        config[13 fitness config shows why each runs]:::screen
        orgRule[14 Turning off an org check goes red]:::blocked
    end

    subgraph r5["Req 0005 Guard commits and pushes"]
        commit(15 git commit):::action
        hooks[(16 Hooks judge staged files and message)]:::system
        push(17 git push):::action
        gate[(18 Push gate reads the PR and its Plan)]:::system
        noPlan[19 Push refused without an approved Plan]:::blocked
        claudeHook[(20 Claude hook shows red checks)]:::system
    end

    subgraph r6["Req 0006 Judge text before and after it posts"]
        draft(21 Draft a comment or PR body):::action
        preJudge[(22 Draft judged before it posts)]:::system
        verdict[23 Bot posts one verdict]:::screen
    end

    subgraph r7["Req 0007 Run fitness on GitHub"]
        actions[(24 Actions judge the changes, then a full scan)]:::system
        merge[(25 Auto-merge merges or says why it waits)]:::system
        issues[(26 Finished work closes its issues)]:::system
    end

    subgraph r8["Req 0008 Release fitness"]
        releasePR[(27 Release PR from version.txt)]:::system
        tagged[28 Tagged release with checked archives and notes]:::screen
    end

    dev --> install
    install --> verify
    verify --> link
    link --> pins
    pins -->|A| download
    pins -->|B| cached
    download --> run
    cached --> run
    run --> results
    results -->|C| red
    red --> explain
    explain --> run
    results -->|D| commit
    results -->|E| choose
    choose --> config
    config --> run
    choose -->|F| orgRule
    orgRule --> choose
    commit --> hooks
    hooks -->|G| red
    hooks --> push
    push --> gate
    gate -->|H| noPlan
    noPlan --> draft
    gate -->|I| actions
    dev -->|J| claudeHook
    claudeHook --> red
    draft --> preJudge
    preJudge -->|K| draft
    preJudge --> verdict
    verdict -->|L| draft
    verdict -->|M| merge
    actions --> merge
    merge --> issues
    merge --> releasePR
    releasePR --> tagged
    tagged -->|N| install

    classDef persona fill:#eef2ff,stroke:#6366f1,color:#312e81,stroke-width:2px
    classDef screen fill:#f8fafc,stroke:#64748b,color:#1e293b,stroke-width:2px
    classDef action fill:#fff7ed,stroke:#f97316,color:#9a3412,stroke-width:2px
    classDef system fill:#ecfdf5,stroke:#10b981,color:#064e3b,stroke-width:2px
    classDef blocked fill:#fef2f2,stroke:#ef4444,color:#991b1b,stroke-width:2px

    style r1 fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
    style r2 fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
    style r3 fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
    style r4 fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
    style r5 fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
    style r6 fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
    style r7 fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
    style r8 fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
```

Numbers match the callout table. System context: [01-system-context.md](01-system-context.md). Components: [03-components.md](03-components.md#resolve-check-specs-priority). Code: [04-code.md](04-code.md).

| #   | Description                                     | Type      | Why                                                                     |
| --- | ----------------------------------------------- | --------- | ----------------------------------------------------------------------- |
| 1   | Developer or AI agent.                          | Persona   | Same CLI for a person and an agent; no agent-only path.                 |
| 2   | `fitness-install`, locally or in a GitHub job.  | Action    | One installer gives the release asked for, or the latest.               |
| 3   | The installer checks the release's checksum.    | System    | An unverified release never runs; the installer says why and exits 1.   |
| 4   | `--link-skills` and `--claude-settings`.        | Action    | The agent gets every skill and each hook once, and its own stay.        |
| 5   | The repo's `checkVersions` pins.                | System    | Each run uses the checks the repo pinned, not whatever is newest.       |
| 6   | Lazy, verified check downloads.                 | System    | A run on a few files fetches only the checks those files reach.         |
| 7   | Cached checks when there is no network.         | System    | Work goes on offline; each uncached check says it is offline.           |
| 8   | `fitness`, `fitness <folder>` or `--blob`.      | Action    | The most common entry point, scoped to what the person touched.         |
| 9   | Green, red or info per check, with skips named. | Screen    | The exit code says whether the bar holds; JSON serves agents.           |
| 10  | A failing check's findings.                     | Blocked   | Each finding names the file, the line and the fix.                      |
| 11  | `fitness explain <check>`, then an edit.        | Action    | The binary explains why the check exists before the person reruns.      |
| 12  | A setting or a repo-local check.                | Action    | A repo picks its checks and adds its own.                               |
| 13  | `fitness config`.                               | Screen    | Shows each setting's source and why each check runs or skips.           |
| 14  | A setting that drops an org check.              | Blocked   | The org bar stays whole; the run names the org rule.                    |
| 15  | `git commit`.                                   | Action    | The hooks the repo ships run on every commit.                           |
| 16  | Pre-commit and commit-msg hooks.                | System    | A change or message that breaks a check never becomes a commit.         |
| 17  | `git push`.                                     | Action    | The pre-push hook checks each pushed branch's PR once.                  |
| 18  | The push gate.                                  | System    | PR and Plan findings show before Actions spends a run.                  |
| 19  | A push whose PR closes no approved Plan.        | Blocked   | Work traces to a Plan the owner approved.                               |
| 20  | `fitness claude-hook`.                          | System    | Claude meets red checks as it edits, stops or starts, and fixes them.   |
| 21  | A comment, PR body or issue body.               | Action    | Text is held to rules like code.                                        |
| 22  | `comment-check --blob` or the Claude post hook. | System    | A red draft is denied before it reaches GitHub.                         |
| 23  | comment-check, plan-check, issue-check, pr-check. | Screen  | Each post shows one current verdict that says what to fix.              |
| 24  | The shared workflows on the right runner.       | System    | Changed files are judged first, then everything else.                   |
| 25  | Auto-merge and Release approve.                 | System    | A ready PR or stack merges with no hand click.                          |
| 26  | Parent and child issues close together.         | System    | An issue stays open while work remains, and closes when it is done.     |
| 27  | Release Please and the release tool.            | System    | The version comes from `version.txt`, verified.                         |
| 28  | Archives with hashes, notes and moved major tags. | Screen  | Users install a release whose files and notes match the repo.           |
| A   | Online: the pins are not cached yet.            | Connector | Downloads stay small.                                                   |
| B   | Offline.                                        | Connector | The cache stands in for GitHub.                                         |
| C   | A check is red.                                 | Connector | The person sees what broke.                                             |
| D   | Every check is green.                           | Connector | Green work moves on to a commit.                                        |
| E   | A check does not fit the repo.                  | Connector | The repo tunes its checks instead of ignoring red.                      |
| F   | The setting turns off an org check.             | Connector | Org policy wins over a repo setting.                                    |
| G   | The hooks refuse the commit.                    | Connector | The refusal shows the same red rows as a run.                           |
| H   | No approved Plan.                               | Connector | The fix is a PR body that closes an approved Plan.                      |
| I   | The push goes through.                          | Connector | GitHub takes over the judging.                                          |
| J   | Claude edits, stops or starts a session.        | Connector | The hook judges Claude's work in the same turn.                         |
| K   | The draft is red.                               | Connector | The person edits before anyone reads it.                                |
| L   | The verdict asks for a fix.                     | Connector | The verdict updates once the text is fixed.                             |
| M   | Every verdict passes.                           | Connector | A ready PR goes to merge.                                               |
| N   | The next install fetches the new release.       | Connector | The loop closes: a release reaches every repo.                          |
