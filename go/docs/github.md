# github

Local checks and GitHub Actions share one set of GitHub judgments, kept in go/internal/metadata.

Actions run pr-check, issue-check, plan-check, close-check, dupe-check and comment-check on each event, while branch-check runs the same judgments locally without posting anything.

## PushDeadline

PushDeadline bounds every GitHub read one push makes, the gate's and local-branch's together, and any one GitHub read outside a push.

## CommentChecks

CommentChecks judge one issue or PR comment: its own structure, then the prose and markdown checks every Plan and PR body gets.

## KnownBots

KnownBots are the org's automation accounts, whose comments no one can fix; the comment-check workflows repeat this list, so change both.

## PRBodyChecks

PRBodyChecks are the checks pr-check gives a PR body, which the Claude Code hook also gives a draft one.

## RepoChecks

RepoChecks applies a repo's own check settings to the GitHub commands, and left nil, no check runs, so each fails closed. Reaches, when set, reports whether a check's files and exclude patterns let a body's virtual path reach it.

## PushPlans

PushPlans are the PRs and Plans one push read successfully, each read once. A failed read is not kept, so a later ask tries GitHub again while the push's deadline lasts.

## NewPushPlans

NewPushPlans starts a push with no Plans read, whose reads all end when ctx, the push's deadline from PushContext, does.

## PushPlans.Approved

Approved reports whether the Plan ref names has the `Plan` label and a comment by an allowed approver that says approve.

## PushPlans.Handoff

Handoff writes the reads to a file only this user can read or write and names it for local-branch. The returned func removes the file.

## PushPlans.PRPlans

PRPlans finds the open PR from branch and returns the issues its body closes, in any spelling, with the PR's URL. The URL is empty when the branch has no open PR yet or origin is not on GitHub.

## PushPlans.TimedOut

TimedOut is one line naming the GitHub host and the deadline once the push's deadline has passed, else empty.

## BranchChecks

BranchChecks are the checks branch-check judges the branch's PR and Plan with: the PR body's, its title's and links', and the Plan body's.

## GitHubAPI

GitHubAPI is the address of GitHub's API: `GITHUB_API_URL`, else api.github.com for github.com, else GitHubHost itself, as GitHub Enterprise Server serves it.

## GitHubHost

GitHubHost is the host gh reads from: `GH_HOST`, else github.com.

## IssueBodyChecks

IssueBodyChecks are the checks issue-check gives a body of the kind label names, or nil for a label that names no kind. An unlabeled body, "", gets the body checks alone.

## LocalBranch

LocalBranch judges live metadata through the normal check result protocol.

## PushContext

PushContext opens the deadline every GitHub read of one push shares.

## RunDependabotPR

RunDependabotPR completes a Dependabot pull request, so every PR check judges it as it judges a person's.

## RunGH

RunGH runs the named GitHub command against the Actions event.

## UnreachableGitHub

UnreachableGitHub probes GitHub's API host once per process and returns it when silent, sharing one answer with the gate and runner.

## WriteBody

WriteBody saves body to a temp file for the --body-file checks; the caller removes it.

## WriteDocument

WriteDocument saves body, judged at path doc, to a temp file ending in doc's extension. A check that reads the file's type sees doc's, and the caller removes the file.
