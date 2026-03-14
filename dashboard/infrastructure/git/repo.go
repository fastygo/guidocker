package gitrepo

import (
	"context"
	"dashboard/domain"
	"fmt"
	"os/exec"
	"strings"
)

type Repository struct{}

// NewGitRepository creates a new git repository adapter.
func NewGitRepository() *Repository {
	return &Repository{}
}

// Clone clones a public repository into destination path and returns resolved commit hash.
func (r *Repository) Clone(ctx context.Context, sourceURL, branch, destination string) (string, error) {
	args := []string{"clone"}
	if strings.TrimSpace(branch) != "" {
		args = append(args, "--branch", strings.TrimSpace(branch))
	}
	args = append(args, "--depth", "1", sourceURL, destination)

	cloneCmd := exec.CommandContext(ctx, "git", args...)
	output, err := cloneCmd.CombinedOutput()
	if err != nil {
		cleanOutput := strings.ToLower(strings.TrimSpace(string(output)))
		if strings.Contains(cleanOutput, "pathspec") && strings.Contains(cleanOutput, "did not match") {
			return "", domain.ErrRepoBranchNotFound
		}
		if strings.Contains(cleanOutput, "couldn't find remote branch") || strings.Contains(cleanOutput, "did not match any file(s) known to git") {
			return "", domain.ErrRepoBranchNotFound
		}
		if strings.Contains(cleanOutput, "repository not found") || strings.Contains(cleanOutput, "repository could not be found") {
			return "", fmt.Errorf("%w: %s", domain.ErrInvalidRepoURL, cleanOutput)
		}
		return "", fmt.Errorf("git clone: %s", cleanOutput)
	}

	revCmd := exec.CommandContext(ctx, "git", "-C", destination, "rev-parse", "HEAD")
	commitOutput, err := revCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve commit: %w", err)
	}

	return strings.TrimSpace(string(commitOutput)), nil
}
