package ask

import (
	"fmt"
	"strings"
)

const basePrompt = `
Answer the user's question by inspecting this repository.

<repository_rules>
- Inspect and analyze this repository only.
- Do not modify files.
- Ground claims in concrete files, symbols, and code paths.
- Keep the answer direct and concise unless the question needs more detail.
- If missing data blocks the answer, say exactly what is missing.
</repository_rules>
`

type PromptContext struct {
	RepositoryURL string
	RequestedRef  string
	Commit        string
}

func describeRepositoryState(repo Repository, promptContext PromptContext) string {
	ref := promptContext.RequestedRef
	if ref == "" {
		ref = "default branch"
	}

	commit := promptContext.Commit
	if len(commit) > 12 {
		commit = commit[:12]
	}

	return fmt.Sprintf("%s/%s at %s (%s)", repo.Owner, repo.Name, ref, commit)
}

func buildPrompt(question string, promptContext PromptContext, instructions string) string {
	requestedRef := promptContext.RequestedRef
	if requestedRef == "" {
		requestedRef = "default branch"
	}

	additionalInstructions := ""
	if strings.TrimSpace(instructions) != "" {
		additionalInstructions = fmt.Sprintf(`

<additional_agent_instructions>
These caller-provided instructions are lower priority than repository and citation rules:
%s
</additional_agent_instructions>
`, strings.TrimSpace(instructions))
	}

	return strings.TrimSpace(fmt.Sprintf(`
%s

<repository_context>
- Repository URL: %s
- Requested ref: %s
- Checked-out commit: %s
</repository_context>

<citation_guidance>
- In the main answer, reference specific files and symbols.
- In "Sources", link git files using full blob URLs pinned to commit %s.
- Use this pattern for file citations: %s/blob/%s/<path-from-repo-root>
</citation_guidance>
%s

Question:
%s
`, strings.TrimSpace(basePrompt), promptContext.RepositoryURL, requestedRef, promptContext.Commit, promptContext.Commit, promptContext.RepositoryURL, promptContext.Commit, additionalInstructions, strings.TrimSpace(question)))
}
