// cspell:ignore releasemeta
//
//fitness:container tools
//fitness:component tool It holds the release facts the tools share.
package releasemeta

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/changelog"
)

// Notes is a release's description: the body of each entry file it ships,
// newest first, then a link to every commit since since. An empty since
// means the first release.
func Notes(entries []changelog.Entry, repo, since, tag string) string {
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(e.Body() + "\n\n")
	}
	if len(entries) == 0 && since != "" {
		b.WriteString("No changelog entries since " + since + ".\n\n")
	}
	if since == "" {
		fmt.Fprintf(&b, "[Commits](https://github.com/%s/commits/%s)\n", repo, tag)
	} else {
		fmt.Fprintf(&b, "[Compare view](https://github.com/%s/compare/%s...%s)\n", repo, since, tag)
	}
	return b.String()
}

// PublishNotes writes tag's release description from the entry files added
// since the previous release tag. It finds the release among all releases,
// so a draft is found too, and replaces only its description.
func (c Client) PublishNotes(ctx context.Context, tag string) error {
	if tag == "" {
		return fmt.Errorf("GITHUB_REF_NAME is empty; run release-notes for a release tag")
	}
	since, err := changelog.PreviousTag(c.Root, tag)
	if err != nil {
		return err
	}
	entries, err := changelog.AddedSince(c.Root, since, tag)
	if err != nil {
		return err
	}
	id, err := c.releaseID(ctx, tag)
	if err != nil {
		return err
	}
	data, _ := json.Marshal(map[string]string{"body": Notes(entries, c.Repo, since, tag)})
	_, err = c.api(ctx, "--method", "PATCH", "repos/"+c.Repo+"/releases/"+id, "--input", "-", string(data))
	return err
}

// releaseID is the ID of the GitHub release for tag.
func (c Client) releaseID(ctx context.Context, tag string) (string, error) {
	jq := fmt.Sprintf(".[] | select(.tag_name == %q) | .id", tag)
	ids, err := c.lines(ctx, "--paginate", "--jq", jq, "repos/"+c.Repo+"/releases?per_page=100")
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no GitHub release for %s; Release Please creates it when the release PR merges", tag)
	}
	return ids[0], nil
}

// lines runs a gh api call and splits its output into non-empty lines.
func (c Client) lines(ctx context.Context, args ...string) ([]string, error) {
	out, err := c.api(ctx, args...)
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}
