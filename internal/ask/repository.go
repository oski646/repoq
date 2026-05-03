package ask

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type Repository struct {
	Owner string
	Name  string
}

func NormalizeGitHubRepo(input string) (Repository, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return Repository{}, errors.New("repository must not be empty")
	}

	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return Repository{}, fmt.Errorf("parse repository URL: %w", err)
		}
		if !strings.EqualFold(parsed.Host, "github.com") {
			return Repository{}, errors.New("only github.com repositories are supported")
		}
		value = strings.Trim(parsed.Path, "/")
	}

	value = strings.TrimSuffix(value, ".git")
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return Repository{}, errors.New("repository must be owner/repo or a full GitHub URL")
	}

	if parts[0] == "" || parts[1] == "" {
		return Repository{}, errors.New("repository owner and name must not be empty")
	}
	if !safePathSegment(parts[0]) || !safePathSegment(parts[1]) {
		return Repository{}, errors.New("repository owner and name must be safe path segments")
	}

	return Repository{
		Owner: parts[0],
		Name:  parts[1],
	}, nil
}

func CachePath(root string, repo Repository, ref string) string {
	return filepath.Join(root, repo.Owner, repo.Name, cacheKey(ref))
}

func BuildCloneArgs(repo Repository, ref, destination string) []string {
	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, repo.CloneURL(), destination)

	return args
}

func (r Repository) CloneURL() string {
	return fmt.Sprintf("https://github.com/%s/%s.git", r.Owner, r.Name)
}

func cacheKey(ref string) string {
	if strings.TrimSpace(ref) == "" {
		return "default"
	}

	return safeCacheKey(strings.TrimSpace(ref))
}

func safeCacheKey(value string) string {
	escaped := url.PathEscape(value)
	if !safePathSegment(escaped) {
		return "_" + escaped
	}

	return escaped
}

func safePathSegment(value string) bool {
	return value != "" &&
		value != "." &&
		value != ".." &&
		!strings.ContainsAny(value, `/\`)
}
