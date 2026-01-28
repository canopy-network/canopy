package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

// GithubRelease represents a GitHub release with the used metadata
type GithubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Release represents a release with metadata on what to update
type Release struct {
	Version       string // version of the release
	DownloadURL   string // url to download the release
	ShouldUpdate  bool   // whether the release should be updated
	ApplySnapshot bool   // whether the release should apply a snapshot (CLI only)
}

// ReleaseType indicates whether this is a CLI or plugin release
type ReleaseType int

const (
	ReleaseTypeCLI ReleaseType = iota
	ReleaseTypePlugin
)

// ReleaseManagerConfig contains configuration for the release manager
type ReleaseManagerConfig struct {
	Type           ReleaseType // type of release (CLI or plugin)
	RepoName       string      // name of the repository
	RepoOwner      string      // owner of the repository
	GithubApiToken string      // github api token for authenticated requests
	// CLI-specific fields
	BinPath     string // path to the binary to be updated (CLI only)
	SnapshotKey string // version metadata key for snapshot (CLI only)
	// Plugin-specific fields
	PluginType string // type of plugin: go, typescript, kotlin, csharp, python
	PluginDir  string // path to the plugin directory
}

// ReleaseManager manages the update process for CLI or plugins
type ReleaseManager struct {
	config     *ReleaseManagerConfig
	httpClient *http.Client
	Version    string // current version
}

// NewReleaseManager creates a new ReleaseManager instance
func NewReleaseManager(config *ReleaseManagerConfig, version string) *ReleaseManager {
	return &ReleaseManager{
		config:     config,
		httpClient: &http.Client{Timeout: httpReleaseClientTimeout},
		Version:    version,
	}
}

// Check checks for updates and returns a release if one is available
func (rm *ReleaseManager) Check() (*Release, error) {
	release, err := rm.GetLatestRelease()
	if err != nil {
		return nil, err
	}
	if err := rm.ShouldUpdate(release); err != nil {
		return nil, err
	}
	return release, nil
}

// GetLatestRelease returns the latest release from the GitHub API
func (rm *ReleaseManager) GetLatestRelease() (*Release, error) {
	apiURL, err := url.JoinPath("https://api.github.com", "repos",
		rm.config.RepoOwner, rm.config.RepoName, "releases", "latest")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "canopy-updater/1.0")
	if token := rm.config.GithubApiToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := rm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	var rel GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	// find matching asset
	targetName := rm.getAssetName()
	for _, asset := range rel.Assets {
		if asset.Name == targetName {
			return &Release{
				Version:     rel.TagName,
				DownloadURL: asset.BrowserDownloadURL,
			}, nil
		}
	}
	if rm.config.Type == ReleaseTypeCLI {
		return nil, fmt.Errorf("unsupported architecture: %s-%s", runtime.GOOS, runtime.GOARCH)
	}
	return nil, fmt.Errorf("no matching asset found for plugin %s (looking for %s)", rm.config.PluginType, targetName)
}

// getAssetName returns the expected asset name based on release type
func (rm *ReleaseManager) getAssetName() string {
	if rm.config.Type == ReleaseTypeCLI {
		return fmt.Sprintf("cli-%s-%s", runtime.GOOS, runtime.GOARCH)
	}
	// Plugin asset names
	switch rm.config.PluginType {
	case "go":
		return fmt.Sprintf("go-plugin-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	case "typescript":
		return "typescript-plugin.tar.gz"
	case "kotlin":
		return "kotlin-plugin.tar.gz"
	case "csharp":
		arch := runtime.GOARCH
		if arch == "amd64" {
			arch = "x64"
		}
		return fmt.Sprintf("csharp-plugin-%s-%s.tar.gz", runtime.GOOS, arch)
	case "python":
		return "python-plugin.tar.gz"
	default:
		return fmt.Sprintf("%s-plugin.tar.gz", rm.config.PluginType)
	}
}

// ShouldUpdate determines whether the given release should be applied
func (rm *ReleaseManager) ShouldUpdate(release *Release) error {
	if release == nil {
		return fmt.Errorf("release is nil")
	}
	candidateTag := release.Version
	// for plugins, extract version from prefixed tags like "plugin-go-v1.0.0"
	if rm.config.Type == ReleaseTypePlugin && strings.Contains(candidateTag, "-v") {
		parts := strings.Split(candidateTag, "-v")
		if len(parts) >= 2 {
			candidateTag = "v" + parts[len(parts)-1]
		}
	}
	candidate := semver.Canonical(candidateTag)
	current := semver.Canonical(rm.Version)
	// for plugins, if current version is invalid (first run), always update
	if rm.config.Type == ReleaseTypePlugin && (current == "" || !semver.IsValid(current)) {
		release.ShouldUpdate = true
		release.Version = candidate
		return nil
	}
	// validate versions
	if candidate == "" || !semver.IsValid(candidate) {
		return fmt.Errorf("invalid release version: %s", release.Version)
	}
	if current == "" || !semver.IsValid(current) {
		return fmt.Errorf("invalid local version: %s", rm.Version)
	}
	// should update if candidate version is greater than current
	release.ShouldUpdate = semver.Compare(candidate, current) > 0
	if !release.ShouldUpdate {
		return nil
	}
	release.Version = candidate
	// for CLI, check if snapshot should be applied
	if rm.config.Type == ReleaseTypeCLI {
		release.ApplySnapshot = strings.Contains(semver.Build(release.Version), rm.config.SnapshotKey)
	}
	return nil
}

// Download downloads the release asset
func (rm *ReleaseManager) Download(ctx context.Context, release *Release) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, release.DownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "canopy-updater/1.0")
	if token := rm.config.GithubApiToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := rm.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	if rm.config.Type == ReleaseTypeCLI {
		return rm.downloadCLI(resp.Body)
	}
	return rm.downloadPlugin(resp.Body)
}

// downloadCLI saves the CLI binary
func (rm *ReleaseManager) downloadCLI(body io.Reader) error {
	bin, err := SaveToFile(rm.config.BinPath, body, 0755)
	if err != nil {
		return err
	}
	return bin.Close()
}

// downloadPlugin saves the plugin tarball and removes old binary
func (rm *ReleaseManager) downloadPlugin(body io.Reader) error {
	tarballPath := filepath.Join(rm.config.PluginDir, rm.getAssetName())
	file, err := SaveToFile(tarballPath, body, 0644)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	// remove old binary so pluginctl.sh will extract the new tarball
	if oldPath := rm.getOldBinaryPath(); oldPath != "" {
		_ = os.Remove(oldPath)
	}
	return nil
}

// getOldBinaryPath returns the path to the old plugin binary to trigger extraction
func (rm *ReleaseManager) getOldBinaryPath() string {
	if rm.config.Type == ReleaseTypeCLI {
		return ""
	}
	switch rm.config.PluginType {
	case "go":
		return filepath.Join(rm.config.PluginDir, "go-plugin")
	case "kotlin":
		return filepath.Join(rm.config.PluginDir, "build", "libs", "canopy-plugin-kotlin-1.0.0-all.jar")
	case "typescript":
		return filepath.Join(rm.config.PluginDir, "dist", "main.js")
	case "python":
		return filepath.Join(rm.config.PluginDir, "main.py")
	case "csharp":
		return filepath.Join(rm.config.PluginDir, "bin", "CanopyPlugin.dll")
	default:
		return ""
	}
}


// SnapshotManager code below

// SnapshotConfig is the config for the snapshot manager
type SnapshotConfig struct {
	// map[chain ID]URL to download the snapshot
	URLs map[uint64]string
	// file name
	Name string
}

// SnapshotManager is the manager for downloading and installing snapshots
type SnapshotManager struct {
	// snapshot config
	config *SnapshotConfig
	// http client for downloading snapshots
	httpClient *http.Client
}

// NewSnapshotManager creates a new SnapshotManager
func NewSnapshotManager(config *SnapshotConfig) *SnapshotManager {
	return &SnapshotManager{
		config:     config,
		httpClient: &http.Client{Timeout: httpSnapshotClientTimeout},
	}
}

// DownloadAndExtract downloads the snapshot to the specified path and extracts it
func (sm *SnapshotManager) DownloadAndExtract(ctx context.Context, path string, chainID uint64) (err error) {
	// create the snapshot directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}
	defer func() {
		// remove snapshot directory on error
		if err != nil {
			_ = os.RemoveAll(path)
		}
	}()
	// download the snapshot
	snapshot, err := sm.Download(ctx, filepath.Join(path, sm.config.Name), chainID)
	if err != nil {
		return err
	}
	snapshot.Close()
	// always remove the snapshot file after downloading
	defer os.Remove(snapshot.Name())
	// extract the snapshot
	return Extract(ctx, snapshot.Name(), path)
}

// Download downloads the snapshot to the specified path
func (sm *SnapshotManager) Download(ctx context.Context, path string, chainID uint64) (*os.File, error) {
	// check if chain ID exists
	url, ok := sm.config.URLs[chainID]
	if !ok {
		return nil, fmt.Errorf("no snapshot URL found for chain ID %d", chainID)
	}
	// download the snapshot
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "canopy-updater/1.0")
	resp, err := sm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	// save the snapshot to a file
	return SaveToFile(path, resp.Body, 0644)
}

// Replace replaces the snapshot to the specified path, creating a backup
// of the existing files before overwriting them
func (sm *SnapshotManager) Replace(snapshotPath string, dbPath string) (err error) {
	backupPath := dbPath + ".backup"
	// always start from a clean backup state
	_ = os.RemoveAll(backupPath)
	defer func() {
		if err != nil {
			// rollback: try to restore DB and drop snapshot
			_ = os.RemoveAll(dbPath)
			_ = os.Rename(backupPath, dbPath)
			_ = os.RemoveAll(snapshotPath)
			return
		}
		// success: remove backup
		os.RemoveAll(backupPath)
	}()
	// move current DB to backup if it exists
	if err := os.Rename(dbPath, backupPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("rename db->backup failed: %w", err)
		}
	}
	// put snapshot in place as the new DB
	if err := os.Rename(snapshotPath, dbPath); err != nil {
		return fmt.Errorf("rename snapshot->db failed: %w", err)
	}
	return nil
}

// Extract decompresses a tar.gz file using the `tar` command.
// Requires `tar` to be installed and available in the system's PATH
func Extract(ctx context.Context, sourceFile string, targetDir string) error {
	// get absolute paths
	absSource, err := filepath.Abs(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute target path: %w", err)
	}
	// ensure source file exists
	if _, err := os.Stat(absSource); err != nil {
		return fmt.Errorf("source file does not exist: %w", err)
	}
	// ensure target directory exists
	if err := os.MkdirAll(absTarget, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	// use tar with built-in gzip decompression: tar -C target -xzf source
	tarCmd := exec.CommandContext(ctx, "tar", "-C", absTarget, "-xzf", absSource)
	tarCmd.Stderr = os.Stderr
	// run the command
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("tar command failed: %w", err)
	}
	// exit
	return nil
}

// SaveToFile saves the response body to a file with the given path and permissions
func SaveToFile(path string, r io.Reader, perm fs.FileMode) (file *os.File, err error) {
	// ensure destination directory exists
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	// create a temporary file in the same directory to allow atomic rename
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	// cleanup on any error
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()
	// copy data to temporary file
	if _, err = io.Copy(tmp, r); err != nil {
		return nil, err
	}
	// set permissions before rename so perms carry over
	if err = tmp.Chmod(perm); err != nil {
		return nil, err
	}
	// close temp file before rename
	if err = tmp.Close(); err != nil {
		return nil, err
	}
	// atomic replace (same directory ensures same filesystem)
	if err = os.Rename(tmpPath, path); err != nil {
		return nil, err
	}
	// reopen the final file to be able to return it
	file, err = os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}
