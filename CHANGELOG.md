---
# Changed package version should correlate with this file
relatedConfigurations: ['.fitnessrc.json']
---

# Changelog

## Changes

### 2026.10.09.1920

- Fix: release-changelog: finds the release section under Release Please's linked version heading.
- Fix: release-changelog: a release PR no longer fails as missing its own release section.
- Test: acceptance 0064.7 runs the real check on a Release Please heading.

### 2026.10.09.1632

- Chore: make requirements-coverage builds with coverage, runs the suite, and prints coverage by package and in total.
- Docs: the add-requirements skill points at requirements-coverage and names May Photos as the Swift pilot.
- Docs: the skill says to convert every component before turning the check on, and links the doc conventions.


### 2026.10.09.1613

- Feat: changelog: a release section may hold Release Please's Features and Bug Fixes subheadings and star bullets.
- Feat: markdown-no-bold-italic: allows the bold scope Release Please gives each release bullet, and no other bold.
- Docs: requirement 0064 states how release notes pass the changelog checks untouched.

### 2026.10.09.1605

- Docs: skills/add-requirements/SKILL.md teaches an agent to add stated requirements proven by real-run tests.
- Docs: the skill covers requirement docs, one test per acceptance, saved answers for the outside world, and coverage.
- Docs: it links this repo's requirement docs and tests as the worked example.

### 2026.10.09.1533

- Perf: install: a cached bundle starts in about 0.1 s instead of 0.75 s; installed files are checked against hashes recorded at install.
- Perf: install: every hook run and fitness-install call skips re-reading and unpacking the cached archive.
- Test: requirement tests run 32 at once, so the whole suite takes about 15 s locally.

### 2026.10.09.1514

- Test: GitHub, npm registry, eslint, prettier, vitest, and swiftlint answers are saved; no requirement test leaves the machine.
- Test: make requirements builds the candidate and runs the whole suite offline in about a minute.
- Ci: smoke jobs no longer install real tools or need the sandbox token, and both run every test.
- Test: dependency-currency now proves registries that answer an error or garbage, not just unreachable ones.

### 2026.10.09.1508

- Test: tests/requirements.sh builds this machine's candidate and runs the suite with one test per core.
- Test: the test binary stands in for gh, eslint, prettier, vitest, and swiftlint, answering from saved responses.
- Test: a local server stands in for the npm registry, so no requirement test leaves the machine.
- Docs: the shared cspell words list goreleaser, the release build tool.

### 2026.10.09.1453

- Fix: the release workflow passes its secrets to the binaries job, so release smoke tests can reach GitHub.
- Fix: release smoke jobs read Plans and write to the sandbox with `FITNESS_SANDBOX_TOKEN`, as pull request smoke does.
- Fix: a release no longer stalls as a draft because smoke tests ran without a token.
- Fix: changelog-bullets ends an entry at a ## release heading and allows releases up to the prose-budget list limit.

## 1.1.0

- Feat: checks: new `issue-checklist` check; plan-check, pr-check, and close-check judge bodies with checks ([#222](https://github.com/may-journal/fitness-runner/issues/222)) ([01c64e6](https://github.com/may-journal/fitness-runner/commit/01c64e6603872b9fbfe23cafd4897f5b71971f80))
- Feat: hooks: trace pushes to branch Plans and prove every hook path ([#221](https://github.com/may-journal/fitness-runner/issues/221)) ([607bf0a](https://github.com/may-journal/fitness-runner/commit/607bf0a9fac7c5dbe8d5b28dfd9d638082e34453))
- Feat: requirements: own every Go test with a stated requirement ([#204](https://github.com/may-journal/fitness-runner/issues/204)) ([c6ba991](https://github.com/may-journal/fitness-runner/commit/c6ba991cd1e6376aa7644ba3f6b46ea2c5c5c0df))
- Fix: prettier: name unformatted files in CI, where Prettier colors its output ([#230](https://github.com/may-journal/fitness-runner/issues/230)) ([d4b529e](https://github.com/may-journal/fitness-runner/commit/d4b529e51fba47c6aa77346c317481a4b2261632))
- Fix: hooks: `fitness init` and `fitness hook` outside a git repo say why ([#221](https://github.com/may-journal/fitness-runner/issues/221)) ([607bf0a](https://github.com/may-journal/fitness-runner/commit/607bf0a9fac7c5dbe8d5b28dfd9d638082e34453))
- Test: requirements: every check and tool is proven by real runs, in parallel; 226 unit tests removed ([#226](https://github.com/may-journal/fitness-runner/issues/226)) ([6fb1169](https://github.com/may-journal/fitness-runner/commit/6fb1169c0d5a1967d6fd39a3d4a6a64536b65124))
- Ci: workflows: cut billed minutes and drop the macOS runners ([#202](https://github.com/may-journal/fitness-runner/issues/202)) ([2dff043](https://github.com/may-journal/fitness-runner/commit/2dff043d15678652e0efac0e3fb8db57843f6045))
- Docs: installation: centralize the verified latest setup ([#198](https://github.com/may-journal/fitness-runner/issues/198)) ([fb84ddd](https://github.com/may-journal/fitness-runner/commit/fb84dddca754d21d6cc0f8ad70aa530c668ce37a))

### 2026.10.09.1428

- Docs: the 1.1.0 notes list the new issue-checklist check and the GitHub command rebuild.
- Docs: the 1.1.0 notes list the prettier fix for colored CI output.
- Docs: the 1.1.0 notes list the test, CI, and docs work since v1.0.1.

### 2026.10.09.1405

- Test: requirement tests run in parallel, so the suite takes about 2.5 minutes locally instead of 8.
- Test: missing-tool tests drop the tool with pathWithout, since Linux runners keep Go and Node in /usr/bin.
- Fix: prettier names unformatted files in CI too, where it colors output; failures show its last lines.
- Test: retire 0057.7; SwiftLint on Linux passes a broken config, which is logged as a bug.

### 2026.10.09.1334

- Docs: requirements 0016 to 0063 state every remaining check's rules and the install, release, and stamp tools.
- Test: each acceptance runs the installed check or tool on a real repo; 226 in-process unit tests are deleted.
- Test: eslint, prettier, vitest, and swiftlint requirement tests run the real tools, pinned in tests/tools/install.sh.
- Ci: the smoke job installs those real tools; each test skips first when no candidate installer is set.

### 2026.10.09.1204

- Feat: issue-checklist check fails an issue body with unchecked items; close-check and pr-check both run it.
- Refactor: plan-check, pr-check, and close-check are declared as checks on a GitHub blob plus an after-effect.
- Refactor: after-effects in internal/aftereffect own every GitHub write: verdicts, labels, hiding, and reopening.
- Test: sandbox tests 0013 to 0015 run in parallel, in the amd64 smoke job only, on branches keyed to source and platform.
- Docs: protocol and catalog describe blob input; engine, git hooks, push plan, and issue requirement files gain prefixes.

### 2026.10.08.2343

- Feat: pre-push traces a push only to Plan trailers on the branch; it no longer reads the open PR body.
- Fix: fitness init and fitness hook outside a git repo say why, and init writes no hooks there.
- Docs: requirements 0010 and 0011 state the push Plan gate and amend rule; 0009 is renamed checks-git-hooks.
- Test: requirement tests prove hook errors, the Go hook, Plan approval, and amends with real git and gh.
- Test: delete the pre-push and amend unit tests that requirements 0010 and 0011 now prove.

### 2026.10.08.1651

- Docs: requirement 0009 states what the git hooks refuse, from fitness init to the Go pre-commit installer.
- Test: each 0009 acceptance installs the hooks with the candidate and runs real git commits and pushes.
- Test: pushes in requirement tests go to a local bare remote, so no test needs GitHub.
- Test: delete the fitness init unit tests that requirement 0009.1 now proves.

### 2026.10.08.1619

- Feat: requirements check holds docs/requirements to one template with never-reused IDs; Go test Test0001_1 owns acceptance 0001.1.
- Feat: retire disabledChecks, --check=, --jobs, legacy JS config detection, and PATH lookup; name a check positionally.
- Fix: a stopped check's leftover tools are killed, and external mode applies the configured timeoutMs.
- Test: requirement tests run the fitness-install one-liner in tests/examples repos; the smoke job runs them.
- Docs: requirements 0001 to 0008 state runner status, errors, annotations, and the two checks they cover.

### 2026.10.06.0014

- Ci: fitness-suite drops the detect job and skips fitness-runner with a job condition, saving one billed minute per run.
- Ci: ci-reusable installs swiftlint when hashFiles finds Swift sources, even without the swift input.
- Ci: every shared workflow job sets timeout-minutes, so a hung job cannot bill for six hours.
- Ci: plan-check uses a per-issue concurrency group with cancel-in-progress, so edit bursts run once.
- Docs: the workflows README describes the fitness-suite job condition and Swift detection.

### 2026.10.05.1923

- Ci: drop the macOS runners from the distribution smoke matrix, so each PR run finishes faster.
- Ci: drop the macOS runners from the release verify matrix; only Linux hosts verify public downloads.
- Docs: the fitness-release README states that darwin tarballs are cross-compiled but not smoke-tested in CI.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1955

- Docs: centralize verified latest installer setup so consumer repositories can link to one maintained guide.
- Docs: support a custom installation directory with a stable executable path across releases.
- Docs: explain that latest release lookup needs network access even when its bundle is cached.

## 1.0.1

- Fix: installer: expose verified bundle tools to hooks ([#193](https://github.com/may-journal/fitness-runner/issues/193)) ([fc22abc](https://github.com/may-journal/fitness-runner/commit/fc22abc5ef7cdba5cde213ca1a3a3dca1f54e1ac))


## 1.0.0 (2026-10-04)

- Feat: coverage: check Go entry coverage and test overlap ([#182](https://github.com/may-journal/fitness-runner/issues/182)) ([98aba91](https://github.com/may-journal/fitness-runner/commit/98aba9109aedddf82d12d13ef5c932fc4aa698b7))
- Feat: release: adopt Release Please and GoReleaser ([#184](https://github.com/may-journal/fitness-runner/issues/184)) ([8904789](https://github.com/may-journal/fitness-runner/commit/8904789a5ecaae79dff784a32a8efe9ffa5a6212))
- Feat: release: require reviewable version notes ([#188](https://github.com/may-journal/fitness-runner/issues/188)) ([43933bb](https://github.com/may-journal/fitness-runner/commit/43933bb12805f8b7a22ffbfd6ea786916c4798fa))

### 2026.10.04.1527

- Feat: correlate Release Please versions with reviewable changelog release sections.
- Fix: reject manifest-only releases, mismatched version files, and empty or duplicate release notes.
- Refactor: make release check availability explicit and unit-tested.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1440

- Fix: reject named test helpers passed through `t.Run` so wrappers cannot hide independently measured tests.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: retain exact and partial overlap as review signals instead of treating path equality as source duplication.

### 2026.10.04.1357

- Feat: adopt Release Please for reviewed release versions, tags, and draft notes.
- Feat: use GoReleaser for binary builds, archives, checksums, and verified asset publication.
- Fix: preserve legacy downloads and choose the matching installer for tagged or latest releases.
- Refactor: run GitHub release metadata and tracking issue updates through tested Go commands.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1309

- Fix: resolve directory aliases before mapping coverage profiles to source paths.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1259

- Feat: export source-line claims, per-test block sets, and pairwise shared coverage as JSON.
- Feat: retain column ranges, shared setup owners, and stable source IDs alongside existing overlap metrics.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1242

- Fix: separate tests with no measured production coverage from tests with overlapping coverage.
- Feat: show covered block totals alongside each test group's unique contribution.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1230

- Fix: reject mismatched child coverage and flag changing setup coverage in test overlap reports.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: explain how repeated setup runs affect overlap reports.

### 2026.10.04.1226

- Feat: an optional Go coverage check with full entry coverage and file, package, and folder reports.
- Feat: test overlap reports and measured coverage from compiled child tools.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1159

- Feat: report all workflow outcomes through native annotations, summaries, and complete overflow artifacts.
- Fix: cover runner setup, issue checks, installer failures, and release commands while preserving exit codes.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Fix: isolate report artifacts and failure markers across repeated workflow and action calls.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1140

- Fix: speed up spelling scans by matching ignore patterns only on lines with the required markers.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1119

- Docs: require native GitHub annotations and job summaries across all workflows in ADR 0003.
- Docs: link the reporting contract from architecture and workflow guides.
- Docs: record reporting gaps, output limits, and checks needed for workflow adoption.

### 2026.10.04.1106

- Fix: remove the check filter from release action verification so it runs the full suite.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: remove the pending-publication note now that the full-suite release is verified.

### 2026.10.04.1050

- Fix: run every check by default in external mode and let each check decide whether it applies.
- Docs: make full-suite runs the main Jenkins, Actions, and compiled hook examples.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1040

- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: add full external-mode examples for Jenkins, Actions, and compiled commit hooks.
- Feat: bind external action inputs to the new Go release and verify chosen checks during publishing.

### 2026.10.04.1036

- Feat: add external policy with exact check lists from CLI flags or repo config.
- Feat: pass external policy and check lists through the compiled Go action entry point.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.1019

- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.0951

- Fix: pin the action and CI examples to the release with compiled Go hooks.
- Docs: align direct installer URLs with the compiled entry points.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.04.0945

- Refactor: move release packaging, tag checks, publishing, and native smoke checks from shell scripts into typed Go.
- Feat: install a compiled Go pre-commit hook and handle action inputs and outputs in Go.
- Docs: download native installers directly from release URLs; remove the shell launcher and its examples.

### 2026.10.04.0918

- Feat: move download, archive, and cache logic into a typed Go installer compiled for each host.
- Fix: switch cache copies with an atomic pointer so a repair leaves running checks intact.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Ci: test public release URLs and the root action on four hosts before completing the release run.
- Style: trim whitespace after the name and separator fields in the printf line.

### 2026.10.04.0900

- Ci: test real release bundles on all four hosts before publishing; keep explicit source refs for shared workflow callers.
- Docs: add full shell, hook, Jenkins, and Actions setup steps with no local installer to maintain.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Fix: keep gzip on the test path for Linux tar; test the root action with the packaged installer on each host.

### 2026.10.04.0850

- Feat: install pinned binary bundles through a shell entry point or root action, with verified downloads and cache repair.
- Ci: publish the shell installer with each release and run shared file checks from binaries.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.01.2331

- Feat: every check judges every tracked file through `git ls-files`; only untracked files, symlinks, and `linguist-generated` files for cspell and jscpd stay out.
- Feat: a repo setting `ignore`, `skipTheseDirectories`, `proseBudget.exempt`, cspell `ignorePaths`, or `.prettierignore` fails instead of skipping files.
- Feat: prose-budget caps prose words per section instead of per file; older `CHANGELOG.md` bullets are reworded to fit.
- Feat: tests, Markdown, and JSON lose their exemptions, and `markdown-front-matter` reads front matter inside an HTML comment.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.10.01.2305

- Fix: the prettier check runs Prettier with `--ignore-unknown`, so a changed file it has no parser for, such as Swift, passes.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: the prettier README says a file type with no parser passes through `--ignore-unknown`.

### 2026.10.01.1602

- Ci: `ci-reusable` sets up Node from `.nvmrc` and runs `npm ci` when a repo has them, so Node repos run their own tools.
- Ci: SwiftLint is pinned to 0.65.1 on macOS and Linux through a `swiftlint-version` input, instead of whatever is latest.
- Docs: the adoption guide says how Node repos are set up and how to move SwiftLint forward.

### 2026.09.30.2301

- Fix: the prettier check puts the repo's `node_modules` folders on `NODE_PATH`, so the shared config loads the plugins the repo installs.
- Fix: when Prettier fails without naming a file, the check reports its own `[error]` lines, such as a plugin it cannot load.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: the prettier README says which two plugins a repo installs.

### 2026.09.30.1928

- Fix: `pr-closes-issue` skips closing keywords inside inline or fenced code, since they quote an example rather than close an issue.
- Fix: a Plan quoting another repo's `Closes #51` no longer requires its PR to close this repo's #51.
- Refactor: the checklist parser reuses the shared `mdx` fence detection.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: note the code rule in the `pr-closes-issue` README.

### 2026.09.30.1927

- Feat: `fitness close-check` reopens an issue closed as completed with unchecked items, unless closed as not planned or duplicate.
- Feat: it comments once, listing the items and mentioning the closer, plus the closing PR's author and merger.
- Feat: `fitness pr-check` fails a PR whose closing keyword targets an issue with unchecked items, naming each item.
- Ci: publish `close-check-reusable.yml`, and run close-check on this repo's own issues.
- Docs: describe the rule and caller in the adoption guide and READMEs; tests cover each case with a fake GitHub API.

### 2026.09.30.1916

- Ci: `auto-merge.yml` acts as the `may-journal-automation` GitHub App, minting a one-hour token, instead of the bot's `AUTO_MERGE_TOKEN`.
- Ci: it fails with a clear error when `AUTOMATION_APP_ID` or `AUTOMATION_APP_KEY` is missing.
- Docs: the workflows README and adoption guide name the App and its two org secrets.

### 2026.09.30.1847

- Feat: on a pull request in CI, `issue-link-once` judges every branch commit and fails a repeat link, in every repo.
- Feat: a commit authored after the PR opened fails when it links an Issue the PR body links.
- Feat: each failure names the commit and says to rewrite it, then force-push with `--force-with-lease`.
- Docs: the check README and `AGENTS.md` describe the CI rule and its fix.

### 2026.09.30.1830

- Fix: a draft pull request gets auto-merge once it is marked ready, through a new `auto-merge-on-ready.yml` caller on `ready_for_review`.
- Fix: `auto-merge.yml` also runs as a `workflow_call`, so the caller reuses it.
- Ci: `auto-merge.yml` lists only the events its ruleset starts it on, since GitHub refuses auto-merge on a draft.
- Docs: the workflows README and adoption guide explain the draft gap and show the caller each repo adds.

### 2026.09.30.1803

- Feat: `issue-link-once`, run by the `commit-msg` hook, fails a commit linking an Issue the branch or its open PR already links.
- Feat: `pre-push` finds the approved Plan in any branch commit or the open PR, so later pushes need no trailer.
- Fix: `pre-push` reads only the pushed commits, not their ancestors, so a trailer on `main` no longer passes the gate.
- Chore: this repo's own `githooks` call the Go hooks, so both share one implementation.
- Docs: `AGENTS.md`, the README, and the `plan-trailer` README put the trailer on the branch's first commit only.

### 2026.09.30.2238

- Feat: `fitness plan-check` edits its passing verdict to `✅ Validated (updated <UTC time>)` when a new body passes too, instead of posting another comment.
- Feat: it hides every older plan-check verdict on the Issue as outdated.
- Feat: it labels each checked Issue `fitness` plus `fitness-valid` or `fitness-invalid`, creating missing labels.
- Feat: each verdict links the fitness-runner commit that judged it and the workflow run.
- Docs: the workflows README and step names describe the single verdict and labels.

### 2026.09.30.2100

- Fix: `fitness-suite` runs in every repo except fitness-runner, even one with its own `ci-reusable` caller.
- Fix: so the org ruleset enforces the suite everywhere, with no gap while per-repo callers are removed.
- Docs: the workflows README says only fitness-runner skips it.

### 2026.09.30.1520

- Feat: add `fitness-suite.yml`, which an org ruleset can inject on every pull request to run the fitness suite through `ci-reusable`.
- Feat: `fitness-suite.yml` installs swiftlint when the repo has Swift sources, since an injected workflow gets no inputs.
- Fix: it skips fitness-runner and any repo whose own workflow already calls `ci-reusable`, so no suite runs twice.
- Docs: describe `fitness-suite` in `.github/workflows/README.md`.

### 2026.09.28.2300

- Fix: `ci-reusable` installs the official Linux swiftlint build on Linux runners, so Swift repos can drop `runs-on: macos-latest`.
- Fix: macOS runners still install swiftlint with brew, so current callers keep working.
- Fix: give `swiftlint` a 120-second budget, since a cold lint on a Linux runner outlasts the 5-second default.
- Docs: the adoption guide's Swift example runs on the default ubuntu runner, which costs about a tenth of macOS.

### 2026.09.28.2234

- Fix: `changelog-updated` accepts a newest heading up to five minutes behind the clock, so a stamp just before a minute boundary passes.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: the `changelog-updated` README describes the five-minute stamp grace.

### 2026.09.28.2140

- Fix: `changelog-updated` requires the current time only on the newest added section heading, so a changelog rewrite that re-adds older headings passes.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: the `changelog-updated` README says only the newest added heading must use the current time, in shorter sentences.

### 2026.09.28.1440

- Perf: pre-commit checks only the staged files, and a pull request only the files it changes against its base branch.
- Perf: a push to `main`, or `fitness --all`, still checks every file, so the full suite runs after merge.
- Perf: `go-test` and `go-vet` skip when no Go file changed, and `eslint` and `prettier` skip when nothing they lint changed.
- Perf: pre-commit no longer runs `make check`, and CI drops its `go` job, since the Go checks already cover both.
- Docs: describe changed-file scoping in the check protocol, adoption guide, and check READMEs.

### 2026.09.28.1408

- Docs: the adoption guide says an org ruleset makes the fitness suite a required check.
- Docs: the caller job reports `fitness / fitness`, the status check name the ruleset requires.
- Docs: a new repo must join `fitness-suite-required.json` in may-journal/.github to be required.

### 2026.09.28.1350

- Fix: `changelog-bullets` accepts a semantic commit subject such as `feat(release): `, alongside the capitalized `Feat: ` prefix.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: describe both accepted prefix forms in the `changelog-bullets` README.

### 2026.09.28.1323

- Ci: host the org Auto-merge workflow here, since public repos cannot run workflows from the private may-journal/.github repo.
- Ci: it turns on squash auto-merge for ready pull requests with the bot's token, so merges still start CI on `main`.
- Docs: describe it in the workflows README and the adoption doc's org-wide section.

### 2026.09.28.1257

- Feat: every check runs in every repo; config keeps only `disabledChecks`, `ignore`, and options, and local path checks are gone.
- Feat: add `go-vet`, `go-test`, and `gofmt` checks, plus an `ignore` list of gitignore-like globs every file check skips.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Feat: `commit-attribution` applies once a repo uses its trailers, filename checks follow the detected case, and `build-output-untracked` needs TypeScript or `dist`.
- Fix: `make` and the `go-test` check drop git's hook variables, so tests run from a hook cannot write into the real repository.

### 2026.09.28.1222

- Fix: `markdown-links` decodes a percent-encoded link path, so a link like `my%20file.md` resolves to the file it names.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: note the decoding in the `markdown-links` README.

### 2026.09.28.1156

- Fix: `fitness plan-check` reads a Plan's existing comments, so a Plan that already has a comment gets a verdict again.
- Fix: every gh query in the workflow subcommands emits JSON objects, since gh prints a bare string result raw.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.09.28.1151

- Fix: the shared workflows install fitness with `GOPROXY=direct`, so `@main` is the latest merge rather than a stale proxy copy.
- Chore: add `goproxy` to the Go spelling dictionary.
- Docs: note in the adoption guide why the shared install skips the Go proxy.

### 2026.09.28.1055

- Feat: add `fitness pr-check`, which validates a PR's title and description and writes the verdict to the job summary and log.
- Feat: add `fitness plan-check`, which validates a Plan issue and comments the result once per body version.
- Refactor: replace the inline JavaScript in the plan-check and pr-check workflows, and their reusable twins, with one step calling `fitness`.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: document both subcommands in the READMEs, and add the runner's subcommands to the architecture docs.

### 2026.09.28.0919

- Fix: `fitness hook pre-push` gates only commits not yet on origin, so merging main into a branch no longer blocks on main's own commits.
- Fix: the pre-push gate exempts merge commits, like chore and docs; the commits a merge brings in are gated on their own.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.09.27.1429

- Feat: add `fitness hook commit-msg`, `pre-commit`, and `pre-push` subcommands that own the hook orchestration in Go.
- Feat: port the eighty-line Bash plan-approval gate into `fitness hook pre-push`; the embedded hooks are now one-line shims that exec the subcommand.
- Feat: `fitness init` installs the shims, so a repo's hooks stay one line while the logic lives in the tested binary.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: document `fitness init` and `fitness hook` in the README, and add a `fitness help` command that prints the usage.

### 2026.09.27.1341

- Feat: add `fitness init`, which installs the shared commit-msg, pre-commit, and pre-push hooks into `.githooks` and sets `core.hooksPath`, so repos pull them by version.
- Feat: embed the canonical consumer hooks in the module; they call `fitness`, and pre-commit runs a repo's `.githooks/pre-commit.local` for its build and tests.
- Feat: publish a reusable `ci-reusable.yml` that owns the file-suite build, so a repo's `ci.yml` is a thin caller.
- Feat: `ci-reusable.yml` covers the runner OS, Go setup, the `fitness` install, and swiftlint.
- Docs: document `fitness init` and the reusable CI caller in `docs/adoption.md`.

### 2026.09.27.1306

- Feat: both reporters lead with a headline of checks passed of total, files scanned, and total time, confirming an all-green run.
- Feat: the terminal total line reads "✓ All N checks passed · F files scanned · Tms" on green, and names passed and failed counts otherwise.
- Feat: tighten the terminal table to one rule under the header and contiguous rows, so passes stay compact and failures stand out.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.09.27.1224

- Feat: `fitness` writes a GitHub Actions job summary: a check, status, files, and time table plus each failure's errors, readable without the logs.
- Feat: emit `::error` annotations, parsing file and line from each error, capped at ten per step; the summary notes any more.
- Feat: gate the CI output behind the `GITHUB_ACTIONS` env, so a local run's terminal output is unchanged.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.09.27.1155

- Feat: add the `no-contrastive-reframing` check; it flags markdown prose that rejects a claim, then restates the point: "not X, it's Y".
- Feat: match the split form ("It's not a workout. It's a lifestyle.") and the single-sentence form, gated on a demonstrative opening for precision.
- Feat: reuse the `mdx` prose masking and drop quoted spans, so fenced code, headings, inline code, and quoted examples never trip it.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.09.27.1116

- Feat: add the `doc-template` check: a `template.md` or `*.template.md` defines a document's sections, and every markdown file in its folder tree must match.
- Feat: each template `##` section, `.github` PR and Issue templates included, needs a guiding HTML comment; the nearest template above a file wins.
- Feat: add `mdtemplate` `SpecFromTemplate`, `MissingCommentSections`, and a `NoPitch` option, so a pitch-less file reuses the section rules of Issue and PR bodies.
- Docs: adopt the shared ADR `template.md`, fold ADR 0002's supersede note from Status into Context, and list the check in `docs/checks.md`.

### 2026.09.26.2235

- Feat: `semantic-commit` accepts an explicit `--message`, ahead of the `FITNESS_CTX_MESSAGE` and HEAD fallbacks, so it can validate a subject like a PR title.
- Feat: the `pr-check` workflows run `semantic-commit` on the PR title, so a non-semantic title never becomes a bad squash-merge subject on `main`.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.09.26.1912

- Feat: language checks self-gate: `eslint`, `prettier`, `no-eslint-disable`, `node-version`, vitest coverage checks, and `swiftlint` pass on zero files without `package.json` or Swift.
- Feat: the runner's default is the full self-gating catalog, so Swift, Go, JS, and docs repos share one list without a `.fitnessrc.json`.
- Feat: the strict, network, format-specific, and workflow-body checks stay opt-in.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: note self-gating and the new default in `docs/checks.md`.

### 2026.09.26.1834

- Fix: default the reusable workflows to install fitness at `main`, not `@latest`, so the org-injected required workflow uses the current check set.
- Fix: the latest release predated `pr-closes-issue`, `mermaid-diagram-table-gap`, and the tighter prose-budget, so an `@latest` install failed on missing binaries; `main` resolves them.
- Chore: callers can still pass a pinned `ref` input to freeze a repo on a released check set.

### 2026.09.26.1741

- Fix: give `pr-check-reusable` `pull_request` and `workflow_dispatch` triggers so an org ruleset can require it; a `workflow_call`-only workflow was rejected as a required workflow.
- Fix: fall back to `@latest` when no `inputs.ref` is passed, as on the required-workflow path.
- Fix: skip the job in fitness-runner, which runs its own build-from-source pr-check.
- Docs: note in `docs/adoption.md` that the reusable pr-check doubles as an org required workflow.

### 2026.09.26.1703

- Feat: publish reusable `pr-check` and `plan-check` workflows (`workflow_call`) that install the checks and validate a calling repo's PR or Plan bodies.
- Feat: a repo adopts them by calling the workflows instead of copying the logic.
- Docs: add `docs/adoption.md` — the reusable-workflow callers, the version policy, and the org-wide rollout path (org required workflow plus the org `.github` repo).
- Chore: fitness-runner keeps its own `pr-check` and `plan-check` built from source, so it tests its unreleased changes while others call the reusable ones.

### 2026.09.26.1611

- Feat: `pr-closes-issue` also fails a PR that closes a `Plan` but not the issue the Plan solves, which would stay open (#77).
- Feat: add `--emit-closed`, which prints a body's closing-keyword targets as JSON, and a `--require-close` flag the check verifies.
- Feat: `pr-check` reads each closed Plan's targets through `--emit-closed`, so one keyword set governs both sides.
- Docs: `AGENTS.md` says a Plan names the issue it solves with a closing keyword, and the `pr-closes-issue` README documents the third rule.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.09.26.1033

- Docs: drop the hardcoded "31 checks" count from the C4 architecture docs, naming the `fitness-check-*` set so the number stops drifting.
- Docs: the containers diagram adds `plan-check` and `pr-check` workflows running checks on a description; the components doc gains a body-mode section.
- Docs: relativize the remaining fixed check counts in `distribution.md` and `checks.md`.

### 2026.09.26.0032

- Fix: `changelog-updated` goes inert during a merge, cherry-pick, or revert, so a replay commit's old `### yyyy.mm.dd.HHMM` headings no longer fail pre-commit.
- Feat: detect replays via `MERGE_HEAD`, `CHERRY_PICK_HEAD`, or `REVERT_HEAD`, resolved with `git rev-parse --git-path` so it works when `.git` is a file, behind an injectable seam.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.09.25.2046

- Feat: lint Issue and PR descriptions beyond structure, with `prose-budget`, `text-readability`, `markdown-no-bold-italic`, the mermaid family, and `cspell` in the `plan-check` and `pr-check` workflows.
- Feat: add a `bodycheck.RunDoc` mode that runs a check's per-document rule on a `--body-file`, with config from `--root`; stdin never triggers it.
- Feat: cut `prose-budget` defaults ~25%: sentence words 30→23, paragraph sentences 5→4, section paragraphs 4→3, item words 30→23, list items 10→8, file words 400→300.
- Chore: tighten every markdown file the repo lints to the lowered budget, splitting long sentences, paragraphs, sections, and lists instead of loosening limits.
- Docs: note description linting in `docs/checks.md` and update the `prose-budget` README limit table.

### 2026.09.25.1458

- Feat: add the `pr-closes-issue` check: every PR must close an issue on merge with a GitHub closing keyword (`close`/`fix`/`resolve`, any tense).
- Feat: fail a PR that only references issues (`addresses`, `part of`, a bare `#NN`) or names none, with no chore or docs exemption.
- Feat: also fail any `Implements #NN` or `Plan #NN` the body does not close.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: wire the check into `.github/workflows/pr-check.yml` beside `pr-structure`, add its README, and list it in `docs/checks.md`.

### 2026.09.25.1434

- Feat: add the `mermaid-diagram-table-gap` check; it flags loose prose between a numbered mermaid diagram or its legend and the callout table after it.
- Feat: allow one caption line (opens with `Numbers`, states they `match the callout table`); prose before the diagram or after the table is left alone.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: add the check README, bump the mermaid family count in `docs/checks.md` and the `internal/mermaid` doc, and enable the check in `.fitnessrc.json`.

### 2026.09.24.1558

- Feat: add the `no-plans-dir` check: any file under `docs/plans/` fails, guarding the plans-to-Issues migration; pre-commit catches a stray plan file.
- Feat: add the `plan-trailer` check: an optional commit trailer referencing a plan must read exactly `Plan #<number>`; the commit-msg hook runs it.
- Feat: add a `pre-push` hook: a push needs commits tracing to an approved `Plan` Issue via `gh`, except chore and docs-only pushes.
- Docs: document the hook set in the README Git hooks section and `AGENTS.md`, and add the two checks to the catalog.

### 2026.09.24.1517

- Chore: relax the plan approval gate: `AGENTS.md` accepts any clear approval comment (`I approve this plan`, `Approved`, or one containing `approve`), not just the exact phrase.
- Docs: spell out the accepted approval forms in the Approval gate section so the rule is unambiguous for agents and humans.
- Docs: align the pending git-hooks plan (issue #63) to enforce the same looser, case-insensitive approval match.

### 2026.09.24.1456

- Feat: add the `prose-budget` check, a hard-cap brevity linter for markdown prose that masks front matter, fenced code, tables, and headings.
- Feat: its limits on sentence and item words, paragraph sentences, section paragraphs, list items, and file words are each overridable in `.fitnessrc.json`.
- Refactor: move prose extraction, word count, sentence splitter, and inline masking into `go/internal/mdx` (`Prose`, `WordCount`, `Sentences`, `MaskInline`, `StripFrontMatter`); `text-readability` builds on them unchanged.
- Chore: enable `prose-budget` on this repo with only the built-in `CHANGELOG.md` exemption, holding every markdown file to the budget.
- Docs: split the top-level README and the prose-cognitive-complexity essay into focused docs under `docs/`, and tighten wordy check READMEs, so the tree conforms.

### 2026.09.24.0000

- Feat: add the `pr-structure` check: a PR description must match the template (a blockquote summary, `## Background`, `## Changelog` with a bullet, nothing else).
- Feat: add `.github/PULL_REQUEST_TEMPLATE.md` and a `pr-check.yml` that validates on `pull_request`, sweeps open PRs on `workflow_dispatch`, and reports via status check and Step Summary.
- Refactor: move template validation to `go/internal/mdtemplate` and body reading (stdin, `--body-file`, `--body`; none passes) to `go/internal/bodycheck`, shared with `plan-structure`.
- Chore: exempt `.github/PULL_REQUEST_TEMPLATE.md` from `markdown-front-matter`, since GitHub inserts the template into every PR body verbatim and it cannot carry front matter.
- Docs: document `pr-check` in the workflows README with a mermaid flow, and note PR-description validation in the top README.

### 2026.09.23.2217

- Chore: delete `.github/scripts/update-tap.sh` — the repo no longer generates or pushes a Homebrew formula.
- Chore: remove the `tap` job from `.github/workflows/release.yml`, so a version tag publishes the `go install` source and per-platform tarballs only.
- Docs: drop the Homebrew tap from the README Distribution section and the architecture index; the tracking issue (#47) is closed as won't-do.

### 2026.09.20.2014

- Feat: add the `plan-structure` check: a `Plan`-labeled Issue body must have a one-line blockquote pitch, `## Background`, and a `## What needs to happen` checklist, nothing else.
- Feat: it reads the body from stdin, `--body-file`, or the context-inline `--body`, and passes inert with no input.
- Feat: add `.github/workflows/plan-check.yml`: validate a `Plan` Issue on `issues`, sweep open Plans on `workflow_dispatch`, comment once per body version, and fail on violations.
- Feat: add `mdx.Headings`, an ATX heading scanner that skips fenced code, and cover it plus the new check with table-driven tests.
- Docs: add `AGENTS.md`, where an agent works an Issue once a human comments "I approve this plan", plus `.github/workflows/README.md` with a `plan-check` flow.

### 2026.09.20.1941

- Docs: add a `Plan` issue template (`.github/ISSUE_TEMPLATE/plan.md`) modeled on the personal `me` repo's: a one-line pitch, `## Background`, a `## What needs to happen` checklist, nothing else.
- Chore: migrate every `docs/plans` file into a `Plan`-labeled Issue: two active plans stay open (#49, #50); eight archived plans close as completed (#51-#58).
- Chore: delete the migrated `docs/plans` tree now that plans live as Issues, pointing the `README.md` and `architecture-index.md` plan links at their issues.
- Docs: open #59 to run fitness checks as GitHub workflows, so a workflow comment validates plan Issues, as the runner did plan files.

### 2026.09.20.1417

- Feat: exempt capitalized doc basenames (README.md, LICENSE.md, AGENTS.md, `CODE_OF_CONDUCT.md`, ...) from the kebab-case and camelCase filename checks, so all-caps docs pass anywhere.
- Refactor: replace the hardcoded `allowedBasenames` map with a single all-caps basename pattern, dropping the fixed OSS-doc list in favor of one rule.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.08.25.1516

- Chore: stop tracking `go/internal/spell/.DS_Store`, a macOS Finder metadata file that a `git add -A` swept into the previous commit.
- Chore: add a `.DS_Store` rule to `.gitignore` as a bare name, excluding the artifact from every directory rather than only the repo root.
- Chore: annotate that rule with a why-comment (Finder view-state metadata, no project value) so it satisfies the `gitignore-why` check.

### 2026.08.25.1511

- Chore: finish the npm-free migration: delete the unused `generate.mjs` (the 14 committed wordlists are the sole source) and a dead `.env` npm token.
- Fix: embedded fallback configs targeted the retired `packages/runner/src` layout; vitest globs now use `src` minus index files, and eslint drops a dead ignore.
- Chore: tailor the suite to Go: a new `disabledChecks` list turns off JavaScript-only checks, `defaultChecks` drops them, and a root `Makefile` joins pre-commit.
- Fix: cspell now honors `ignorePaths` for staged files too, so committing dictionary sources under `go/internal/spell/dict` no longer fails on their fragments.
- Docs: add ADR 0002 retiring 0001's dual `fitness-shared lint` path, a camelCase check README, a Cursor rule aimed at `go/cmd`, and README fixes.

### 2026.07.19.0934

- Feat: add the `markdown-links` check: every relative link in every markdown file must resolve to a real file or directory.
- Feat: absolute URLs are skipped, so the check stays offline and deterministic; the catalog is 31 names and the dogfood suite 21 checks.
- Fix: the new check found nine broken links on arrival: two path-depth bugs from the docs consolidation, in `competition.md` and an archived plan.
- Fix: six archived-plan links to files the npm purge deleted become code spans noting the removal.
- Docs: README and architecture check counts move to 31; the check's README documents fence and code-span masking and fragment stripping.

### 2026.07.19.0915

- Docs: architecture docs catch up: check counts move from 27 to 30 across the system context, containers, and code levels.
- Docs: the monorepo layout tree gains `.github/`, `docs/`, the stamper binary, and `internal/par`.
- Fix: the containers doc's README link pointed at `docs/README.md` after the docs consolidation; it now targets `../../README.md`.
- Docs: the containers doc adds a sentence on artifact shipping via Releases and the Go module proxy.
- Docs: the architecture index links the research directory and plan 03 alongside the two archived milestone plans.

### 2026.07.19.0909

- Docs: README install instructions lead with the working channels: `go install` with pin and upgrade commands, then prebuilt release tarballs.
- Docs: tarballs come with a verified download URL, platform list, and checksum note; build-from-source moves to a contributors block.
- Docs: the Distribution section drops the pre-launch phrasing: the first two channels are live, and the Homebrew item points at issue #47.
- Chore: the documented release download URL was tested against the published release before landing; both plain and encoded tag forms serve the asset.

### 2026.07.19.0859

- Feat: the repo is public, so `go install github.com/may-journal/fitness-runner/go/cmd/...` resolves through the Go module proxy; all 32 binaries install and run from a clean environment.
- Chore: the Homebrew tap is deferred to issue #47; its release automation is in place and skips politely until the tap exists.
- Docs: plan 03 flips the public-flip and go-install verification boxes — only the tap items remain open, each annotated with the issue.

### 2026.07.19.0852

- Fix: asset names had a stray leading v because the workflow stripped only the tag's directory prefix, breaking the tap script's checksum match.
- Fix: the workflow now strips the full prefix, so asset names agree with the formula generator.
- Chore: the first release published end to end from the changelog-derived tag: both jobs green, five assets, and the darwin binary runs here.
- Docs: plan 03 pre-flight and release-automation boxes flip; the public flip and tap creation remain open.

### 2026.07.19.0844

- Fix: release versions derive from the changelog heading instead of hand-cut semver: timestamp `2026.07.19.0837` maps to tag `go/v0.20260719.837`.
- Fix: the major stays 0 because Go reserves higher majors for `/vN` module paths, and date and minute keep their ordering.
- Feat: add `.github/scripts/release-tag.sh` — prints the tag for the newest heading; the release workflow refuses any tag that does not match it.
- Docs: README and plan 03 state the one-version rule — the pre-commit stamper owns the version, releases only transcribe it.

### 2026.07.19.0837

- Feat: add the release workflow: every `go/vX.Y.Z` tag cross-compiles static tarballs for darwin and linux on both architectures, plus a checksums file.
- Feat: it publishes a GitHub Release with the newest CHANGELOG section as its notes.
- Feat: add `.github/scripts/update-tap.sh`, which pushes a Homebrew formula built from the release checksums to `may-journal/homebrew-tap`; the job skips until `TAP_PUSH_TOKEN` exists.
- Docs: add `docs/plans/03-publish.md` — the publishing milestone in the may-journals template: pre-flight history audit, release automation, brew tap, verification last.

### 2026.07.19.0821

- Docs: README gains a Distribution section: three decided channels layered on one artifact host.
- Docs: they are `go install` via the Go module proxy, GitHub Releases with per-platform tarballs, and a Homebrew tap the release workflow bumps.
- Docs: the upgrade story is stated per channel — re-run with `@latest`, grab the next release, or `brew upgrade`.
- Docs: the CHANGELOG timestamp stays the internal version; releases are semver tags in `go/vX.Y.Z` subdirectory-module form, used as plain `@vX.Y.Z`.

### 2026.07.19.0812

- Feat: add the `text-readability` check, the catalog's 30th, a document-level smoke detector scoring each markdown file with three character-based formulas.
- Feat: a file fails only when 2 of 3 formulas exceed their alarm band; files under 100 prose words are never judged.
- Feat: failures are LLM prompt fuel: per-file alarms show each formula's value and band, then guidance gives the formulas, fixes, and doc pointers.
- Chore: enable it at tightened bands (grade 15, LIX 50) via `textReadability`, keeping shipped defaults at 18/60; the dogfood suite is 20 checks.
- Docs: fix the three files the bands flagged (README.md, the research doc, the vitest-coverage-full README) by splitting sentences and fencing quoted tool output.

### 2026.07.18.1940

- Docs: the prose-complexity research now uses the archived plan documents as its experiment corpus.
- Docs: the changelog was a weak example, since bullets are notation and its one clean separator was true by construction.
- Docs: the plans result is stronger and inverted: every readability formula scores the preferred template plans as harder than the rejected free-form ones.
- Docs: one uniformly written plan swings 17 grade levels from paragraph to paragraph.
- Docs: the changelog numbers stay as corroboration; the implications name both house interventions (`changelog-bullets`, the plan template) as structural gates that beat formulas.

### 2026.07.18.1931

- Docs: rename the research doc to `docs/research/0001-prose-cognitive-complexity.md`.
- Docs: research docs now carry a `NNNN-` number prefix, matching the ADR convention (`docs/architecture/adr/0001-...`), so they order by arrival.
- Chore: a pure `git mv` — content, front matter, and relative paths are unchanged.

### 2026.07.18.1930

- Docs: add `docs/research/prose-cognitive-complexity.md`, asking whether a check can judge the cognitive complexity of paragraphs, and surveying readability formulas, cognitive-science measures, and CI tooling.
- Docs: the research tests this repo's own changelog rewrite: no classical formula separates tight bullets from essay bullets one bullet at a time.
- Docs: only whole-section scoring and the existing 365-character cap discriminate cleanly.
- Docs: verdict for a future check: structural budgets and character-based formulas at document scale behind a frozen code-masking spec; sentence-connection measures are unexplored.

### 2026.07.18.1909

- Feat: `changelog-bullets` now judges every `###` section of CHANGELOG.md, not only the newest — count errors name their section.
- Docs: rewrite the entire changelog history into compliance (219 findings to zero), keeping every heading byte-identical and facts and issue refs intact.
- Docs: essays split into typed bullets, and thin sections fill from their commits' real diffs.
- Docs: the check README now documents whole-file semantics.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.07.18.1848

- Feat: add the `changelog-bullets` check — the newest CHANGELOG section must be 3-5 bullets, each under 365 characters, each with a semantic type prefix.
- Chore: enable it in this repo's dogfood list (19 checks) and add it to the catalog (29 names).
- Docs: rewrite the previous entry to comply — it was one 1,230-character bullet, precisely the style this check exists to end.

### 2026.07.18.1844

- Perf: cspell lookups binary-search the embedded sorted dictionary bytes with a `sync.Map` memo: zero startup parsing, 7.2x faster on small repos, ~30ms here.
- Perf: jscpd hashes tokens directly (FNV-1a 64) without a shared intern map, so hashing runs in parallel: 2.3x faster at half the memory.
- Docs: both changes proven behavior-identical — a 216k-word differential test for the spell engine and byte-identical clone statistics on two corpora.
- Fix: the earlier "27ms process baseline" was a shell-timer artifact (node startup inside the timed window); the real baseline is ~3ms.

### 2026.07.18.1821

- Perf: parallelize file scanning inside the heavy checks — the full 18-check dogfood suite drops from ~111ms to ~68-76ms.
- Feat: add `internal/par`, a generic deterministic worker pool: results keep input order, so parallelism never changes a check's output; ordering tests pin it.
- Refactor: the pool backs `walkfs.ScanFiles`, the cspell per-file loop, jscpd's read-and-lex phase (detection stays serial on shared hash tables), and `go-complexity` parsing.
- Chore: `go-complexity` flagged the new pool's own `Map` at 6 before it could land — the clamp logic became a helper.

### 2026.07.18.1819

- Feat: add `go-complexity`, the Go era's first net-new check: the house eslint rule (`complexity: max 5`) in pure `go/ast` and `go/parser`.
- Feat: scoring matches eslint: each function starts at 1, plus one per `if`/`for`/`range`/non-default `switch` or `select` clause/`&&`/`||`; function literals score separately.
- Chore: opt-in like `swiftlint`, with `_test.go` exempt and the ceiling set by `goComplexity.max`; enabled here, scanning all 50 Go files in about 10ms.
- Refactor: burn all 78 flagged functions under the ceiling with behavior-identical helper extractions; 40 packages stay green with no test expectations touched.
- Refactor: the fixes span the runner, vitestconf and clonedetect lexers (28 and 25 at worst), spell engine, mermaid parser, and 24 check binaries.

### 2026.07.18.1749

- Docs: align every document with the Go-only, npm-free reality; all 27 check READMEs correct their era, keeping rules, error formats, and examples byte-identical.
- Docs: `.fitnessrc.js`/`.ts` snippets become `.fitnessrc.json`, `npx fitness` becomes `fitness`, "bundled" claims become peer-tool exec or native-engine truth, and TypeScript internals become Go.
- Docs: config-fallback descriptions now state the three-step resolution (repo-local, installed `@mayjournal/fitness-shared`, embedded copy materialized on demand); dead `.cursor/rules` pointers are dropped.
- Docs: the architecture docs drop the last stale claims, and `competition.md` contrasts against the static check-binary catalog.
- Docs: `architecture-index.md` leads with the two completed milestone plans; the ADR stays untouched as a dated decision record.

### 2026.07.18.1742

- Chore: consolidate all documentation under `docs/`: `architecture/`, `plans/` with its archive, `architecture-index.md`, `wardley.md`, and `competition.md` move together, keeping relative links intact.
- Fix: front matter `relatedConfigurations` paths deepen one level in 17 files and README links cross the new boundary.
- Chore: the `mermaid-level-bleed` check keeps matching level files at `docs/architecture/` thanks to its unanchored path pattern; a stale npm-era phrase in `competition.md` refreshed.

### 2026.07.18.1721

- Fix: `go build -o bin` fails when the gitignored `bin/` directory does not exist yet, breaking fresh clones, the pre-commit hook, and CI alike.
- Build: every build command — hooks, CI, README, architecture docs — now runs `mkdir -p bin` first.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.07.18.1720

- Feat: the repo is npm-free — plan 02 complete and archived; cloning and building requires exactly one tool: Go.
- Feat: `go/internal/sharedconf` embeds the shared configs, cached on demand by content key, as the fallback after local config and an installed `@mayjournal/fitness-shared`.
- Feat: the Go `fitness-stamp-changelog` restamps the first heading of a staged CHANGELOG.md and re-stages it; the timestamp is now the version.
- Chore: deleted package.json, package-lock.json, `node_modules`, .npmrc, .nvmrc, node scripts, npm publish workflows, the node CI action, and `@mayjournal/fitness-shared`; CI is two Go jobs.
- Chore: spell dictionaries are frozen committed data; the dogfood list drops eslint, prettier, node-version, and dependency-currency, all in the catalog, leaving 17 checks.

### 2026.07.18.1556

- Feat: retire the TypeScript implementation — the Go suite is now the only fitness runner.
- Chore: deleted `packages/runner`, `packages/checks` (26 TypeScript checks), `packages/checks-bundle`, the legacy `.fitnessrc.js`, and the go-parity harness (26/27 byte-parity proven first).
- Docs: each check's rule docs moved to `go/cmd/fitness-check-<name>/README.md`, the `read-repo-first` banner points there, and README and the C4 docs are rewritten for Go.
- Refactor: `packages/shared` survives as a configs-only npm package of eslint, prettier, vitest, and cspell data; build machinery, bin scripts, and runtime sources go.
- Chore: scripts slim to `npm run fitness` and `npm test`; CI drops fitness-ts and go-parity; devDependencies keep lint, format, and `cspell`; vitest checks leave the list.

### 2026.07.18.1524

- Chore: merge the go-rewrite branch to main — a fast-forward (the branch was strictly ahead), so no merge commit and no hook exception needed.
- Docs: archive the completed milestone plan to `plans/archive/01-go-rewrite.md` with `status: completed` front matter per the plan-doc convention.
- Docs: the README Go-runner section now links to the archived plan.

### 2026.07.18.1521

- Feat: land plan 01 section 5 — lock it in; plan 01 is fully checked off.
- Feat: the go-parity harness (`npm run parity:go`) diffs every Go check's ok, errors, and filesChecked against its TypeScript twin; 27/27 agree, jscpd counts excepted.
- Ci: CI gains `go` (gofmt/vet/build/test), `go-parity`, and `fitness-ts` jobs; the `fitness` job now runs the Go suite.
- Feat: dogfood cutover: `.fitnessrc.json` carries the full 23-check list, and `npm run fitness` runs the Go runner (~1.9s vs ~6.5s for TypeScript); `npm run fitness:ts` stays CI-gated.
- Chore: README distribution picks GitHub Releases plus `go install`, with an npm shim only for npx continuity; `jscpd` joins both configs' repeated-string-literals allow baseline.

### 2026.07.18.1511

- Feat: plan 01 section 4 lands four tool-exec checks, completing the 27-name catalog; each runs its real tool from `node_modules/.bin` upward, then PATH.
- Feat: `prettier` ports staged filtering, glob mode, passthrough, and `[warn]` parsing; `eslint` runs the CLI with the shared flat config, matching TypeScript byte-for-byte.
- Feat: `vitest-coverage-full` reuses `internal/vitestconf` for the threshold gate (22 new cases), then runs `vitest run --coverage`; `swiftlint` parses real JSON violations, byte-identical against swiftlint 0.65.0.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Chore: full-catalog sweep: 26 of 27 checks byte-match their TypeScript twins here; only `jscpd` differs, in scanned-file counts, with verdict parity.

### 2026.07.18.1312

- Feat: `cspell` is a Go binary over `internal/spell`: camelCase-aware words, inline directives, default masks, and 14 committed wordlists (~217k entries) from `@cspell` packages.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Feat: `jscpd` is a Go binary over `internal/clonedetect`: a comment-stripping lexer, rolling-hash windows with jscpd's thresholds, the `jscpd:ignore-start`/`end` escape hatch, and batched `git check-ignore`.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Fix: staged 2.1MB wordlists hung pre-commit's `cspell`, as cspell ignores ignorePaths for explicit arguments; the filter drops covered paths, and `go/internal/spell/dict` joins ignorePaths.

### 2026.07.18.1208

- Feat: land plan 01 section 2: the parsers-and-network checks are Go binaries; all seven match their TypeScript twins, file counts included.
- Feat: `internal/mermaid` ports `mermaid.ts` exactly: fence scanning, five callout patterns with JS-lookahead emulation over RE2, GFM callout tables, and legend-invisible pairing.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Feat: the five mermaid checks are thin binaries over that parser; `vitest-coverage-exclude` scans config text with string-aware comment stripping in `internal/vitestconf`.
- Feat: `dependency-currency` swaps `npm outdated` for a native net/http registry client (`.npmrc` registry, bounded concurrency); offline or garbage responses pass; ~2.6x faster.

### 2026.07.18.1140

- Feat: land plan 01 section 1: all thirteen pure-logic checks are Go binaries, the markdown-filename pair as two thin binaries over `internal/mdfilename`.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Feat: `changelog` parses JSON for invalid-JSON errors; `semantic-commit` and `commit-attribution` declare the `--message` handshake; `repeated-string-literals` reads its allow list from `.fitnessrc.json`.
- Chore: add `.fitnessrc.json` with the `repeated-string-literals` allow baseline for the Go runner; the TypeScript suite keeps `.fitnessrc.js`, both in sync until cutover.
- Fix: `jscpd` also ignores Go test files (`**/*_test.go`); the two real production clones from the ports became a shared `walkfs.ScanFiles` loop and `render.ColorsEnabled`.

### 2026.07.18.1113

- Fix: Prettier has no parser for `.go` or `go.mod`, so the `prettier` check failed the first `go/` commit; staged mode drops `go/` paths.
- Feat: land plan 01 section 0, the Go scaffold: a stdlib-only `go/` module whose `fitness` runner and first check, `fitness-check-node-version`, run here.
- Feat: the runner finds check binaries beside it or on PATH, passes `--root` and `FITNESS_*`, reads stdout JSON, and kills hung process groups.
- Feat: config is `.fitnessrc.json`; a lone legacy `.fitnessrc.js`/`.ts` gets a migration hint on full-suite runs only; shared internals carry `go test` coverage.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.07.18.1057

- Docs: add `plans/01-go-rewrite.md`, the milestone plan to rebuild the runner and every check in Go as zero-dependency static binaries.
- Docs: one stdlib-only binary per check plus a `fitness` runner, speaking an exec protocol with JSON results.
- Docs: checks port one at a time with TypeScript parity; the dep-heavy four (prettier, eslint, vitest-coverage-full, swiftlint) come last, each decided on arrival.
- Docs: the plan follows the may-journals template — numbered title, Goal, numbered checkbox sections, verification last.
- Chore: bump `@typescript-eslint/eslint-plugin` + `@typescript-eslint/parser` 8.63.0 → 8.64.0, `eslint-plugin-jsdoc` 63.0.13 → 63.1.0, and `knip` 6.26.0 → 6.27.0 to satisfy `dependency-currency`; full build/fitness/lint suite verified on the updated tree.

### 2026.07.10.2041

- Feat: add `repeated-string-literals`: a literal used 3+ times across source files fails, skipping comments, regexes, templates, imports, directives, and tests. Closes #42.
- Feat: derive the bundler's check list from `packages/checks/*`; each directory bundles as its same-named check unless package.json maps entries via `fitnessChecks`. Closes #41.
- Feat: add opt-in `commit-attribution`: commit messages must disclose AI usage via `AI-Tools:` and `AI-Models:` trailers; merge and revert commits exempt. Closes #7.
- Fix: `eslint` drops the TypeScript type-checker no rule used, which cost ~5s and broke CI; parsing is byte-identical, ~4x faster (ADR 0001).
- Chore: dogfood both checks, fixing repeated literals (baseline 55 → 3), add `repeatedStringLiterals.allow`, bump TypeScript to 7.0.2 and eslint to 10.7.0, and merge `main`.

### 2026.07.07.0850

- Feat: add check packages `markdown-filename-convention` (kebab and camelCase, #11), `no-eslint-disable` (#10), `gitignore-why` (#6), `build-output-untracked` (#8), five mermaid checks, `dependency-currency` (#30), `jscpd`, and `swiftlint`.
- Feat: `.fitnessrc` `checks` mixes local module paths with npm check names in order; bad paths fail, and `disabledChecks` never removes them (#23).
- Fix: `dependency-currency` timed out in CI, so checks gain a `timeoutMs` the runner honors; `bundle-check-dist.mjs` always rebuilds, and `npx fitness` loads the consumer's `.fitnessrc`.
- Chore: update every dependency (TypeScript 6, ESLint 10, cspell 10, Vitest 4.1, knip 6), enable the full suite and `jscpd`, and drop `boxen` (#17).
- Build: publish only `@mayjournal/fitness-shared`, `fitness-checks`, and `fitness` via npm OIDC; Git hooks replace Husky; add C4 docs, Wardley map, and competition matrix.

### 2026.05.20.1716

- Perf: land the publish-audit plan (#13): an `audit:publish` script, publish-audit CI with PR comments, `bench:load-check`, per-workspace knip, and types-first `exports`.
- Fix: publish-audit CI tracks vitest `.d.ts` for publint, drops broken attw, strips publint ANSI, fixes the `findInstallRoot` test, and repairs PR-comment audit JSON.
- Feat: add provision scripts and publish-time checks that seed `@mayjournal` workspaces and set up trusted publishing, slimmed to OIDC `npm publish -ws` with `test:scripts` coverage.
- Chore: dependency and config hygiene: drop runner's redundant jiti, duplicate prettier plugins, misplaced check deps, and a duplicate cspell config; add `repository` fields.
- Build: `@mayjournal/fitness-shared` and `@mayjournal/fitness-checks` build via `fitness-shared build` (inline `node --eval` tsconfig generation removed); provision and related scripts formatted with Prettier for CI.

### 2026.05.20.1516

- Fix: `fitness-shared` build, lint, and test resolve the monorepo root internally.
- Refactor: remove `cd` from the package scripts now that the bins resolve the root themselves.
- Fix: generate an ephemeral runner `tsconfig.json` at lint time so ESLint `projectService` maps runner sources on clean checkouts.

### 2026.05.20.1509

- Refactor: centralize ESLint rules in `eslint.base.cjs`; the CLI and the eslint check both import `createEslintConfig` with their own parser options.
- Refactor: separate parser options keep `projectService` and `project` from conflicting.
- Feat: add `fitness-shared lint` for monorepo ESLint; workspace lint scripts route through it; pre-commit runs lint alongside fitness.
- Chore: check packages get a minimal lint-only `tsconfig.json` (extends the shared check config) for `projectService` discovery; monorepo check enumeration leaves the shared config.

### 2026.05.20.1453

- Feat: split the repo into npm workspaces: `packages/runner` and twelve `packages/checks/*` with dynamic loading, `@mayjournal/fitness-checks-bundle` defaults, and per-check tool deps (PR #12).
- Feat: checks fall back to `@mayjournal/fitness` configs when consumers lack cspell, prettier, vitest, or tsconfig, via `resolveFitnessConfigPath`, `resolveLintTsconfig`, and `tsconfig.lint.cjs`.
- Feat: restore `disabledChecks` on `.fitnessrc`: the optional list removes names from `checks` or bundle `defaultChecks` after `resolveCheckNames`, with tests.
- Build: centralize `compilerOptions` in `tsconfig.compiler.cjs`, generate and commit check and bundle tsconfigs, build `fitness-shared` first, and publish via npm OIDC with `NODE_AUTH_TOKEN`.
- Chore: CI runs workspace tests and per-package Vitest; `changelog-updated` matches the root version suffix; staged cspell honors `ignorePaths`; the stamper bumps `packages/` versions.

### 2026.04.04.1748

- Feat: export `./vitest.config` as `vitest.config.mjs`.
- Feat: add `vitest.config.d.ts` for TypeScript consumers.
- Fix: resolve Prettier plugin paths with `createRequire` so consumers loading `@mayjournal/fitness/prettier.config` resolve plugins from this package.

### 2026.04.04.1712

- Feat: the ESLint check runs via the Node API with this package's `eslint.config.cjs` and the parent cwd, so consumers need not install ESLint.
- Feat: add `getFitnessRunnerRoot`, shared with the runner and vitest-coverage-full, plus `runInProcess` and `RunContext._eslintRunForTesting`.
- Fix: show `file:line:col` errors by extracting the JSON array from mixed stderr; on parse failure, append truncated ESLint output to the fallback.
- Build: scope the package to `@mayjournal/fitness`: MIT LICENSE, a publish workflow on CI success, `publish:ci`, and `.npmrc` for `NPM_TOKEN`; add `plan-deps-vs-devdeps-check.md`.

### 2026.03.07.1922

- Chore: Prettier uses `prettier-plugin-packagejson` for conventional `package.json` field order; `package-lock.json` joins `.prettierignore`.
- Refactor: one source of truth for check registration: `Check.folder` plus a registry-built `RunContext.checkFolderByName` replace `CHECK_TO_FOLDER`; a test keeps enum and registry in sync.
- Refactor: shared `quoteForShell` (`src/utils/shellQuote.ts`) used by eslint, prettier, and cspell; a shared Vitest config loader (`src/checks/vitest-config`) used by both coverage checks.
- Refactor: shared exec helpers — `execSyncResult()` with `EXEC_OPTS`, and `buildExecCheckResult()` — adopted by eslint, prettier, cspell, vitest-coverage-full, and changelog-updated.
- Fix: `markdown-no-bold-italic` ignores emphasis inside link blocks `[text](url)` so underscores in URLs or link text are not falsely flagged.

### 2026.03.07.1431

- Fix: the runner dedupes `config.checks` by name — `checksFromConfigList` keeps the first occurrence, so a check listed multiple times in `.fitnessrc` runs once.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Refactor: extract `runImpl` to satisfy the eslint complexity ceiling.

### 2026.03.07.1406

- Feat: ESLint gains the `max-lines` rule (200, skipBlankLines/skipComments).
- Refactor: split `run.ts` into `run-resolve.ts` (getChecks, config/spec resolution), `run-execute.ts` (runOneCheck, worker/in-process), and `run-output.ts` (buildTable, buildTotalLine).
- Refactor: `run.ts` keeps orchestration only.

### 2026.03.07.1007

- Feat: registry checks run in workers, so `worker.terminate()` enforces the 5s timeout and the run continues; read-repo-first, vitest-coverage-full, and path-based checks stay in-process.
- Feat: `node_modules` is always ignored: `getSkipDirs` merges the runner skip dirs (`node_modules`, dist, coverage, .git, .husky) with config, and filters staged files too.
- Feat: progress messages on stderr (Resolving checks…, Running checks:, and → name before each check) show where a run is or where it hangs.
- Feat: add `vitest-coverage-full` (`vitest run --coverage` at 100% thresholds); cspell gains a CLI runner and enUS dictionary; `vitest-coverage-exclude` allows barrel `index.ts` excludes and reads `vitest.config.cjs`.
- Fix: `isMainModule` works via npx: argv[1] and `import.meta.url` resolve to real paths, so symlinked `.bin/fitness` counts as main; a symlink test rides along.

### 2026.02.22.1620

- Feat: add `checkResult(ok, errors?, filesChecked?)` and a `runContext` helper (getStagedFiles, getExecSync); every check migrates to them; `RunContext` gains `_now` for tests.
- Docs: add `plan-checks-abstractions.md` (the repeating patterns in `checks/*` and abstraction options); the README flowchart node renamed to PassthroughArgs; the cspell words list trimmed.
- Build: flatten dependencies into `dependencies` only (no dev/optional split).
- Chore: Prettier parses `package.json` as json so sort-json runs recursively over exports paths and condition keys; `prettier-plugin-packagejson` removed; `prettier.config.cjs` commented.
- Refactor: an `enUS` enum holds all user-facing runner copy, with an `interpolate()` util for `{{key}}` templates; eslint adds `typescript-sort-keys` with `@typescript-eslint` at ^8.55.

### 2026.02.22.1511

- Refactor: Prettier collapses to a single config (`prettier.config.cjs`); `.prettierrc.json` removed.
- Feat: add `.prettierignore` and `prettier.config.d.ts` with a package.json types export.
- Fix: the Prettier check recognizes `.ts`/`.mts`/`.cts` config names.
- Chore: gitignore the generated Prettier files and drop the unsupported `ignore` option from the config.

### 2026.02.22.1454

- Fix: `semantic-commit` fails when there is no message to validate (empty or git unavailable).
- Feat: the commit-msg hook passes message content via `--message="$(cat "$1")"`.
- Chore: export `MSG_EMPTY`.

### 2026.02.22.1431

- Refactor: replace `findMd` with `findFilesByExtension(root, extension)`.
- Feat: `getSkipDirs` uses `skipTheseDirectories` from `.fitnessrc` when present, else cspell.json `ignorePaths` (dir names only); no default list.
- Feat: add `FitnessConfig.skipTheseDirectories`.
- Chore: add `.git` and `.husky` to cspell.json.

### 2026.02.22.1408

- Feat: add the `CheckName` enum; all checks use it for `name` (no static strings).
- Refactor: the runner casts config check names to `CheckName` for registry lookup.
- Chore: export `CheckName` from types.

### 2026.02.22.1359

- Fix: make the README Mermaid flowchart edge labels readable; the theme gains `textColor` and `labelColor`, so yes/no arrow labels stop blending in.
- Docs: reroute the context edges — `buildContext(staged, inlineFragment, checks, passthrough)` now feeds `runChecks`, and the single-check `contextInline` fragment hangs off the resolved checks.
- Docs: annotate `getStagedContext()` with its mechanism (`git diff --cached` → stagedFiles).

### 2026.02.22.1332

- Feat: the runner uses check-registered `contextInline` — no check-name logic remains in the runner.
- Feat: the `Check` type gains optional `contextInline` (argName, contextKey); `ContextInline` is exported.
- Feat: `semantic-commit` registers `--message` → `proposedCommitMessage`; the commit-msg hook uses `--message="$(cat "$1")"`.
- Docs: update README and the flow diagram.

### 2026.02.22.1322

- Docs: the runner-no-check-names plan lands on check-registered context — `contextInline` (and a future `contextPath`) on the `Check` type.
- Docs: `.fitnessrc` needs no change under the chosen approach.
- Docs: the plan doc slims from exploratory options down to the decision (139 lines removed).

### 2026.02.22.1259

- Refactor: rename `specFromPositional` to `checkNameIsFirstArg` and `getCommitMsgPath` to `getContextFilePath` for clarity.
- Docs: add `plans/plan-runner-no-check-names.md` with front matter.
- Style: no bold/italic in the new plan, for the markdown checks.

### 2026.02.22.1247

- Refactor: abstract check dependencies out of the runner — `getColumns` moves to `src/utils/terminal` so the runner has no check-specific imports.
- Refactor: `read-repo-first` no longer exports `getColumns`.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.02.22.1241

- Docs: align the README Mermaid code-flow diagram with the real runner flow.
- Docs: nodes now name the actual functions — `resolveCheckSpec`, `resolveChecksBySpec`, `buildContext`.
- Docs: the spec-defined branch appears in the flow.

### 2026.02.16.1646

- Feat: the `changelog` check requires the `package.json` version suffix to match the first `###` heading (`yyyy.mm.dd.HHMM`).
- Feat: the `package-lock.json` version must match the same heading.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: update the check README.

### 2026.02.16.1634

- Feat: `changelog-updated` requires the new section heading to use the current date and time (`yyyy.mm.dd.HHMM`) so GenAI cannot guess the time.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Docs: update the check README for the current-time rule.

### 2026.02.16.1900

- Refactor: `changelog-updated` exports its human-facing message consts (`MSG_*`).
- Refactor: the implementation reuses the exported consts — one source for the copy.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.02.16.1800

- Feat: the runner adds `passthroughArgs` to `RunContext` when running a single check.
- Feat: args after the check name forward to checks (e.g. `npx fitness prettier --write`).
- Feat: the Prettier check runs Prettier with forwarded args instead of `--check` when present.

### 2026.02.16.1700

- Build: move eslint, prettier, vitest, and the related config plugins from devDependencies to dependencies so consumers can use the exported configs.
- Build: keep `@types/node`, `tsconfig.js`, tsx, and typescript as devDependencies.
- Feat: the Prettier check detects config via the package.json `"prettier"` field so consumers using `"prettier": "@mayjournal/fitness/prettier.config"` are checked.

### 2026.02.16.1600

- Feat: ESLint adds natural ascending `sort-keys` for object keys in `.ts`/`.cjs`/`.js`/`.mjs`; the ESLint check covers `.cjs`/`.js`/`.mjs`, and object literals are reordered.
- Feat: add Prettier (`eslint-config-prettier`, sort-json, CI format job, exported config); the Prettier check runs `prettier --check` on staged files or repo, skipping without config.
- Feat: the runner renders table feedback — a colspan row per failed check, errors contextual to the row, dynamic width via `getColumns` from read-repo-first.
- Docs: sync main and checks READMEs with the registry, add rules-front-matter README and `fitnessFunctions` to the eslint-check README, and update the flow diagram.
- Fix: `rules-front-matter` rejects empty `fitnessFunctions` and `relatedConfigurations` arrays — at least one entry per array.

### 2026.02.16.1500

- Feat: results render in a cli-table3 table: Check, Status, Files, Time columns, bold white headers, chalk green/red for results, errors, and totals.
- Feat: `read-repo-first` asks Y/N to confirm familiarity with checks, lists enabled ones with a plain-path Src column for IDE links, and runs first.
- Fix: `read-repo-first` drops its TTY requirement — feedback displays for Agent/User contexts and always passes; the `FITNESS_READ_REPO_CI_ONLY_DO_NOT_USE_OTHERWISE` bypass leaves CI.
- Chore: consolidate the cursor rules into `fitness-checks.mdc`, removing the per-check rule files.
- Chore: add `eslint.config.d.ts` for ESM package compatibility; package and gitignore updates for the consolidated rules.

### 2026.02.16.1400

- Feat: add `package.json` exports so downstream projects can reuse the shared configs.
- Feat: the exported entries — `./eslint.config`, `./vitest.config`, `./tsconfig.cjs`, and `./cspell` (raw `cspell.json` data).
- Build: ship the config files in the npm package — the `files` list adds them beside `dist`.

### 2026.02.16.1300

- Build: one root `tsconfig.cjs` drives build and lint, with options inlined and shared base and `build/` gone; `tsconfig.js` converts `.cjs` to gitignored JSON.
- Refactor: replace `eslint.config.ts` with `eslint.config.cjs` for ESM package compatibility.
- Chore: ESLint drops the stylistic plugin and rules, keeping `jsdoc/require-jsdoc` and `complexity` max 5 in a single block with project tsconfig.json.
- Fix: `changelog-updated` types the `ExecSyncFn` maxBuffer and adds an execSync fallback coverage test.
- Docs: the `vitest-coverage-exclude` README uses relative paths and is properly associated with the name of the check.

### 2026.02.16.1025

- Feat: the ESLint check runs eslint (staged paths or `.`), parses its JSON output, and reports errors.
- Fix: only `.ts`/`.tsx` staged paths are passed, avoiding no-config failures on `.md`; tests use `.ts` fixtures (bar, pathWithQuote, quoted).
- Refactor: hoist the feedback and CLI consts.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.02.16.1005

- Feat: the runner adds feedback dressing — "Please fix these items."
- Refactor: hoist the messages to shared consts.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.02.16.0958

- Docs: add the `read-repo-first` check design doc (`plans/read-repo-first-check.md`).
- Docs: frame the problem as machine-checkable proxies for a behavioral rule, with four directions: structure-only validation, staged correlation, advisory no-op, and configurable hybrid.
- Docs: draft recommendation — start with structure-only validation (numbered folders at the repo root) as the minimal viable check.

### 2026.02.15.1700

- Docs: clarify the README tagline (fitness runner, checks, workflows).
- Perf: run cspell in-process via cspell-lib (readConfigFile, spellCheckFile) for speed; the CLI path remains when tests mock exec.
- Feat: the summary prints total success and failure counts, rounded time, and total files scanned (the sum of `filesChecked` from checks).
- Fix: `markdown-no-bold-italic` no longer flags unordered-list asterisk markers as italic.

### 2026.02.15.1600

- Docs: remove bold/italic from the check READMEs to satisfy `markdown-no-bold-italic`; README and CHANGELOG front matter fixed (---, flow-style).
- Feat: `markdown-front-matter` requires `fitnessFunctions` or `relatedConfigurations` in every `.md`, with file-relative paths or a check name; `findMd` skips `node_modules`, dist, coverage, .git, .husky.
- Feat: `vitest-coverage-exclude` allows only `**/*.d.ts` and `**/*.types.ts` in coverage exclude, as Vitest skips tests by default; type-only files become `*.types.ts`.
- Chore: remove `.fitnessrc.ts`, with a runner test covering custom checks from a config to restore 100% coverage.
- Ci: one fitness job runs `npm run fitness`, replacing the discover matrix; the husky pre-commit hook sources nvm, so `nvm use` runs without it on PATH.

### 2026.02.15.1500

- Refactor: the node-version check compares `.nvmrc` to the current Node only — the earlier nvm-subshell validation (with a process.version fallback) is removed.
- Ci: the CI script runs `nvm use` when available.
- Fix: default the strategy matrix `fromJson(needs.discover.outputs.checks)` to `'[]'` when the checks output is empty.

### 2026.02.15.1400

- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.
- Feat: two positionals (check then message path) supported for semantic-commit.
- Fix: `getPositionalSpec` and `getCommitMsgContext` handle single vs two positionals.

### 2026.02.15.1300

- Feat: the rules front-matter check validates `fitnessFunctions` and `relatedConfigurations` paths in all markdown.
- Refactor: a shared `findMd` helper backs the markdown checks.
- Feat: the runner accepts a check by name or path (`--check=./path/to/check.js` or positional).
- Feat: a `Check` loads from a module's default or named export.

### 2026.02.15.0100

- Refactor: the runner takes a single CLI flag — `--check=` only.
- Feat: the commit-msg path rides as a positional.
- Feat: staged context is always built.
- Test: the release gate checks the linux platform entries with a strings.HasPrefix filter, since darwin no longer verifies.

### 2026.02.15.1200

- Ci: GitHub Actions runs dynamic fitness jobs from the registry with a composite setup action.
- Feat: add the optional cspell check, run when `cspell.json` exists; cspell lives only in runner dependencies, and spell script leaves ci and package.json.
- Docs: add a Mermaid code-flow diagram to the README (modern colors, moved to the bottom); cspell README updates.

### 2026.02.15.1100

- Feat: a commit-msg hook runs `semantic-commit`; the check merges into a single file; cursor rules point to the check READMEs.
- Feat: the `changelog` check requires `### yyyy.mm.dd.HHMM` — every `###` heading must match, and the section format matches the package version.
- Feat: `changelog-updated` suggests up to 10 random words from the staged diff when overlap is too low.
- Chore: remove the duplicate check-node-version script and check-node; ci runs `npm run fitness` only.

### 2026.02.15.1000

- Feat: add the fitness config (`.fitnessrc.ts`) and config loader; turn on all checks (changelog, semantic-commit).
- Feat: add the `changelog-updated` check — fuzzy-matches the staged diff to the changelog, checking only changed lines, not the whole file.
- Refactor: colocate tests with source; merge the runner tests.
