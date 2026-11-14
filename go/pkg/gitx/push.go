package gitx

import "strings"

// BranchContext names the GitHub branch and the local commit being checked.
type BranchContext struct {
	Name string `json:"name"`
	Tip  string `json:"tip"`
}

// PushedBranches selects each live destination branch once, excluding tags.
func PushedBranches(updates []string) []BranchContext {
	seen := map[string]bool{}
	var branches []BranchContext
	for _, update := range updates {
		branch := pushedBranch(update)
		if branch != nil && !seen[branch.Name] {
			branches = append(branches, *branch)
			seen[branch.Name] = true
		}
	}
	return branches
}

func pushedBranch(update string) *BranchContext {
	f := strings.Fields(update)
	if len(f) < 4 || f[1] == strings.Repeat("0", 40) || !strings.HasPrefix(f[2], "refs/heads/") {
		return nil
	}
	return &BranchContext{Name: strings.TrimPrefix(f[2], "refs/heads/"), Tip: f[1]}
}
