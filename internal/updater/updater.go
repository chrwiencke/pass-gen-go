package updater

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/minio/selfupdate"
)

const (
	defaultGitHubAPIBaseURL = "https://api.github.com"
	userAgent               = "GoPass self-updater"
)

// Config describes where update metadata is published and which build is running.
type Config struct {
	Owner          string
	Repo           string
	CurrentVersion string
	BaseURL        string
	TargetOS       string
	TargetArch     string
	HTTPClient     *http.Client
}

// AvailableUpdate is the release asset that can replace the current executable.
type AvailableUpdate struct {
	Version     string
	AssetName   string
	DownloadURL string
	ChecksumURL string
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	State              string `json:"state"`
}

// CheckGitHubRelease returns nil when the latest GitHub release is not newer or
// when this binary was built with a development version.
func CheckGitHubRelease(ctx context.Context, cfg Config) (*AvailableUpdate, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if isDevelopmentVersion(cfg.CurrentVersion) {
		return nil, nil
	}

	current, ok := parseVersion(cfg.CurrentVersion)
	if !ok {
		return nil, nil
	}

	release, err := fetchLatestRelease(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, nil
	}

	latestName := strings.TrimSpace(release.TagName)
	if latestName == "" {
		latestName = strings.TrimSpace(release.Name)
	}
	latest, ok := parseVersion(latestName)
	if !ok {
		return nil, fmt.Errorf("latest release tag %q is not a semantic version", latestName)
	}
	if compareVersions(latest, current) <= 0 {
		return nil, nil
	}

	targetOS := cfg.TargetOS
	if targetOS == "" {
		targetOS = runtime.GOOS
	}
	targetArch := cfg.TargetArch
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}

	assetName := ReleaseAssetName(targetOS, targetArch)
	asset := findAsset(release.Assets, assetName)
	if asset == nil {
		return nil, fmt.Errorf("latest release %s does not include asset %q", latestName, assetName)
	}

	checksumURL := ""
	if checksumAsset := findAsset(release.Assets, assetName+".sha256"); checksumAsset != nil {
		checksumURL = checksumAsset.BrowserDownloadURL
	}

	return &AvailableUpdate{
		Version:     latestName,
		AssetName:   asset.Name,
		DownloadURL: asset.BrowserDownloadURL,
		ChecksumURL: checksumURL,
	}, nil
}

// ApplyGitHubRelease downloads and applies the selected release asset to the
// running executable. The updated binary is used on the next launch.
func ApplyGitHubRelease(ctx context.Context, update AvailableUpdate, client *http.Client) error {
	if strings.TrimSpace(update.DownloadURL) == "" {
		return errors.New("update download URL is empty")
	}
	if strings.TrimSpace(update.AssetName) == "" {
		return errors.New("update asset name is empty")
	}
	if client == nil {
		client = http.DefaultClient
	}

	checksum, err := downloadChecksum(ctx, client, update)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, update.DownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download update asset: unexpected HTTP status %s", resp.Status)
	}

	targetPath, err := executablePath()
	if err != nil {
		return err
	}

	opts := selfupdate.Options{
		TargetPath: targetPath,
		Checksum:   checksum,
	}
	if err := selfupdate.Apply(resp.Body, opts); err != nil {
		if rollbackErr := selfupdate.RollbackError(err); rollbackErr != nil {
			return fmt.Errorf("apply update failed and rollback failed: %w", rollbackErr)
		}
		return err
	}

	return finalizeUpdate(targetPath)
}

// ReleaseAssetName returns the exact GitHub release asset name this updater expects.
func ReleaseAssetName(goos, goarch string) string {
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("gopass-%s-%s%s", goos, goarch, ext)
}

func (cfg Config) validate() error {
	if strings.TrimSpace(cfg.Owner) == "" {
		return errors.New("GitHub owner is empty")
	}
	if strings.TrimSpace(cfg.Repo) == "" {
		return errors.New("GitHub repo is empty")
	}
	return nil
}

func fetchLatestRelease(ctx context.Context, cfg Config) (*githubRelease, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultGitHubAPIBaseURL
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases/latest",
		baseURL,
		neturl.PathEscape(cfg.Owner),
		neturl.PathEscape(cfg.Repo),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", userAgent)

	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("check latest release: unexpected HTTP status %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func downloadChecksum(ctx context.Context, client *http.Client, update AvailableUpdate) ([]byte, error) {
	if strings.TrimSpace(update.ChecksumURL) == "" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, update.ChecksumURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download checksum asset: unexpected HTTP status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, err
	}
	return parseSHA256Checksum(string(body), update.AssetName)
}

func parseSHA256Checksum(text, assetName string) ([]byte, error) {
	fields := strings.Fields(text)
	for i, field := range fields {
		token := strings.TrimPrefix(strings.TrimSpace(field), "*")
		if len(token) != 64 || !isHex(token) {
			continue
		}
		if assetName == "" || i == len(fields)-1 || strings.TrimPrefix(fields[i+1], "*") == assetName {
			return hex.DecodeString(token)
		}
	}
	return nil, fmt.Errorf("checksum file does not contain a SHA-256 sum for %q", assetName)
}

func isHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func findAsset(assets []githubAsset, name string) *githubAsset {
	for i := range assets {
		if assets[i].Name == name && assets[i].BrowserDownloadURL != "" && (assets[i].State == "" || assets[i].State == "uploaded") {
			return &assets[i]
		}
	}
	return nil
}

func executablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return normalizeExecutablePath(path), nil
}

func normalizeExecutablePath(path string) string {
	path = filepath.Clean(path)
	base := filepath.Base(path)
	if filepath.Ext(base) == ".old" {
		base = strings.TrimSuffix(base, ".old")
		base = strings.TrimPrefix(base, ".")
		path = filepath.Join(filepath.Dir(path), base)
	}
	return path
}

func isDevelopmentVersion(version string) bool {
	version = strings.ToLower(strings.TrimSpace(version))
	return version == "" || version == "dev" || strings.Contains(version, "dev")
}

type versionNumber struct {
	major      int
	minor      int
	patch      int
	prerelease string
}

func parseVersion(version string) (versionNumber, bool) {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	if version == "" {
		return versionNumber{}, false
	}
	if i := strings.Index(version, "+"); i >= 0 {
		version = version[:i]
	}

	prerelease := ""
	if i := strings.Index(version, "-"); i >= 0 {
		prerelease = version[i+1:]
		version = version[:i]
	}

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return versionNumber{}, false
	}

	major, ok := parseVersionPart(parts[0])
	if !ok {
		return versionNumber{}, false
	}
	minor, ok := parseVersionPart(parts[1])
	if !ok {
		return versionNumber{}, false
	}
	patch, ok := parseVersionPart(parts[2])
	if !ok {
		return versionNumber{}, false
	}

	return versionNumber{
		major:      major,
		minor:      minor,
		patch:      patch,
		prerelease: prerelease,
	}, true
}

func parseVersionPart(part string) (int, bool) {
	if part == "" {
		return 0, false
	}
	value, err := strconv.Atoi(part)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}

func compareVersions(a, b versionNumber) int {
	switch {
	case a.major != b.major:
		return compareInt(a.major, b.major)
	case a.minor != b.minor:
		return compareInt(a.minor, b.minor)
	case a.patch != b.patch:
		return compareInt(a.patch, b.patch)
	case a.prerelease == b.prerelease:
		return 0
	case a.prerelease == "":
		return 1
	case b.prerelease == "":
		return -1
	default:
		return strings.Compare(a.prerelease, b.prerelease)
	}
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
