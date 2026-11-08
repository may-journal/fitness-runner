// cspell:ignore mispeled
package requirements

import (
	"encoding/base64"
	"testing"
)

// renovateBody is the end of a Renovate PR body, with the hidden debug block
// that holds its state encoded as Renovate encodes it.
var renovateBody = "This PR updates the tooling.\n\n<!--renovate-debug:" +
	base64.StdEncoding.EncodeToString([]byte(`{"createdInVer":"44.222.2","updatedInVer":"44.222.2","targetBranch":"main","labels":[]}`)) +
	"-->\n"

func Test0156_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, code := piped(t, repo, renovateBody, nil, "--blob", "-", "--checks=cspell")
	sees(t, out, code, 0, "All 1 checks passed")
	out, code = piped(t, repo, misspelledNote+renovateBody, nil, "--blob", "-", "--checks=cspell")
	sees(t, out, code, 1, "Unknown word (mispeled)")
}
