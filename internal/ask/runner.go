package ask

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const DefaultCacheRoot = "/tmp/repoq/repos"

type Options struct {
	Repository   string
	Question     string
	Ref          string
	Instructions string
	Stdout       io.Writer
	Stderr       io.Writer
}

type Runner interface {
	Run(ctx context.Context, opts Options) (string, error)
}

type commandFactory func(context.Context, string, ...string) *exec.Cmd

type CommandRunner struct {
	CacheRoot         string
	HeartbeatInterval time.Duration
	LookPath          func(string) (string, error)
	Command           commandFactory
	CreateTemp        func(string, string) (*os.File, error)
	ReadFile          func(string) ([]byte, error)
	MkdirAll          func(string, os.FileMode) error
	Stat              func(string) (os.FileInfo, error)
	Remove            func(string) error
	DefaultErr        io.Writer
}

func NewRunner() *CommandRunner {
	return &CommandRunner{
		CacheRoot:         DefaultCacheRoot,
		HeartbeatInterval: 5 * time.Second,
		LookPath:          exec.LookPath,
		Command:           exec.CommandContext,
		CreateTemp:        os.CreateTemp,
		ReadFile:          os.ReadFile,
		MkdirAll:          os.MkdirAll,
		Stat:              os.Stat,
		Remove:            os.Remove,
		DefaultErr:        os.Stderr,
	}
}

func (r *CommandRunner) Run(ctx context.Context, opts Options) (string, error) {
	if err := CheckDependencies(r.LookPath); err != nil {
		return "", err
	}
	if strings.TrimSpace(opts.Question) == "" {
		return "", errors.New("question must not be empty")
	}

	repo, err := NormalizeGitHubRepo(opts.Repository)
	if err != nil {
		return "", err
	}

	stderr := opts.Stderr
	if stderr == nil {
		stderr = r.DefaultErr
	}

	cacheDir := CachePath(r.CacheRoot, repo, opts.Ref)
	if _, err := r.Stat(cacheDir); errors.Is(err, os.ErrNotExist) {
		if err := r.MkdirAll(filepath.Dir(cacheDir), 0o755); err != nil {
			return "", fmt.Errorf("create cache parent directory: %w", err)
		}
		fmt.Fprintf(stderr, "preparing repository %s\n", repo.CloneURL())
		if err := r.runClone(ctx, repo, opts.Ref, cacheDir); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", fmt.Errorf("check cache directory: %w", err)
	}

	promptContext, err := r.resolvePromptContext(ctx, repo, cacheDir, opts.Ref)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(stderr, "repository ready: %s\n", describeRepositoryState(repo, promptContext))

	outputFile, err := r.CreateTemp("", "repoq-codex-*.txt")
	if err != nil {
		return "", fmt.Errorf("create codex output file: %w", err)
	}
	outputPath := outputFile.Name()
	if err := outputFile.Close(); err != nil {
		return "", fmt.Errorf("close codex output file: %w", err)
	}
	defer func() {
		_ = r.Remove(outputPath)
	}()

	if err := r.runCodex(ctx, cacheDir, opts.Question, promptContext, outputPath, opts.Instructions, stderr); err != nil {
		return "", err
	}

	fmt.Fprintln(stderr, "codex analysis finished")

	answer, err := r.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("read codex answer: %w", err)
	}

	trimmed := strings.TrimSpace(string(answer))
	if trimmed == "" {
		return "", errors.New("codex returned an empty answer")
	}

	return trimmed, nil
}

func CheckDependencies(lookPath func(string) (string, error)) error {
	for _, binary := range []string{"git", "codex"} {
		if _, err := lookPath(binary); err != nil {
			return fmt.Errorf("%s is not installed or not available in PATH", binary)
		}
	}

	return nil
}
