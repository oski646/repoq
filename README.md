# repoq

Ask questions about GitHub repositories from your terminal.

`repoq` clones a GitHub repository into a local cache, checks out the requested branch or tag, then runs Codex against that checkout in read-only mode. It is meant for quick repository research when you want an answer grounded in the actual files, symbols, and code paths.

## What it does

- Accepts GitHub repositories as `owner/repo` or full GitHub URLs.
- Supports an optional branch or tag with `--ref`.
- Runs Codex with a repository-aware prompt.
- Keeps cloned repositories in a local cache so repeated questions are faster.
- Asks Codex to cite concrete files and pinned GitHub blob URLs.
- Runs Codex in a read-only sandbox so analysis does not modify the cloned repository.

## Requirements

- Go 1.26 or newer.
- `git` available in `PATH`.
- `codex` available in `PATH` and already authenticated.

## Installation

```sh
go install github.com/oski646/repoq@latest
```

For local development:

```sh
git clone https://github.com/oski646/repoq.git
cd repoq
go build -o repoq .
```

## Usage

```sh
repoq <github_repository> --question "<question>"
```

Repository input can be short:

```sh
repoq openai/codex --question "Where is the CLI argument parsing implemented?"
```

Or a full GitHub URL:

```sh
repoq https://github.com/openai/codex --question "How does authentication work?"
```

Inspect a specific branch or tag:

```sh
repoq openai/codex --ref main --question "Where are commands registered?"
```

Add extra guidance for the analysis:

```sh
repoq openai/codex \
  --question "Explain the startup flow" \
  --instructions "Keep the answer short and cite the most important files."
```

`repoq` prints progress messages to stderr. The final response is printed to stdout inside `<answer>` tags:

```text
<answer>
The answer from Codex.
</answer>
```

## Options

| Flag | Short | Description |
| --- | --- | --- |
| `--question` | `-q` | Question to ask about the repository. Required. |
| `--ref` | | Branch or tag to inspect. |
| `--instructions` | | Extra instructions passed to the analysis agent. |

## Cache

Repositories are cached under:

```text
/tmp/repoq/repos
```

The cache key includes the repository owner, repository name, and requested ref. If no ref is provided, `repoq` uses a `default` cache entry.

Remove cached repositories manually if you want a fresh clone:

```sh
rm -rf /tmp/repoq/repos
```

## Development

Run tests with:

```sh
go test ./...
```

The test suite uses fake `git` and `codex` binaries for integration-style coverage, so it does not need network access or a real Codex session.

## Project status

`repoq` is early-stage software. Today it focuses on public GitHub repositories and terminal-based research workflows. The goal is to keep the tool small, predictable, and easy to understand.
