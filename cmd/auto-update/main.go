package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/canopy-network/canopy/cmd/cli"
	"github.com/canopy-network/canopy/cmd/rpc"
	"github.com/canopy-network/canopy/lib"
)

const (
	snapshotFileName    = "snapshot.tar.gz"
	snapshotMetadataKey = "snapshot"

	httpReleaseClientTimeout  = 30 * time.Second
	httpSnapshotClientTimeout = 10 * time.Minute

	// program defaults
	defaultRepoName     = "canopy"
	defaultRepoOwner    = "canopy-network"
	defaultBinPath      = "./cli"
	defaultCheckPeriod  = time.Minute * 30 // default check period for updates
	defaultGracePeriod  = time.Second * 2  // default grace period for graceful shutdown
	defaultMaxDelayTime = 30               // default max delay time for staggered updates
)

var (
	// snapshotURLs contains the snapshot map for existing chains
	snapshotURLs = map[uint64]string{
		1: "http://canopy-mainnet-latest-chain-id1.us.nodefleet.net",
		2: "http://canopy-mainnet-latest-chain-id2.us.nodefleet.net",
	}
)

func main() {
	// check if start was called
	if len(os.Args) < 2 || os.Args[1] != "start" {
		log.Fatalf("invalid input, only `start` command is allowed")
	}
	// get configs and logger
	configs, logger := getConfigs()
	// do not run the auto-update process if its disabled
	if !configs.Coordinator.Canopy.AutoUpdate {
		logger.Info("auto-update disabled, starting CLI directly")
		cli.Start()
		return
	}
	logger.Info("auto-update enabled, starting coordinator")
	// handle external shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// setup the dependencies
	updater := NewUpdateManager(configs.Updater, rpc.SoftwareVersion)
	snapshot := NewSnapshotManager(configs.Snapshot)
	supervisor := NewSupervisor(logger)
	coordinator := NewCoordinator(configs.Coordinator, updater, supervisor, snapshot, logger)
	// start the update loop
	err := coordinator.UpdateLoop(sigChan)
	if err != nil {
		logger.Errorf("canopy stopped with error: %v", err)
		// try to extract the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		// default of 1 for unknown errors
		os.Exit(1)
	}
}

// Configs holds the configuration for the updater, snapshotter, and process supervisor.
type Configs struct {
	Updater     *UpdaterConfig
	Snapshot    *SnapshotConfig
	Coordinator *CoordinatorConfig
	LoggerI     lib.LoggerI
}

// getConfigs returns the configuration for the updater, snapshotter, and process supervisor.
func getConfigs() (*Configs, lib.LoggerI) {
	canopyConfig, _ := cli.InitializeDataDirectory(cli.DataDir, lib.NewDefaultLogger())
	l := lib.NewLogger(lib.LoggerConfig{
		Level:      canopyConfig.GetLogLevel(),
		Structured: canopyConfig.Structured,
		JSON:       canopyConfig.JSON,
	})

	binPath := envOrDefault("BIN_PATH", defaultBinPath)

	updater := &UpdaterConfig{
		RepoName:       envOrDefault("REPO_NAME", defaultRepoName),
		RepoOwner:      envOrDefault("REPO_OWNER", defaultRepoOwner),
		GithubApiToken: envOrDefault("CANOPY_GITHUB_API_TOKEN", ""),
		BinPath:        binPath,
		SnapshotKey:    snapshotMetadataKey,
	}
	snapshot := &SnapshotConfig{
		URLs: snapshotURLs,
		Name: snapshotFileName,
	}
	coordinator := &CoordinatorConfig{
		Canopy:       canopyConfig,
		BinPath:      binPath,
		MaxDelayTime: defaultMaxDelayTime,
		CheckPeriod:  defaultCheckPeriod,
		GracePeriod:  defaultGracePeriod,
	}

	return &Configs{
		Updater:     updater,
		Snapshot:    snapshot,
		Coordinator: coordinator,
		LoggerI:     l,
	}, l
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
