package cmd

import (
	"context"
	"strings"
	"testing"

	askrunner "github.com/oski646/repoq/internal/ask"
)

func TestRootCommandRequiresQuestion(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd(stubRunner{})
	cmd.SetArgs([]string{"openai/codex"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--question must not be empty") {
		t.Fatalf("expected required question error, got %v", err)
	}
}

func TestRootCommandRejectsInvalidRepo(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd(stubRunner{answer: "ok"})
	cmd.SetArgs([]string{"invalid/repo/shape", "--question", "hello"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "repository must be owner/repo or a full GitHub URL") {
		t.Fatalf("expected invalid repo error, got %v", err)
	}
}

func TestRootCommandPrintsAnswer(t *testing.T) {
	t.Parallel()

	runner := stubRunner{answer: "found answer"}
	cmd := NewRootCmd(runner)

	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"openai/codex", "-q", "where is auth?"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout.String() != "<answer>\nfound answer\n</answer>\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRootCommandPassesInstructions(t *testing.T) {
	t.Parallel()

	runner := &captureRunner{answer: "ok"}
	cmd := NewRootCmd(runner)
	cmd.SetArgs([]string{"openai/codex", "-q", "where is auth?", "--instructions", "Prefer a short answer."})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.options.Instructions != "Prefer a short answer." {
		t.Fatalf("unexpected instructions: %q", runner.options.Instructions)
	}
}

type stubRunner struct {
	answer string
	err    error
}

func (s stubRunner) Run(_ context.Context, opts askrunner.Options) (string, error) {
	if _, err := askrunner.NormalizeGitHubRepo(opts.Repository); err != nil {
		return "", err
	}
	if s.err != nil {
		return "", s.err
	}
	return s.answer, nil
}

type captureRunner struct {
	answer  string
	options askrunner.Options
}

func (r *captureRunner) Run(_ context.Context, opts askrunner.Options) (string, error) {
	r.options = opts
	return r.answer, nil
}
