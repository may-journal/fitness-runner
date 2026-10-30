// Command fitness-check-workflow-reporting fails a workflow job that can
// fail without a report on its run page ([#181]). [ADR 0003] asks every
// job for a job summary, so a failure never shows only a red badge.
//
// # Behavior
//
//   - It reads the `.yml` and `.yaml` workflow files directly under `.github/workflows`, keeping each value's line.
//   - Pass: each job with `steps` has one step with `if: always()`, bare or as `${{ always() }}`, whose `run` writes to `$GITHUB_STEP_SUMMARY`.
//   - Pass: a job that only calls a reusable workflow with `uses:` has no steps, and the called workflow reports.
//   - A matrix job is one definition, so it is judged once and named once.
//   - Fail: a job with steps and no such step, naming the file, the job's line, and the step to add.
//   - Fail: a job whose summary step lacks `if: always()`, naming that step's line.
//   - Fail: a file that is not valid YAML, naming the line where reading stopped.
//   - A repo with no workflow files passes having checked none.
//
// In a pull request, or a staged commit, it reads only the changed workflow
// files. It reads the files only: it never runs a step or fetches a called
// workflow.
//
// # Errors
//
//	.github/workflows/ci.yml:8: job build has no step that runs with if: always() and writes $GITHUB_STEP_SUMMARY, so a failed run shows only a badge; add a last step with `if: ${{ always() }}` whose run writes the job's status to "$GITHUB_STEP_SUMMARY"
//	.github/workflows/ci.yml:3: this workflow is not valid YAML: a tab indents this line; YAML indents with spaces; fix the YAML on this line
//
// # Adding the step
//
// A last step like this one reports every outcome:
//
//	steps:
//	  - name: Report job outcome
//	    if: ${{ always() }}
//	    run: printf '### Job %s: %s\n' "$GITHUB_JOB" "${{ job.status }}" >> "$GITHUB_STEP_SUMMARY"
//
// [#181]: https://github.com/may-journal/fitness-runner/issues/181
// [ADR 0003]: https://github.com/may-journal/fitness-runner/blob/main/fitness/adr-0003/README.md
package check
