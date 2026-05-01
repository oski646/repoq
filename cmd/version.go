package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const modulePath = "github.com/oski646/repoq"

var (
	latestVersionURL    = "https://api.github.com/repos/oski646/repoq/tags?per_page=100"
	latestVersionClient = http.DefaultClient
)

type githubTag struct {
	Name string `json:"name"`
}

func currentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}

	return info.Main.Version
}

func versionText(ctx context.Context) string {
	current := currentVersion()
	latest, err := latestVersion(ctx, latestVersionClient)
	if err != nil {
		return fmt.Sprintf("repoq %s\nLatest: unavailable (%s)\n", current, err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "repoq %s\n", current)
	fmt.Fprintf(&b, "Latest: %s\n", latest)

	if current == "(devel)" || !validSemverTag(current) {
		fmt.Fprintf(&b, "Install: go install %s@latest\n", modulePath)
		return b.String()
	}
	if compareSemverTag(current, latest) < 0 {
		fmt.Fprintf(&b, "Upgrade: go install %s@latest\n", modulePath)
	}

	return b.String()
}

func latestVersion(ctx context.Context, client *http.Client) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestVersionURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "repoq")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github returned %s", resp.Status)
	}

	var tags []githubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", err
	}

	latest := ""
	for _, tag := range tags {
		if !validSemverTag(tag.Name) {
			continue
		}
		if latest == "" || compareSemverTag(tag.Name, latest) > 0 {
			latest = tag.Name
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no version tags found")
	}

	return latest, nil
}

func validSemverTag(version string) bool {
	_, ok := parseSemverTag(version)
	return ok
}

func compareSemverTag(a, b string) int {
	av, _ := parseSemverTag(a)
	bv, _ := parseSemverTag(b)

	for i := range av {
		if av[i] > bv[i] {
			return 1
		}
		if av[i] < bv[i] {
			return -1
		}
	}

	return 0
}

func parseSemverTag(version string) ([3]int, bool) {
	var parsed [3]int

	value := strings.TrimPrefix(version, "v")
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return parsed, false
	}

	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return parsed, false
		}
		parsed[i] = number
	}

	return parsed, true
}
