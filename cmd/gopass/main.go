package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopass/internal/clipboard"
	"gopass/internal/password"

	"fyne.io/systray"
	"github.com/minio/selfupdate"
)

const appName = "GoPass"

// Change this to your real GitHub repo, for example:
// const updateRepository = "cwiencke/gopass"
const updateRepository = "chrwiencke/pass-gen-go"

// Override this in your build script:
//
//	go build -trimpath -ldflags="-s -w -X main.version=1.0.0"
var version = "0.1.0"

var (
	updateMu       sync.Mutex
	updateMenuItem *systray.MenuItem
	pendingUpdate  *availableUpdate
)

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type availableUpdate struct {
	Version       string
	Asset         githubAsset
	ChecksumAsset *githubAsset
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	systray.Run(onReady, onExit)
}

func onReady() {
	iconPNG := makeKeyIconPNG()

	systray.SetIcon(iconPNG)
	systray.SetTemplateIcon(iconPNG, iconPNG)
	systray.SetTitle(appName)
	systray.SetTooltip(appName + ": left-click to copy a Norwegian password")

	copyItem := systray.AddMenuItem("Copy password", "Copy a Norwegian password")

	updateMenuItem = systray.AddMenuItem("Update available", "Install the latest version")
	updateMenuItem.Hide()

	versionItem := systray.AddMenuItem("Version "+version, "Current version")
	versionItem.Disable()

	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Quit "+appName)

	// Left-clicking the macOS menu-bar icon or Windows taskbar tray icon copies a new password.
	systray.SetOnTapped(func() {
		copyPassword()
	})

	go func() {
		for range copyItem.ClickedCh {
			copyPassword()
		}
	}()

	go func() {
		for range updateMenuItem.ClickedCh {
			go installPendingUpdate()
		}
	}()

	go func() {
		<-quitItem.ClickedCh
		systray.Quit()
	}()

	// Check for updates in the background, but do not install automatically.
	// The update button only appears if a newer compatible release exists.
	go func() {
		time.Sleep(3 * time.Second)
		checkForUpdateAndShowButton()
	}()
}

func onExit() {
	// Nothing to clean up.
}

func copyPassword() {
	pw, err := password.Generate()
	if err != nil {
		systray.SetTooltip(appName + ": could not generate password")
		log.Printf("password generation failed: %v", err)
		return
	}

	if err := clipboard.CopyText(pw); err != nil {
		systray.SetTooltip(appName + ": clipboard is unavailable")
		log.Printf("clipboard copy failed: %v", err)
		return
	}

	systray.SetTooltip(fmt.Sprintf(appName+": password copied at %s", time.Now().Format("15:04:05")))

	// Do not log, display, or notify the actual password. It is only written to the clipboard.
}

func checkForUpdateAndShowButton() {
	updateMu.Lock()
	defer updateMu.Unlock()

	pendingUpdate = nil
	updateMenuItem.Hide()

	if strings.Contains(updateRepository, "YOUR_GITHUB_USERNAME") || strings.TrimSpace(updateRepository) == "" {
		log.Print("update repository is not configured")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	update, err := findLatestCompatibleUpdate(ctx)
	if err != nil {
		log.Printf("update check failed: %v", err)
		return
	}

	if update == nil {
		log.Printf("no update available: current=%s", version)
		return
	}

	pendingUpdate = update
	updateMenuItem.SetTitle("Update to " + update.Version)
	updateMenuItem.SetTooltip("Install " + appName + " " + update.Version)
	updateMenuItem.Enable()
	updateMenuItem.Show()

	systray.SetTooltip(appName + ": update " + update.Version + " available")
	log.Printf("update available: current=%s latest=%s asset=%s", version, update.Version, update.Asset.Name)
}

func findLatestCompatibleUpdate(ctx context.Context) (*availableUpdate, error) {
	release, err := fetchLatestGitHubRelease(ctx)
	if err != nil {
		return nil, err
	}

	if release.Draft || release.Prerelease {
		return nil, nil
	}

	if !isNewerVersion(release.TagName, version) {
		return nil, nil
	}

	asset, ok := findBinaryAsset(release.Assets)
	if !ok {
		return nil, fmt.Errorf("no compatible release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	checksumAsset := findChecksumAsset(release.Assets, asset.Name)

	return &availableUpdate{
		Version:       release.TagName,
		Asset:         asset,
		ChecksumAsset: checksumAsset,
	}, nil
}

func fetchLatestGitHubRelease(ctx context.Context) (*githubRelease, error) {
	url := "https://api.github.com/repos/" + updateRepository + "/releases/latest"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", appName+" updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("latest GitHub release not found")
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("GitHub release check failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func findBinaryAsset(assets []githubAsset) (githubAsset, bool) {
	goos := strings.ToLower(runtime.GOOS)
	goarch := strings.ToLower(runtime.GOARCH)

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)

		if !strings.Contains(name, goos) || !strings.Contains(name, goarch) {
			continue
		}

		// minio/selfupdate.Apply expects the actual binary bytes.
		// Do not pass zip/tar/signature/checksum files here.
		if strings.HasSuffix(name, ".zip") ||
			strings.HasSuffix(name, ".tar.gz") ||
			strings.HasSuffix(name, ".tgz") ||
			strings.HasSuffix(name, ".sha256") ||
			strings.HasSuffix(name, ".sha256sum") ||
			strings.HasSuffix(name, ".sig") ||
			strings.HasSuffix(name, ".minisig") {
			continue
		}

		if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
			continue
		}

		return asset, true
	}

	return githubAsset{}, false
}

func findChecksumAsset(assets []githubAsset, binaryAssetName string) *githubAsset {
	want1 := strings.ToLower(binaryAssetName + ".sha256")
	want2 := strings.ToLower(binaryAssetName + ".sha256sum")

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if name == want1 || name == want2 {
			copyAsset := asset
			return &copyAsset
		}
	}

	return nil
}

func installPendingUpdate() {
	updateMu.Lock()
	update := pendingUpdate
	updateMu.Unlock()

	if update == nil {
		updateMenuItem.Hide()
		return
	}

	updateMenuItem.Disable()
	updateMenuItem.SetTitle("Updating to " + update.Version + "...")
	systray.SetTooltip(appName + ": downloading update...")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	opts := selfupdate.Options{}
	if err := opts.CheckPermissions(); err != nil {
		updateFailed(update, fmt.Errorf("no permission to update executable: %w", err))
		return
	}

	checksum, err := downloadOptionalChecksum(ctx, update.ChecksumAsset)
	if err != nil {
		updateFailed(update, err)
		return
	}

	if checksum != nil {
		opts.Checksum = checksum
	}

	resp, err := downloadUpdateBinary(ctx, update.Asset.BrowserDownloadURL)
	if err != nil {
		updateFailed(update, err)
		return
	}
	defer resp.Body.Close()

	systray.SetTooltip(appName + ": installing update...")

	if err := selfupdate.Apply(resp.Body, opts); err != nil {
		if rollbackErr := selfupdate.RollbackError(err); rollbackErr != nil {
			log.Printf("rollback also failed: %v", rollbackErr)
		}

		updateFailed(update, err)
		return
	}

	systray.SetTooltip(appName + ": updated to " + update.Version + ", restarting...")
	log.Printf("updated from %s to %s", version, update.Version)

	if err := restartApp(); err != nil {
		updateMenuItem.SetTitle("Updated - restart manually")
		updateMenuItem.SetTooltip("Update installed, but automatic restart failed")
		updateMenuItem.Disable()

		systray.SetTooltip(appName + ": updated; restart manually")
		log.Printf("updated, but restart failed: %v", err)
	}
}

func updateFailed(update *availableUpdate, err error) {
	log.Printf("update failed: %v", err)

	updateMenuItem.SetTitle("Update to " + update.Version)
	updateMenuItem.SetTooltip("Install " + appName + " " + update.Version)
	updateMenuItem.Enable()
	updateMenuItem.Show()

	systray.SetTooltip(appName + ": update failed")
}

func downloadUpdateBinary(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", appName+" updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("update download failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return resp, nil
}

func downloadOptionalChecksum(ctx context.Context, checksumAsset *githubAsset) ([]byte, error) {
	if checksumAsset == nil {
		return nil, nil
	}

	resp, err := downloadUpdateBinary(ctx, checksumAsset.BrowserDownloadURL)
	if err != nil {
		return nil, fmt.Errorf("checksum download failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return nil, errors.New("checksum file is empty")
	}

	checksumHex := strings.TrimSpace(fields[0])
	checksum, err := hex.DecodeString(checksumHex)
	if err != nil {
		return nil, fmt.Errorf("invalid checksum file: %w", err)
	}

	return checksum, nil
}

func restartApp() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	var cmd *exec.Cmd

	if runtime.GOOS == "darwin" {
		if bundlePath, ok := macOSAppBundlePath(exePath); ok {
			cmd = exec.Command("open", bundlePath)
		} else {
			cmd = exec.Command(exePath)
		}
	} else {
		cmd = exec.Command(exePath)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start updated app: %w", err)
	}

	systray.Quit()
	os.Exit(0)
	return nil
}

func macOSAppBundlePath(exePath string) (string, bool) {
	marker := ".app/Contents/MacOS/"
	index := strings.Index(exePath, marker)
	if index == -1 {
		return "", false
	}

	return exePath[:index+len(".app")], true
}

func isNewerVersion(remoteVersion string, currentVersion string) bool {
	remote, err := parseSemver(remoteVersion)
	if err != nil {
		log.Printf("could not parse remote version %q: %v", remoteVersion, err)
		return false
	}

	current, err := parseSemver(currentVersion)
	if err != nil {
		log.Printf("could not parse current version %q: %v", currentVersion, err)
		return false
	}

	if remote.major != current.major {
		return remote.major > current.major
	}

	if remote.minor != current.minor {
		return remote.minor > current.minor
	}

	return remote.patch > current.patch
}

type semver struct {
	major int
	minor int
	patch int
}

func parseSemver(value string) (semver, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")

	if index := strings.IndexAny(value, "-+"); index >= 0 {
		value = value[:index]
	}

	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("expected major.minor.patch")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semver{}, err
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semver{}, err
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return semver{}, err
	}

	return semver{
		major: major,
		minor: minor,
		patch: patch,
	}, nil
}

func makeKeyIconPNG() []byte {
	const size = 32

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	transparent := color.RGBA{0, 0, 0, 0}
	draw.Draw(img, img.Bounds(), &image.Uniform{transparent}, image.Point{}, draw.Src)

	// Pixel-art key tray icon. PNG is accepted by fyne.io/systray on macOS and Windows.
	key := color.RGBA{245, 190, 60, 255}
	shadow := color.RGBA{132, 88, 0, 255}

	// Slight shadow/outline first, then the key shape on top.
	drawFilledCircle(img, 11, 16, 8, shadow)
	drawFilledCircle(img, 11, 16, 4, transparent)
	draw.Draw(img, image.Rect(17, 14, 30, 19), &image.Uniform{shadow}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(23, 18, 27, 24), &image.Uniform{shadow}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(27, 18, 31, 22), &image.Uniform{shadow}, image.Point{}, draw.Src)

	drawFilledCircle(img, 10, 15, 8, key)
	drawFilledCircle(img, 10, 15, 4, transparent)
	draw.Draw(img, image.Rect(16, 13, 29, 18), &image.Uniform{key}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(22, 17, 26, 23), &image.Uniform{key}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(26, 17, 30, 21), &image.Uniform{key}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func drawFilledCircle(img *image.RGBA, centerX, centerY, radius int, c color.RGBA) {
	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			dx := x - centerX
			dy := y - centerY
			if dx*dx+dy*dy <= radius*radius && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}
