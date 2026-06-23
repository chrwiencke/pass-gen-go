package updater

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCheckGitHubReleaseFindsNewerAsset(t *testing.T) {
	client := fakeClient(func(r *http.Request) *http.Response {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{
			"tag_name": "v1.2.0",
			"assets": [
				{"name": "gopass-darwin-arm64", "browser_download_url": "https://example.test/gopass-darwin-arm64", "state": "uploaded"},
				{"name": "gopass-darwin-arm64.sha256", "browser_download_url": "https://example.test/gopass-darwin-arm64.sha256", "state": "uploaded"}
			]
		}`)
	})

	update, err := CheckGitHubRelease(context.Background(), Config{
		Owner:          "owner",
		Repo:           "repo",
		CurrentVersion: "1.1.0",
		BaseURL:        "https://api.example.test",
		TargetOS:       "darwin",
		TargetArch:     "arm64",
		HTTPClient:     client,
	})
	if err != nil {
		t.Fatal(err)
	}
	if update == nil {
		t.Fatal("expected an available update")
	}
	if update.Version != "v1.2.0" {
		t.Fatalf("unexpected version: %s", update.Version)
	}
	if update.AssetName != "gopass-darwin-arm64" {
		t.Fatalf("unexpected asset: %s", update.AssetName)
	}
	if !strings.HasSuffix(update.ChecksumURL, ".sha256") {
		t.Fatalf("expected checksum URL, got %q", update.ChecksumURL)
	}
}

func TestCheckGitHubReleaseHidesWhenNotNewer(t *testing.T) {
	client := fakeClient(func(r *http.Request) *http.Response {
		return jsonResponse(http.StatusOK, `{
			"tag_name": "v1.2.0",
			"assets": [
				{"name": "gopass-darwin-arm64", "browser_download_url": "https://example.test/gopass-darwin-arm64", "state": "uploaded"}
			]
		}`)
	})

	update, err := CheckGitHubRelease(context.Background(), Config{
		Owner:          "owner",
		Repo:           "repo",
		CurrentVersion: "1.2.0",
		BaseURL:        "https://api.example.test",
		TargetOS:       "darwin",
		TargetArch:     "arm64",
		HTTPClient:     client,
	})
	if err != nil {
		t.Fatal(err)
	}
	if update != nil {
		t.Fatalf("expected no update, got %+v", update)
	}
}

func TestCheckGitHubReleaseSkipsDevelopmentVersion(t *testing.T) {
	called := false
	client := fakeClient(func(r *http.Request) *http.Response {
		called = true
		return jsonResponse(http.StatusOK, `{}`)
	})

	update, err := CheckGitHubRelease(context.Background(), Config{
		Owner:          "owner",
		Repo:           "repo",
		CurrentVersion: "dev",
		BaseURL:        "https://api.example.test",
		TargetOS:       "darwin",
		TargetArch:     "arm64",
		HTTPClient:     client,
	})
	if err != nil {
		t.Fatal(err)
	}
	if update != nil {
		t.Fatalf("expected no update, got %+v", update)
	}
	if called {
		t.Fatal("development version should not call GitHub")
	}
}

func TestReleaseAssetName(t *testing.T) {
	tests := []struct {
		goos string
		arch string
		want string
	}{
		{"darwin", "arm64", "gopass-darwin-arm64"},
		{"darwin", "amd64", "gopass-darwin-amd64"},
		{"windows", "amd64", "gopass-windows-amd64.exe"},
	}

	for _, tt := range tests {
		if got := ReleaseAssetName(tt.goos, tt.arch); got != tt.want {
			t.Fatalf("ReleaseAssetName(%q, %q) = %q, want %q", tt.goos, tt.arch, got, tt.want)
		}
	}
}

func TestParseSHA256Checksum(t *testing.T) {
	const sum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	got, err := parseSHA256Checksum(sum+"  gopass-darwin-arm64\n", "gopass-darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 32 {
		t.Fatalf("expected 32 checksum bytes, got %d", len(got))
	}
}

type roundTripFunc func(*http.Request) *http.Response

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req), nil
}

func fakeClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
