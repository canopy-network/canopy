package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/canopy-network/canopy/cmd/cli"
	"github.com/canopy-network/canopy/cmd/rpc"
	"github.com/canopy-network/canopy/lib"
)

const (
	repoOwner = "pablocampogo"
	repoName  = "canopy"
	binPath   = "./cli"
)

var (
	autoUpdate    bool
	rawAutoUpdate = os.Getenv("AUTO_UPDATE")
)

func init() {
	// is safe to ignore the error and assume false
	autoUpdate, _ = strconv.ParseBool(rawAutoUpdate)
}

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
		URL      string      `json:"url"`
		ID       int         `json:"id"`
		NodeID   string      `json:"node_id"`
		Name     string      `json:"name"`
		Label    interface{} `json:"label"`
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
		ContentType        string      `json:"content_type"`
		State              string      `json:"state"`
		Size               int         `json:"size"`
		Digest             interface{} `json:"digest"`
		DownloadCount      int         `json:"download_count"`
		CreatedAt          time.Time   `json:"created_at"`
		UpdatedAt          time.Time   `json:"updated_at"`
		BrowserDownloadURL string      `json:"browser_download_url"`
	} `json:"assets"`
	TarballURL string `json:"tarball_url"`
	ZipballURL string `json:"zipball_url"`
	Body       string `json:"body"`
}

func getLatestRelease() (string, string, error) {
	// return "v0.0.5", "https://github.com/pablocampogo/canopy/releases/download/v0.0.5/cli", nil
	apiURL := "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
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

	return "", "", errors.New("NON 200 OK")
}

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
		if err != nil {
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

	return errors.New("NON 200 OK")
}

func runBinary() (*exec.Cmd, error) {
	cmd := exec.Command(binPath, "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func main() {
	// make the data dir if missing
	if err := os.MkdirAll(cli.DataDir, os.ModePerm); err != nil {
		log.Print(err.Error())
	}
	configFilePath := filepath.Join(cli.DataDir, lib.ConfigFilePath)
	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		log.Printf("Creating %s file", lib.ConfigFilePath)
		if err = lib.DefaultConfig().WriteToFile(configFilePath); err != nil {
			log.Print(err.Error())
		}
	}
	// load the config object
	config, err := lib.NewConfigFromFile(configFilePath)
	if err != nil {
		log.Print(err.Error())
	}
	if !config.AutoUpdate {
		cli.Start()
	} else {
		var cmd *exec.Cmd
		var err error
		var uploadingNewVersion atomic.Bool
		var newVersionAlreadyFound atomic.Bool
		firstTime := true
		downloadLock := new(sync.Mutex)
		curRelease := rpc.SoftwareVersion
		notifyStartRun := make(chan struct{}, 1)
		notifyEndRun := make(chan struct{}, 1)
		go func() {
			for {
				// Wait for a signal to start
				<-notifyStartRun

				downloadLock.Lock()
				log.Printf("Starting %s...\n", binPath)
				cmd, err = runBinary()
				if err != nil {
					log.Printf("Failed to run binary: %v", err)
				}
				downloadLock.Unlock()

				// Wait for the child process to exit
				err = cmd.Wait()
				log.Printf("Process exited: %v", err)
				if !uploadingNewVersion.Load() {
					os.Exit(1)
				}

				// Notify that the process has ended
				notifyEndRun <- struct{}{}
			}
		}()
		go func() {
			for {
				version, url, err := getLatestRelease()
				if err != nil {
					log.Printf("Failed get latest release: %v", err)
					continue
				}

				if curRelease != version {
					log.Println("NEW VERSION FOUND")
					err := downloadRelease(url, downloadLock)
					if err != nil {
						log.Printf("Failed to download release: %v", err)
						continue
					}
					curRelease = version

					log.Println("NEW DOWNLOAD DONE")

					if !newVersionAlreadyFound.Load() {
						newVersionAlreadyFound.Store(true)
						go func() {
							defer newVersionAlreadyFound.Store(false)

							if !firstTime {
								minutes := rand.Intn(30) + 1 // rand.Intn(30) returns [0, 29] so add 1 to get [1, 30]
								duration := time.Duration(minutes) * time.Minute

								log.Printf("Waiting for %v before uploading...\n", duration)

								// Sleep for the random duration
								time.Sleep(duration)

								log.Println("Done waiting.")
							} else {
								firstTime = false
							}

							if cmd != nil {
								uploadingNewVersion.Store(true)
								defer uploadingNewVersion.Store(false)

								err = cmd.Process.Signal(syscall.SIGINT)
								if err != nil {
									log.Printf("Failed to send syscall to child process: %v", err)
									return
								}

								log.Println("SENT KILL SIGNAL")

								<-notifyEndRun

								log.Println("KILLED OLD PROCESS")
							}

							notifyStartRun <- struct{}{}
						}()
					}
				}

				log.Println("Checking for upgrade in 5s...")
				time.Sleep(5 * time.Second)
			}
		}()
		_, err = os.Stat(binPath)
		if err == nil {
			notifyStartRun <- struct{}{}
		}
		// Block forever
		select {}
	}

}
