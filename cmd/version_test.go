package cmd

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRootCommandPrintsVersion(t *testing.T) {
	previousURL := latestVersionURL
	previousClient := latestVersionClient
	latestVersionURL = "https://example.test/tags"
	latestVersionClient = &http.Client{Transport: httpClientFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != latestVersionURL {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}

		return jsonResponse(`[{"name":"v0.1.0"}]`), nil
	})}
	t.Cleanup(func() {
		latestVersionURL = previousURL
		latestVersionClient = previousClient
	})

	cmd := NewRootCmd(stubRunner{})
	cmd.SetArgs([]string{"--version"})

	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	for _, expected := range []string{
		"repoq ",
		"Latest: v0.1.0",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected version output to contain %q, got %q", expected, output)
		}
	}
}

func TestLatestVersionChoosesHighestSemverTag(t *testing.T) {
	client := httpClientFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(`[
			{"name":"v0.2.0"},
			{"name":"v0.10.0"},
			{"name":"not-a-version"},
			{"name":"v0.1.9"}
		]`), nil
	})

	previousURL := latestVersionURL
	latestVersionURL = "https://example.test/tags"
	t.Cleanup(func() {
		latestVersionURL = previousURL
	})

	got, err := latestVersion(t.Context(), &http.Client{Transport: client})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v0.10.0" {
		t.Fatalf("unexpected latest version: %s", got)
	}
}

func TestVersionTextShowsInstallForDevelopmentBuild(t *testing.T) {
	previousClient := latestVersionClient
	latestVersionClient = &http.Client{Transport: httpClientFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(`[{"name":"v0.1.0"}]`), nil
	})}
	t.Cleanup(func() {
		latestVersionClient = previousClient
	})

	output := versionText(t.Context())
	if !strings.Contains(output, "Install: go install github.com/oski646/repoq@latest") {
		t.Fatalf("expected development build install hint, got %q", output)
	}
	if strings.Contains(output, "Upgrade:") {
		t.Fatalf("did not expect upgrade hint for development build, got %q", output)
	}
}

func TestCompareSemverTag(t *testing.T) {
	t.Parallel()

	if compareSemverTag("v0.10.0", "v0.2.0") <= 0 {
		t.Fatal("expected v0.10.0 to be newer than v0.2.0")
	}
	if compareSemverTag("v1.0.0", "v1.0.0") != 0 {
		t.Fatal("expected equal versions to compare equally")
	}
}

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
