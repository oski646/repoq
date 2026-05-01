package ask

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

func (r *CommandRunner) runClone(ctx context.Context, repo Repository, ref, destination string) error {
	cmd := r.Command(ctx, "git", BuildCloneArgs(repo, ref, destination)...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message != "" {
			return fmt.Errorf("clone repository: %w: %s", err, message)
		}
		return fmt.Errorf("clone repository: %w", err)
	}

	return nil
}

func (r *CommandRunner) resolvePromptContext(
	ctx context.Context,
	repo Repository,
	workingDir, requestedRef string,
) (PromptContext, error) {
	commit, err := r.runGitString(ctx, workingDir, "rev-parse", "HEAD")
	if err != nil {
		return PromptContext{}, err
	}

	return PromptContext{
		RepositoryURL: strings.TrimSuffix(repo.CloneURL(), ".git"),
		RequestedRef:  strings.TrimSpace(requestedRef),
		Commit:        strings.TrimSpace(commit),
	}, nil
}

func (r *CommandRunner) runGitString(
	ctx context.Context,
	workingDir string,
	args ...string,
) (string, error) {
	cmd := r.Command(ctx, "git", args...)
	cmd.Dir = workingDir

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message != "" {
			return "", fmt.Errorf("inspect repository metadata: %w: %s", err, message)
		}
		return "", fmt.Errorf("inspect repository metadata: %w", err)
	}

	return strings.TrimSpace(output.String()), nil
}
