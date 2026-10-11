package metadata

import (
	"context"
	"encoding/json"
	"os"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// branchContexts uses push destinations and local tips only inside pre-push.
func branchContexts(ctx context.Context, root string) ([]gitx.BranchContext, error) {
	if os.Getenv("FITNESS_INTERNAL_HOOK") == "pre-push" {
		var branches []gitx.BranchContext
		err := json.Unmarshal([]byte(os.Getenv("FITNESS_PUSH_BRANCHES")), &branches)
		return branches, err
	}
	name, err := currentBranch(ctx, root)
	if err != nil || name == "" {
		return nil, err
	}
	return []gitx.BranchContext{{Name: name, Tip: "HEAD"}}, nil
}
