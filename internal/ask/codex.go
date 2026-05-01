package ask

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

const CodexModel = "gpt-5.4-mini"

func BuildCodexArgs(question string, promptContext PromptContext, outputPath, instructions string) []string {
	return []string{
		"exec",
		"-m", CodexModel,
		"--sandbox", "read-only",
		"--ephemeral",
		"--color", "never",
		"--output-last-message", outputPath,
		buildPrompt(question, promptContext, instructions),
	}
}

func (r *CommandRunner) runCodex(
	ctx context.Context,
	workingDir, question string,
	promptContext PromptContext,
	outputPath string,
	instructions string,
	stderr io.Writer,
) error {
	cmd := r.Command(ctx, "codex", BuildCodexArgs(question, promptContext, outputPath, instructions)...)
	cmd.Dir = workingDir

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	fmt.Fprintln(stderr, "starting codex analysis")

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

	err := cmd.Run()
	close(done)
	if err != nil {
		message := strings.TrimSpace(output.String())
		if message != "" {
			return fmt.Errorf("run codex: %w: %s", err, message)
		}
		return fmt.Errorf("run codex: %w", err)
	}

	return nil
}
