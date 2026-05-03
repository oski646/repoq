package ask

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	DefaultSettingsRelativePath = ".repoq/settings.json"
	DefaultCodexModel           = "gpt-5.4-mini"
	DefaultCursorModel          = "composer-2-fast"
)

type Provider string

const (
	ProviderCodex  Provider = "codex"
	ProviderCursor Provider = "cursor"
)

type Settings struct {
	Provider Provider `json:"provider"`
	Model    string   `json:"model"`
}

func DefaultSettings() Settings {
	return Settings{
		Provider: ProviderCodex,
		Model:    DefaultCodexModel,
	}
}

func CursorDefaultSettings() Settings {
	return Settings{
		Provider: ProviderCursor,
		Model:    DefaultCursorModel,
	}
}

func (p Provider) BinaryName() string {
	switch p {
	case ProviderCursor:
		return "cursor"
	default:
		return "codex"
	}
}

func (s Settings) WithDefaults() (Settings, error) {
	provider := Provider(strings.ToLower(strings.TrimSpace(string(s.Provider))))
	if provider == "" {
		provider = ProviderCodex
	}

	switch provider {
	case ProviderCodex:
		if strings.TrimSpace(s.Model) == "" {
			s.Model = DefaultCodexModel
		}
	case ProviderCursor:
		if strings.TrimSpace(s.Model) == "" {
			s.Model = DefaultCursorModel
		}
	default:
		return Settings{}, fmt.Errorf("unsupported provider %q; supported providers are codex and cursor", s.Provider)
	}

	s.Provider = provider
	s.Model = strings.TrimSpace(s.Model)
	return s, nil
}

func (r *CommandRunner) loadSettings() (Settings, error) {
	path, err := r.settingsPath()
	if err != nil {
		return Settings{}, err
	}

	data, err := r.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		settings := r.defaultSettingsForAvailableCLI()
		if err := r.writeDefaultSettings(path, settings); err != nil {
			return Settings{}, err
		}
		return settings, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("read settings %s: %w", path, err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, fmt.Errorf("parse settings %s: %w", path, err)
	}

	return settings.WithDefaults()
}

func (r *CommandRunner) defaultSettingsForAvailableCLI() Settings {
	lookPath := r.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	if _, err := lookPath(ProviderCodex.BinaryName()); err == nil {
		return DefaultSettings()
	}
	if _, err := lookPath(ProviderCursor.BinaryName()); err == nil {
		return CursorDefaultSettings()
	}

	return DefaultSettings()
}

func (r *CommandRunner) writeDefaultSettings(path string, settings Settings) error {
	writeFile := r.WriteFile
	if writeFile == nil {
		writeFile = os.WriteFile
	}

	mkdirAll := r.MkdirAll
	if mkdirAll == nil {
		mkdirAll = os.MkdirAll
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("format default settings: %w", err)
	}
	data = append(data, '\n')

	if err := mkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create settings directory %s: %w", filepath.Dir(path), err)
	}
	if err := writeFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write default settings %s: %w", path, err)
	}

	return nil
}

func (r *CommandRunner) settingsPath() (string, error) {
	if strings.TrimSpace(r.SettingsPath) != "" {
		return r.SettingsPath, nil
	}

	userHomeDir := r.UserHomeDir
	if userHomeDir == nil {
		userHomeDir = os.UserHomeDir
	}

	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("find user home directory: %w", err)
	}

	return filepath.Join(home, DefaultSettingsRelativePath), nil
}
