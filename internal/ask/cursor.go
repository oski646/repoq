package ask

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

func BuildCursorArgs(question string, promptContext PromptContext, instructions, model, workspace string) []string {
	return []string{
		"agent",
		"--print",
		"--output-format", "text",
		"--mode", "ask",
		"--sandbox", "enabled",
		"--trust",
		"--workspace", workspace,
		"--model", model,
		buildPrompt(question, promptContext, instructions),
	}
}

func (r *CommandRunner) runCursor(
	ctx context.Context,
	settings Settings,
	workingDir, question string,
	promptContext PromptContext,
	instructions string,
	stderr io.Writer,
) (string, error) {
	cmd := r.Command(ctx, "cursor", BuildCursorArgs(question, promptContext, instructions, settings.Model, workingDir)...)
	cmd.Dir = workingDir

	var stdout bytes.Buffer
	var stderrOutput bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderrOutput

	fmt.Fprintf(stderr, "starting cursor (%s) analysis\n", settings.Model)

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
		message := strings.TrimSpace(stderrOutput.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message != "" {
			return "", fmt.Errorf("run cursor: %w: %s", err, message)
		}
		return "", fmt.Errorf("run cursor: %w", err)
	}

	fmt.Fprintln(stderr, "cursor analysis finished")

	return stdout.String(), nil
}
