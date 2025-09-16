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

	"github.com/canopy-network/canopy/lib"
	"golang.org/x/mod/semver"
)

// GithubRelease represents a GitHub release with all its associated metadata
type GithubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Release represents a release of the current binary with metadata on what to update
type Release struct {
	Version       string // version of the release
	DownloadURL   string // url to download the release
	ShouldUpdate  bool   // whether the release should be updated
	ApplySnapshot bool   // whether the release should apply a snapshot
}

// UpdaterConfig contains configuration for the updater
type UpdaterConfig struct {
	RepoName       string // name of the repository
	RepoOwner      string // owner of the repository
	BinPath        string // path to the binary to be updated
	SnapshotKey    string // version metadata key to know if a snapshot should be applied
	GithubApiToken string // github api token for authenticated requests
}

// UpdateManager manages the update process for the current binary
type UpdateManager struct {
	config     *UpdaterConfig
	httpClient *http.Client
	Version    string
	log        lib.LoggerI
}

// NewUpdateManager creates a new UpdateManager instance
func NewUpdateManager(config *UpdaterConfig, logger lib.LoggerI, version string) *UpdateManager {
	return &UpdateManager{
		config:     config,
		httpClient: &http.Client{Timeout: httpReleaseClientTimeout},
		Version:    version,
		log:        logger,
	}
}

// Check checks for updates of the current binary
func (um *UpdateManager) Check() (*Release, error) {
	// Get the latest release
	release, err := um.GetLatestRelease()
	if err != nil {
		return nil, err
	}
	// Check if the release is valid to update
	if err := um.ShouldUpdate(release); err != nil {
		return nil, err
	}
	// exit
	return release, nil
}

// GetLatestRelease returns the latest valid release for the system from the GitHub API
func (um *UpdateManager) GetLatestRelease() (release *Release, err error) {
	// build the URL: https://api.github.com/repos/<owner>/<repo>/releases/latest
	apiURL, err := url.JoinPath("https://api.github.com", "repos",
		um.config.RepoOwner, um.config.RepoName, "releases", "latest")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	// github recommends to add an user agent to any API request
	req.Header.Set("User-Agent", "canopy-updater/1.0")
	if token := um.config.GithubApiToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// make the request
	resp, err := um.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// check the response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	// parse the response
	var rel GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	// find asset matching OS and ARCH
	targetName := fmt.Sprintf("cli-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range rel.Assets {
		if asset.Name == targetName {
			// match found, stop
			release = &Release{
				Version:     rel.TagName,
				DownloadURL: asset.BrowserDownloadURL,
			}
			break
		}
	}
	// return based on tagName
	if release == nil {
		return nil, fmt.Errorf("unsupported architecture: %s-%s", runtime.GOOS, runtime.GOARCH)
	}
	return release, nil
}

// ShouldUpdate checks if the release should be updated, updating the release
// object with the result
func (um *UpdateManager) ShouldUpdate(release *Release) error {
	if release == nil {
		return fmt.Errorf("release is nil")
	}
	// convert the versions to their canonical form
	candidate := semver.Canonical(release.Version)
	current := semver.Canonical(um.Version)
	// check if the versions are valid
	if candidate == "" || !semver.IsValid(candidate) {
		return fmt.Errorf("invalid release version: %s", release.Version)
	}
	if current == "" || !semver.IsValid(current) {
		return fmt.Errorf("invalid local version: %s", um.Version)
	}
	release.Version = candidate
	// should update if the candidate version is greater than the current version
	release.ShouldUpdate = semver.Compare(candidate, current) > 0
	// should apply snapshot if the candidate version contains the snapshot key
	release.ApplySnapshot = strings.Contains(candidate, um.config.SnapshotKey)
	return nil
}

// Download downloads the release assets into the config bin directory
func (um *UpdateManager) Download(ctx context.Context, release *Release) error {
	// download the release binary
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, release.DownloadURL, nil)
	if err != nil {
		return err
	}
	// github recommends to add an user agent to any API request
	req.Header.Set("User-Agent", "canopy-updater/1.0")
	if token := um.config.GithubApiToken; token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := um.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	// save the response as an executable
	bin, err := SaveToFile(um.config.BinPath, resp.Body, 0755)
	if err != nil {
		return err
	}
	return bin.Close()
}

type SnapshotConfig struct {
	// canopy config
	canopy lib.Config
	// map[chain ID]URL to download the snapshot
	URLs map[uint64]string
	// file name
	Name string
}

// SnapshotManager is the manager for downloading and installing snapshots
type SnapshotManager struct {
	// snapshot config
	config *SnapshotConfig
	// httpClient
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

// Install installs the snapshot to the specified path, creating a backup
// of the existing files before overwriting them
func (sm *SnapshotManager) Install(snapshotPath string, dbPath string) (err error) {
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
	// use tar with built-in gzip decompression: tar -C target -xzvf source
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
