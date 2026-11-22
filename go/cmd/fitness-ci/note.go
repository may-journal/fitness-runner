package main

import (
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

// noteMarker ends the bot's auto-merge note on a PR, so a later run finds
// and edits it instead of posting another.
const noteMarker = "<!-- fitness:auto-merge -->"

// noteRefused is the warning when GitHub refuses a change to a note.
const noteRefused = "GitHub refused a change to a PR's auto-merge comment; its answer is above."

// issuesPath is the REST path of the run's repository's issues.
func issuesPath() string { return actions.RepoPath() + "/issues" }

// noteID finds the bot's note on PR pr: its comment id, or "" for none.
func noteID(pr string) (string, error) {
	query := fmt.Sprintf(`.[] | select(.user.login == "%s" and (.body | contains("%s"))) | .id`, env("BOT"), noteMarker)
	ids, err := actions.Gh("api", issuesPath()+"/"+pr+"/comments", "--paginate", "--jq", query)
	id, _, _ := strings.Cut(ids, "\n")
	return id, err
}

// writeNote posts text as the note on PR pr when it has none at id, or
// edits the note when its text changed. GitHub may store CRLF line ends,
// so those never count.
func writeNote(pr, id, text string) error {
	body := text + "\n\n" + noteMarker
	if id == "" {
		_, err := actions.Gh("api", issuesPath()+"/"+pr+"/comments", "-f", "body="+body)
		return err
	}
	current, err := actions.Gh("api", issuesPath()+"/comments/"+id, "--jq", ".body")
	if err == nil && strings.ReplaceAll(current, "\r", "") != body {
		_, err = actions.Gh("api", "-X", "PATCH", issuesPath()+"/comments/"+id, "-f", "body="+body)
	}
	return err
}

// say writes text as the note on PR pr, and warns without failing when
// GitHub refuses.
func say(pr, text string) {
	id, err := noteID(pr)
	if err == nil {
		err = writeNote(pr, id, text)
	}
	if err != nil {
		actions.Warning(noteRefused)
	}
}

// sayAll writes text as the note on each PR of prs.
func sayAll(prs []string, text string) {
	for _, pr := range prs {
		say(pr, text)
	}
}

// sayIfNoted rewrites the note on PR pr as text only when one exists, since
// a PR auto-merge never had to explain gets none.
func sayIfNoted(pr, text string) {
	id, err := noteID(pr)
	if err == nil && id != "" {
		err = writeNote(pr, id, text)
	}
	if err != nil {
		actions.Warning(noteRefused)
	}
}
