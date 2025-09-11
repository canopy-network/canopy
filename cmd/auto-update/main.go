// Package main implements an auto-update mechanism for the Canopy CLI application.
// It periodically checks for new releases and automatically updates the binary when available.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/canopy-network/canopy/cmd/cli"
	"github.com/canopy-network/canopy/cmd/rpc"
	"github.com/canopy-network/canopy/lib"
)

// constants defining the GitHub repository information
const (
	repoName         = "canopy"
	snapshotFilename = "snapshot.tar.gz"
	waitTime         = 30 * time.Second
)

var ()

// global variables for managing the auto-update process
var (
	binPath   = envOrDefault("BIN_PATH", "./cli")
	repoOwner = envOrDefault("REPO_OWNER", "rem1niscence")

	config lib.Config

	cmd *exec.Cmd // Pointer to the running CLI process, used to control and monitor the main application
	err error     // Standard error variable for capturing and handling errors throughout the process

	// Atomic flags for thread-safe state management
	uploadingNewVersion    atomic.Bool // Indicates if a new version is currently being installed
	newVersionAlreadyFound atomic.Bool // Prevents multiple goroutines from handling the same update
	killedFromChild        atomic.Bool // Indicates if a kill signal was sent from the child process

	downloadLock = new(sync.Mutex)     // Mutex to prevent concurrent binary downloads and replacements
	curRelease   = rpc.SoftwareVersion // Current version of the software for comparison

	// Channels for process coordination
	notifyStartRun = make(chan struct{}, 1)  // Signals when to start the CLI process
	notifyEndRun   = make(chan struct{}, 1)  // Signals when the CLI process has ended
	stop           = make(chan os.Signal, 1) // Signals stop in auto update to do graceful shutdown

	// snapshotURLs contains the snapshot map for existing chains
	snapshotURLs = map[uint64]string{
		1: "http://canopy-mainnet-latest-chain-id1.us.nodefleet.net",
		2: "http://canopy-mainnet-latest-chain-id2.us.nodefleet.net",
	}
	// snapshotVersion dictates which versions should download a new snahpshot
	snapshotVersion = map[string]struct{}{
		"beta-0.1.12": {},
	}
)

// Release represents a GitHub release with all its associated metadata
type Release struct {
	URL       string `json:"url"`
	AssetsURL string `json:"assets_url"`
	UploadURL string `json:"upload_url"`
	HTMLURL   string `json:"html_url"`
	ID        int    `json:"id"`
	Author    struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		UserViewType      string `json:"user_view_type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	NodeID          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Assets          []struct {
		URL      string `json:"url"`
		ID       int    `json:"id"`
		NodeID   string `json:"node_id"`
		Name     string `json:"name"`
		Label    any    `json:"label"`
		Uploader struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			NodeID            string `json:"node_id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			UserViewType      string `json:"user_view_type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
		ContentType        string    `json:"content_type"`
		State              string    `json:"state"`
		Size               int       `json:"size"`
		Digest             any       `json:"digest"`
		DownloadCount      int       `json:"download_count"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		BrowserDownloadURL string    `json:"browser_download_url"`
	} `json:"assets"`
	TarballURL string `json:"tarball_url"`
	ZipballURL string `json:"zipball_url"`
	Body       string `json:"body"`
}

// HandleStart starts the CLI process every time a new version is available
func HandleStart() {
	for {
		// Wait for a signal to start
		<-notifyStartRun

		downloadLock.Lock()
		log.Printf("Starting version: %s in %s from %s...\n", curRelease, binPath, repoOwner)
		// run the new binary
		cmd, err = runBinary()
		if err != nil {
			log.Printf("Failed to run binary: %v", err)
		}
		downloadLock.Unlock()

		// Wait for the child process to exit
		err = cmd.Wait()
		log.Printf("Process exited: %v", err)
		if !uploadingNewVersion.Load() {
			killedFromChild.Store(true)
			stop <- syscall.SIGINT
		}

		// Notify that the process has ended
		notifyEndRun <- struct{}{}
	}
}

// HandleUpdateCheck checks for updates and starts the update process if a new version is available
func HandleUpdateCheck() {
	for i := 0; ; i++ {
		version, url, err := getLatestRelease()
		if err != nil {
			log.Printf("Failed get latest release from %s: %v", repoOwner, err)
			log.Println("Checking for upgrade in 30m... [1]")
			time.Sleep(waitTime)
			continue
		}
		if curRelease != version {
			log.Println("NEW VERSION FOUND")
			err := downloadRelease(url, downloadLock)
			if err != nil {
				log.Printf("Failed to download release from %s: %v", repoOwner, err)
				log.Println("Checking for upgrade in 30m... [2]")
				time.Sleep(waitTime)
				continue
			}
			curRelease = version

			log.Println("NEW DOWNLOAD DONE")

			if !newVersionAlreadyFound.Load() {
				newVersionAlreadyFound.Store(true)
				go func() {
					defer newVersionAlreadyFound.Store(false)

					if i != 0 {
						// Add random delay between 1-30 minutes before updating
						minutes := rand.Intn(30) + 1
						duration := time.Duration(minutes) * time.Second

						log.Printf("Waiting for %v before uploading...\n", duration)
						time.Sleep(duration)
						log.Println("Done waiting.")
					}

					if cmd != nil {
						uploadingNewVersion.Store(true)
						defer uploadingNewVersion.Store(false)

						// Gracefully terminate the current process
						err = cmd.Process.Signal(syscall.SIGINT)
						if err != nil {
							log.Printf("Failed to send syscall to child process in routine of check updates: %v", err)
							return // use of return instead of continue here is correct since this routine is short lived
						}

						log.Println("SENT KILL SIGNAL in routine of check updates")
						<-notifyEndRun
						log.Println("KILLED OLD PROCESS in routine of check updates")
					}

					// check whether the new version requires a snapshot update
					if _, ok := snapshotVersion[curRelease]; ok {
						log.Printf("New version %s requires snapshot update, installing...", curRelease)
						// Perform snapshot update logic here
						if err := HandleNewSnapshot(config); err != nil {
							log.Printf("Failed to install snapshot: %v", err)
						} else {
							log.Printf("Successfully installed new snapshot at version %s", curRelease)
						}
					}

					notifyStartRun <- struct{}{}
				}()
			}
		}

		log.Println("Checking for upgrade in 30m... [3]")
		time.Sleep(waitTime)
	}
}

// HandleKill gracefully waits and terminates the current process
func HandleKill() {
	for {
		signal.Notify(stop, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGABRT)
		// block until kill signal is received
		s := <-stop
		log.Printf("Exit command %s received in auto update\n", s)
		if !killedFromChild.Load() {
			// Gracefully terminate the current process
			err = cmd.Process.Signal(syscall.SIGINT)
			if err != nil {
				killedFromChild.Store(false) // in case of error when a new kill signal comes it is not necessarily because of child process kill
				log.Printf("Failed to send syscall to child process in routine wait for kill: %v", err)
				continue
			}
			log.Println("SENT KILL SIGNAL in routine wait for kill")
			<-notifyEndRun
			log.Println("KILLED OLD PROCESS in routine wait for kill")

		}
		log.Println("Finished auto update setup for closure")
		os.Exit(0)
	}
}

func runAutoUpdate(config lib.Config) {
	if binPath == "" {
		binPath = "./cli"
	}
	if repoOwner == "" {
		repoOwner = "rem1niscence"
	}

	if !config.AutoUpdate {
		cli.Start()
	} else {
		// goroutine to handle binary execution and restart
		go HandleStart()
		// goroutine to check for updates periodically
		go HandleUpdateCheck()
		// goroutine to wait for kill
		go HandleKill()
		// start the initial binary if it exists
		_, err = os.Stat(binPath)
		if err == nil {
			notifyStartRun <- struct{}{}
		}
	}
}

// main is the entry point of the application
// It handles both manual and auto-update modes based on configuration
func main() {
	// check if start was called or just waken up for setup
	if len(os.Args) < 2 || os.Args[1] != "start" {
		log.Println("Set up complete! Now you can start!")
		return
	}

	// Initialize data directory and configuration
	err := os.MkdirAll(cli.DataDir, os.ModePerm)
	if err != nil {
		log.Print(err.Error())
	}
	configFilePath := filepath.Join(cli.DataDir, lib.ConfigFilePath)
	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		log.Printf("Creating %s file", lib.ConfigFilePath)
		if err = lib.DefaultConfig().WriteToFile(configFilePath); err != nil {
			log.Print(err.Error())
		}
	}
	// Load configuration
	config, err = lib.NewConfigFromFile(configFilePath)
	if err != nil {
		log.Print(err.Error())
	}
	// Run auto update
	go runAutoUpdate(config)
	// Block forever
	select {}
}

// getLatestRelease fetches the latest release information from GitHub
// Returns the tag name, download URL for the current platform, and any error
func getLatestRelease() (string, string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		var rel Release
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			return "", "", err
		}

		// Find asset matching OS and ARCH
		targetName := "cli" + "-" + runtime.GOOS + "-" + runtime.GOARCH
		for _, asset := range rel.Assets {
			if asset.Name == targetName {
				return rel.TagName, asset.BrowserDownloadURL, nil
			}
		}
		return "", "", io.EOF
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("HTTP %d: failed to read response body", resp.StatusCode)
	}
	return "", "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
}

// downloadRelease downloads and replaces the current binary with the new version
// Uses a mutex to prevent concurrent downloads
func downloadRelease(downloadURL string, downloadLock *sync.Mutex) error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		downloadLock.Lock()
		defer downloadLock.Unlock()

		err = os.Remove(binPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		out, err := os.Create(binPath)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, resp.Body); err != nil {
			return err
		}

		err = os.Chmod(binPath, 0755)
		if err != nil {
			return err
		}

		return nil
	}

	// Read response body for error message
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read response body", resp.StatusCode)
	}
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
}

// runBinary executes the CLI binary with the 'start' command
// Returns the command and any error that occurred during startup
func runBinary() (*exec.Cmd, error) {
	cmd = exec.Command(binPath, "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func HandleNewSnapshot(config lib.Config) error {
	// check if a snapshot is available for the chain ID
	url, ok := snapshotURLs[config.ChainId]
	if !ok {
		return fmt.Errorf("no snapshot available for chain ID %d", config.ChainId)
	}
	// remove any previous dangling backup
	dbDir := filepath.Join(cli.DataDir, config.DBName)
	backupDBDir := dbDir + ".backup"
	if err := os.RemoveAll(backupDBDir); err != nil {
		return err
	}
	// rename the current database directory as a backup
	if err := os.Rename(dbDir, backupDBDir); err != nil {
		return err
	}
	log.Printf("backup canopy folder done")
	defer func() {
		// restore the backup if an error occurs
		if err != nil {
			log.Printf("snapshot install failed, restoring backup")
			os.Rename(dbDir, backupDBDir)
			return
		}
		// otherwise, delete the backup
		os.RemoveAll(backupDBDir)
	}()
	var err error
	// create a temporary file to store the snapshot
	path, file, err := createFile(filepath.Join(cli.DataDir, config.DBName, snapshotFilename))
	if err != nil {
		return err
	}
	defer file.Close()
	// download the snapshot file
	log.Printf("downloading snapshot file...")
	if err := downloadToFile(file, url); err != nil {
		return err
	}
	defer func() {
		// delete the snapshot file
		if err := os.Remove(path); err != nil {
			log.Printf("failed to remove snapshot file: %v", err)
		}
	}()
	log.Printf("snapshot file downloaded")
	// decompress the snapshot file in the same directory
	log.Printf("decompressing snapshot file...")
	if err := decompressCMD(context.Background(), path, filepath.Dir(path)); err != nil {
		return err
	}
	log.Printf("decompressed snapshot file")
	return nil
}

func createFile(path string) (string, *os.File, error) {
	// create all intermediate directories
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", nil, err
	}
	// create the file
	out, err := os.Create(path)
	if err != nil {
		return "", nil, err
	}
	return path, out, nil
}

func downloadToFile(f *os.File, url string) error {
	// download the file
	resp, err := http.Get(url)
	// check for error
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// write the body to file
	_, err = io.Copy(f, resp.Body)
	return err
}

// envOrDefault returns the value of the environment variable with the given key,
// or the default value if the variable is not set.
func envOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// decompressCMD decompresses a tar.gz file using pigz + tar
func decompressCMD(ctx context.Context, sourceFile, targetDir string) error {
	// get absolute paths
	absSource, err := filepath.Abs(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute target path: %w", err)
	}

	// ensure target directory exists
	if err := os.MkdirAll(absTarget, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// construct the commands: pigz -dc source | tar -C target -xv
	pigzCmd := exec.CommandContext(ctx, "pigz", "-dc", absSource)
	tarCmd := exec.CommandContext(ctx, "tar", "-C", absTarget, "-xv")

	// create the pipeline
	pipe, err := pigzCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	tarCmd.Stdin = pipe
	pigzCmd.Stderr = os.Stderr
	tarCmd.Stderr = os.Stderr

	// start both commands
	if err := tarCmd.Start(); err != nil {
		return fmt.Errorf("failed to start tar: %w", err)
	}

	if err := pigzCmd.Start(); err != nil {
		return fmt.Errorf("failed to start pigz: %w", err)
	}

	// wait for completion
	if err := pigzCmd.Wait(); err != nil {
		return fmt.Errorf("pigz command failed: %w", err)
	}

	pipe.Close()

	if err := tarCmd.Wait(); err != nil {
		return fmt.Errorf("tar command failed: %w", err)
	}

	return nil
}
