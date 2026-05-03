package ask

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

func BuildCodexArgs(question string, promptContext PromptContext, outputPath, instructions, model string) []string {
	return []string{
		"exec",
		"-m", model,
		"--sandbox", "read-only",
		"--ephemeral",
		"--color", "never",
		"--output-last-message", outputPath,
		buildPrompt(question, promptContext, instructions),
	}
}

func (r *CommandRunner) runAnalysis(
	ctx context.Context,
	settings Settings,
	workingDir, question string,
	promptContext PromptContext,
	instructions string,
	stderr io.Writer,
) (string, error) {
	switch settings.Provider {
	case ProviderCursor:
		return r.runCursor(ctx, settings, workingDir, question, promptContext, instructions, stderr)
	case ProviderCodex:
		return r.runCodex(ctx, settings, workingDir, question, promptContext, instructions, stderr)
	default:
		return "", fmt.Errorf("unsupported provider %q", settings.Provider)
	}
}

func (r *CommandRunner) runCodex(
	ctx context.Context,
	settings Settings,
	workingDir, question string,
	promptContext PromptContext,
	instructions string,
	stderr io.Writer,
) (string, error) {
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

	cmd := r.Command(ctx, "codex", BuildCodexArgs(question, promptContext, outputPath, instructions, settings.Model)...)
	cmd.Dir = workingDir

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	fmt.Fprintf(stderr, "starting codex (%s) analysis\n", settings.Model)

	done := make(chan struct{})
	if r.HeartbeatInterval > 0 {
		startedAt := time.Now()
		go func() {
			ticker := time.NewTicker(r.HeartbeatInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					elapsed := time.Since(startedAt).Round(time.Second)
					fmt.Fprintf(stderr, "still analyzing... %s elapsed\n", elapsed)
				case <-done:
					return
				}
			}
		}()
	}

	err = cmd.Run()
	close(done)
	if err != nil {
		message := strings.TrimSpace(output.String())
		if message != "" {
			return "", fmt.Errorf("run codex: %w: %s", err, message)
		}
		return "", fmt.Errorf("run codex: %w", err)
	}

	fmt.Fprintln(stderr, "codex analysis finished")

	answer, err := r.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("read codex answer: %w", err)
	}

	return string(answer), nil
}
