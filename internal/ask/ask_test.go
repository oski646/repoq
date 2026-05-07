package ask

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCheckDependencies(t *testing.T) {
	t.Parallel()

	err := CheckDependencies(func(name string) (string, error) {
		return "/bin/" + name, nil
	}, ProviderCodex)
	if err != nil {
		t.Fatalf("expected dependencies to pass, got %v", err)
	}
}

func TestCheckDependenciesMissingBinary(t *testing.T) {
	t.Parallel()

	err := CheckDependencies(func(name string) (string, error) {
		if name == "codex" {
			return "", os.ErrNotExist
		}
		return "/bin/" + name, nil
	}, ProviderCodex)
	if err == nil || !strings.Contains(err.Error(), "codex is not installed") {
		t.Fatalf("expected codex missing error, got %v", err)
	}
}

func TestNormalizeGitHubRepo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Repository
		wantErr string
	}{
		{
			name:  "owner repo",
			input: "openai/codex",
			want:  Repository{Owner: "openai", Name: "codex"},
		},
		{
			name:  "https url",
			input: "https://github.com/openai/codex",
			want:  Repository{Owner: "openai", Name: "codex"},
		},
		{
			name:  "git suffix",
			input: "https://github.com/openai/codex.git",
			want:  Repository{Owner: "openai", Name: "codex"},
		},
		{
			name:    "non github",
			input:   "https://gitlab.com/openai/codex",
			wantErr: "only github.com repositories are supported",
		},
		{
			name:    "invalid shape",
			input:   "openai/codex/extra",
			wantErr: "repository must be owner/repo or a full GitHub URL",
		},
		{
			name:    "dot owner segment",
			input:   "./codex",
			wantErr: "repository owner and name must be safe path segments",
		},
		{
			name:    "dotdot repo segment",
			input:   "openai/..",
			wantErr: "repository owner and name must be safe path segments",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeGitHubRepo(tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected repo: got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestCachePath(t *testing.T) {
	t.Parallel()

	repo := Repository{Owner: "openai", Name: "codex"}
	if got, want := CachePath("/tmp/repoq/repos", repo, ""), "/tmp/repoq/repos/openai/codex/default"; got != want {
		t.Fatalf("unexpected default cache path: got %s want %s", got, want)
	}
	if got, want := CachePath("/tmp/repoq/repos", repo, "feature/test"), "/tmp/repoq/repos/openai/codex/feature%2Ftest"; got != want {
		t.Fatalf("unexpected ref cache path: got %s want %s", got, want)
	}
	if got, want := CachePath("/tmp/repoq/repos", repo, ".."), "/tmp/repoq/repos/openai/codex/_.."; got != want {
		t.Fatalf("unexpected unsafe ref cache path: got %s want %s", got, want)
	}
}

func TestBuildCloneArgs(t *testing.T) {
	t.Parallel()

	repo := Repository{Owner: "openai", Name: "codex"}
	args := BuildCloneArgs(repo, "main", "/tmp/cache")
	got := strings.Join(args, " ")
	want := "clone --depth 1 --branch main git@github.com:openai/codex.git /tmp/cache"
	if got != want {
		t.Fatalf("unexpected clone args: got %q want %q", got, want)
	}
}

func TestBuildCodexArgs(t *testing.T) {
	t.Parallel()

	args := BuildCodexArgs("where is auth?", PromptContext{
		RepositoryURL: "https://github.com/openai/codex",
		RequestedRef:  "main",
		Commit:        "abc123",
	}, "/tmp/output.txt", "Prefer a short answer.", DefaultCodexModel)
	got := strings.Join(args[:8], " ")
	want := "exec -m gpt-5.4-mini --sandbox read-only --ephemeral --color never"
	if got != want {
		t.Fatalf("unexpected codex prefix args: got %q want %q", got, want)
	}
	if args[8] != "--output-last-message" || args[9] != "/tmp/output.txt" {
		t.Fatalf("unexpected output args: %+v", args)
	}
	if !strings.Contains(args[10], "where is auth?") {
		t.Fatalf("question missing from prompt: %q", args[10])
	}
	if !strings.Contains(args[10], "Answer the user's question by inspecting this repository.") {
		t.Fatalf("base prompt missing: %q", args[10])
	}
	if !strings.Contains(args[10], "https://github.com/openai/codex/blob/abc123/<path-from-repo-root>") {
		t.Fatalf("blob url guidance missing: %q", args[10])
	}
	if !strings.Contains(args[10], "Prefer a short answer.") {
		t.Fatalf("additional instructions missing: %q", args[10])
	}
}

func TestBuildCursorArgs(t *testing.T) {
	t.Parallel()

	args := BuildCursorArgs("where is auth?", PromptContext{
		RepositoryURL: "https://github.com/openai/codex",
		RequestedRef:  "main",
		Commit:        "abc123",
	}, "Prefer a short answer.", DefaultCursorModel, "/tmp/cache")

	got := strings.Join(args[:9], " ")
	want := "agent --print --output-format text --mode ask --trust --workspace /tmp/cache"
	if got != want {
		t.Fatalf("unexpected cursor prefix args: got %q want %q", got, want)
	}
	if args[9] != "--model" || args[10] != "composer-2-fast" {
		t.Fatalf("unexpected model args: %+v", args)
	}
	if strings.Contains(strings.Join(args, " "), "--sandbox") {
		t.Fatalf("cursor args should use cursor sandbox config, got %+v", args)
	}
	if !strings.Contains(args[11], "where is auth?") {
		t.Fatalf("question missing from prompt: %q", args[11])
	}
	if !strings.Contains(args[11], "Prefer a short answer.") {
		t.Fatalf("additional instructions missing: %q", args[11])
	}
}

func TestSettingsDefaults(t *testing.T) {
	t.Parallel()

	settings, err := (Settings{Provider: ProviderCursor}).WithDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.Provider != ProviderCursor || settings.Model != DefaultCursorModel {
		t.Fatalf("unexpected cursor defaults: %+v", settings)
	}

	settings, err = (Settings{}).WithDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.Provider != ProviderCodex || settings.Model != DefaultCodexModel {
		t.Fatalf("unexpected codex defaults: %+v", settings)
	}
}

func TestSettingsRejectsUnsupportedProvider(t *testing.T) {
	t.Parallel()

	_, err := (Settings{Provider: Provider("other")}).WithDefaults()
	if err == nil || !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}

func TestLoadSettingsCreatesDefaultFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := NewRunner()
	runner.UserHomeDir = func() (string, error) { return tempDir, nil }
	runner.LookPath = func(name string) (string, error) {
		if name == "codex" {
			return "/bin/codex", nil
		}
		return "", os.ErrNotExist
	}

	settings, err := runner.loadSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings != DefaultSettings() {
		t.Fatalf("unexpected settings: got %+v want %+v", settings, DefaultSettings())
	}

	settingsPath := filepath.Join(tempDir, DefaultSettingsRelativePath)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read created settings: %v", err)
	}
	want := "{\n  \"provider\": \"codex\",\n  \"model\": \"gpt-5.4-mini\"\n}\n"
	if string(data) != want {
		t.Fatalf("unexpected settings file:\n%s", string(data))
	}
}

func TestLoadSettingsCreatesCursorDefaultWhenOnlyCursorExists(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := NewRunner()
	runner.UserHomeDir = func() (string, error) { return tempDir, nil }
	runner.LookPath = func(name string) (string, error) {
		if name == "cursor" {
			return "/bin/cursor", nil
		}
		return "", os.ErrNotExist
	}

	settings, err := runner.loadSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings != CursorDefaultSettings() {
		t.Fatalf("unexpected settings: got %+v want %+v", settings, CursorDefaultSettings())
	}

	settingsPath := filepath.Join(tempDir, DefaultSettingsRelativePath)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read created settings: %v", err)
	}
	want := "{\n  \"provider\": \"cursor\",\n  \"model\": \"composer-2-fast\"\n}\n"
	if string(data) != want {
		t.Fatalf("unexpected settings file:\n%s", string(data))
	}
}

func TestLoadSettingsPrefersCodexWhenBothCLIsExist(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := NewRunner()
	runner.UserHomeDir = func() (string, error) { return tempDir, nil }
	runner.LookPath = func(name string) (string, error) {
		if name == "codex" || name == "cursor" {
			return "/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	settings, err := runner.loadSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings != DefaultSettings() {
		t.Fatalf("unexpected settings: got %+v want %+v", settings, DefaultSettings())
	}
}

func TestRunnerIntegration(t *testing.T) {
	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	logDir := filepath.Join(tempDir, "logs")
	cacheRoot := filepath.Join(tempDir, "cache")

	for _, dir := range []string{binDir, logDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	writeExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
printf '%s\n' "$1" >> "$FAKE_LOG_DIR/git.log"
if [ "$1" = "rev-parse" ] && [ "$2" = "HEAD" ]; then
	printf '%s\n' "${FAKE_GIT_COMMIT}"
	exit 0
fi
dest=""
for arg in "$@"; do
	dest="$arg"
done
mkdir -p "$dest/.git"
`)

	writeExecutable(t, filepath.Join(binDir, "codex"), `#!/bin/sh
printf 'codex\n' >> "$FAKE_LOG_DIR/codex.log"
prev=""
out=""
for arg in "$@"; do
	if [ "$prev" = "--output-last-message" ]; then
		out="$arg"
	fi
	prev="$arg"
done
if [ -z "$out" ]; then
	echo "missing output path" >&2
	exit 1
fi
printf '%s\n' "${FAKE_CODEX_ANSWER}" > "$out"
`)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_LOG_DIR", logDir)
	t.Setenv("FAKE_CODEX_ANSWER", "mocked answer")
	t.Setenv("FAKE_GIT_COMMIT", "abc123")

	runner := &CommandRunner{
		CacheRoot:         cacheRoot,
		HeartbeatInterval: 5 * time.Second,
		LookPath:          execLookPathForTest,
		Command:           exec.CommandContext,
		CreateTemp:        os.CreateTemp,
		ReadFile:          os.ReadFile,
		WriteFile:         os.WriteFile,
		MkdirAll:          os.MkdirAll,
		Stat:              os.Stat,
		Remove:            os.Remove,
		UserHomeDir:       func() (string, error) { return tempDir, nil },
	}

	for i := 0; i < 2; i++ {
		answer, err := runner.Run(context.Background(), Options{
			Repository: "openai/codex",
			Question:   "Where is auth?",
			Stderr:     ioDiscard{},
		})
		if err != nil {
			t.Fatalf("run %d failed: %v", i+1, err)
		}
		if answer != "mocked answer" {
			t.Fatalf("unexpected answer on run %d: %q", i+1, answer)
		}
	}

	if lines := readLogLines(t, filepath.Join(logDir, "git.log")); strings.Join(lines, ",") != "clone,rev-parse,rev-parse" {
		t.Fatalf("unexpected git invocations: %v", lines)
	}
	if lines := readLogLines(t, filepath.Join(logDir, "codex.log")); len(lines) != 2 {
		t.Fatalf("expected 2 codex invocations, got %d", len(lines))
	}
}

func TestRunnerUsesCursorProviderFromSettings(t *testing.T) {
	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	logDir := filepath.Join(tempDir, "logs")
	cacheRoot := filepath.Join(tempDir, "cache")
	settingsPath := filepath.Join(tempDir, ".repoq", "settings.json")

	for _, dir := range []string{binDir, logDir, filepath.Dir(settingsPath)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(settingsPath, []byte(`{"provider":"cursor"}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	writeExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
if [ "$1" = "rev-parse" ] && [ "$2" = "HEAD" ]; then
	printf '%s\n' "${FAKE_GIT_COMMIT}"
	exit 0
fi
dest=""
for arg in "$@"; do
	dest="$arg"
done
mkdir -p "$dest/.git"
`)

	writeExecutable(t, filepath.Join(binDir, "cursor"), `#!/bin/sh
printf '%s\n' "$*" >> "$FAKE_LOG_DIR/cursor.log"
printf '%s\n' "${FAKE_CURSOR_ANSWER}"
`)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_LOG_DIR", logDir)
	t.Setenv("FAKE_CURSOR_ANSWER", "cursor answer")
	t.Setenv("FAKE_GIT_COMMIT", "abc123")

	runner := &CommandRunner{
		CacheRoot:         cacheRoot,
		HeartbeatInterval: 5 * time.Second,
		LookPath:          execLookPathForTest,
		Command:           exec.CommandContext,
		CreateTemp:        os.CreateTemp,
		ReadFile:          os.ReadFile,
		WriteFile:         os.WriteFile,
		MkdirAll:          os.MkdirAll,
		Stat:              os.Stat,
		Remove:            os.Remove,
		UserHomeDir:       func() (string, error) { return tempDir, nil },
	}

	answer, err := runner.Run(context.Background(), Options{
		Repository: "openai/codex",
		Question:   "Where is auth?",
		Stderr:     ioDiscard{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "cursor answer" {
		t.Fatalf("unexpected answer: %q", answer)
	}

	logData, err := os.ReadFile(filepath.Join(logDir, "cursor.log"))
	if err != nil {
		t.Fatalf("read cursor log: %v", err)
	}
	if !strings.Contains(string(logData), "--model composer-2-fast") {
		t.Fatalf("expected default cursor model, got %q", string(logData))
	}
}

func TestRunnerWritesProgressMessages(t *testing.T) {
	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	logDir := filepath.Join(tempDir, "logs")
	cacheRoot := filepath.Join(tempDir, "cache")

	for _, dir := range []string{binDir, logDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	writeExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
if [ "$1" = "rev-parse" ] && [ "$2" = "HEAD" ]; then
	printf '%s\n' "${FAKE_GIT_COMMIT}"
	exit 0
fi
dest=""
for arg in "$@"; do
	dest="$arg"
done
mkdir -p "$dest/.git"
`)

	writeExecutable(t, filepath.Join(binDir, "codex"), `#!/bin/sh
prev=""
out=""
for arg in "$@"; do
	if [ "$prev" = "--output-last-message" ]; then
		out="$arg"
	fi
	prev="$arg"
done
sleep "${FAKE_CODEX_DELAY}"
printf '%s\n' "${FAKE_CODEX_ANSWER}" > "$out"
`)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GIT_COMMIT", "abc123")
	t.Setenv("FAKE_CODEX_DELAY", "0.12")
	t.Setenv("FAKE_CODEX_ANSWER", "mocked answer")
	t.Setenv("FAKE_LOG_DIR", logDir)

	stderr := &lockedBuffer{}
	runner := &CommandRunner{
		CacheRoot:         cacheRoot,
		HeartbeatInterval: 50 * time.Millisecond,
		LookPath:          execLookPathForTest,
		Command:           exec.CommandContext,
		CreateTemp:        os.CreateTemp,
		ReadFile:          os.ReadFile,
		WriteFile:         os.WriteFile,
		MkdirAll:          os.MkdirAll,
		Stat:              os.Stat,
		Remove:            os.Remove,
		UserHomeDir:       func() (string, error) { return tempDir, nil },
	}

	answer, err := runner.Run(context.Background(), Options{
		Repository: "openai/codex",
		Question:   "Where is auth?",
		Stderr:     stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "mocked answer" {
		t.Fatalf("unexpected answer: %q", answer)
	}

	progress := stderr.String()
	for _, expected := range []string{
		"preparing repository git@github.com:openai/codex.git",
		"repository ready: openai/codex at default branch (abc123)",
		"starting codex (gpt-5.4-mini) analysis",
		"still analyzing...",
		"codex analysis finished",
	} {
		if !strings.Contains(progress, expected) {
			t.Fatalf("expected progress output to contain %q, got:\n%s", expected, progress)
		}
	}
}

func TestRunnerMissingGit(t *testing.T) {
	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	writeExecutable(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir)

	runner := NewRunner()
	runner.CacheRoot = filepath.Join(tempDir, "cache")
	runner.UserHomeDir = func() (string, error) { return tempDir, nil }

	_, err := runner.Run(context.Background(), Options{
		Repository: "openai/codex",
		Question:   "Where is auth?",
		Stderr:     ioDiscard{},
	})
	if err == nil || !strings.Contains(err.Error(), "git is not installed") {
		t.Fatalf("expected missing git error, got %v", err)
	}
}

func TestRunnerMissingCodex(t *testing.T) {
	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	writeExecutable(t, filepath.Join(binDir, "git"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir)

	runner := NewRunner()
	runner.CacheRoot = filepath.Join(tempDir, "cache")
	runner.UserHomeDir = func() (string, error) { return tempDir, nil }

	_, err := runner.Run(context.Background(), Options{
		Repository: "openai/codex",
		Question:   "Where is auth?",
		Stderr:     ioDiscard{},
	})
	if err == nil || !strings.Contains(err.Error(), "codex is not installed") {
		t.Fatalf("expected missing codex error, got %v", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func readLogLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log %s: %v", path, err)
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}

	return strings.Split(trimmed, "\n")
}

func execLookPathForTest(file string) (string, error) {
	return exec.LookPath(file)
}

type lockedBuffer struct {
	mu sync.Mutex
	sb strings.Builder
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.String()
}
