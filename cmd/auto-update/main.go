package main

import (
	"fmt"
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
	defaultMaxDelayTime = 30               // default max delay time for staggered updates (minutes)
)

var (
	// snapshotURLs contains the snapshot map for existing chains
	snapshotURLs = map[uint64]string{
		1: envOrDefault("SNAPSHOT_1_URL", "http://canopy-mainnet-latest-chain-id1.us.nodefleet.net"),
		2: envOrDefault("SNAPSHOT_2_URL", "http://canopy-mainnet-latest-chain-id2.us.nodefleet.net"),
	}
	// pluginReleaseConfigs maps plugin type to its full release configuration
	pluginReleaseConfigs = map[string]*PluginReleaseConfig{
		"go": {
			AssetName:      "go-plugin-%s-%s.tar.gz",
			ArchSpecific:   true,
			OldBinaryPath:  "go-plugin",
			ProcessPattern: "go-plugin",
			PIDFile:        "/tmp/plugin/go-plugin.pid",
		},
		"kotlin": {
			AssetName:      "kotlin-plugin.tar.gz",
			ArchSpecific:   false,
			OldBinaryPath:  "build/libs/canopy-plugin-kotlin-1.0.0-all.jar",
			ProcessPattern: "canopy-plugin-kotlin",
			PIDFile:        "/tmp/plugin/kotlin-plugin.pid",
		},
		"typescript": {
			AssetName:      "typescript-plugin.tar.gz",
			ArchSpecific:   false,
			OldBinaryPath:  "dist/main.js",
			ProcessPattern: "plugin/typescript/dist/main.js",
			PIDFile:        "/tmp/plugin/typescript-plugin.pid",
		},
		"python": {
			AssetName:      "python-plugin.tar.gz",
			ArchSpecific:   false,
			OldBinaryPath:  "main.py",
			ProcessPattern: "plugin/python/main.py",
			PIDFile:        "/tmp/plugin/python-plugin.pid",
		},
		"csharp": {
			AssetName:      "csharp-plugin-%s-%s.tar.gz",
			MuslAssetName:  "csharp-plugin-%s-musl-%s.tar.gz",
			ArchSpecific:   true,
			UseX64Arch:     true,
			OldBinaryPath:  "bin/CanopyPlugin",
			ProcessPattern: "plugin/csharp/bin/CanopyPlugin",
			PIDFile:        "/tmp/plugin/csharp-plugin.pid",
		},
	}
)

func main() {
	// get configs and logger
	configs, logger := getConfigs()
	// check if no start was called, this means it was just called as config
	if len(os.Args) < 2 || os.Args[1] != "start" {
		// TODO: This message is partly misleading due to the fact that the only place that it would
		// make sense to have a setup complete message is on the context of the deployments repository.
		// The actual behavior of this program should be to only start the CLI directly, not perform
		// any kind of setup or initialization.
		logger.Info("setup complete.\nKey configuration is ready.\n" +
			"From now on, run this service using the 'start' command to launch Canopy with the auto-updater.\n" +
			"This message appears because the program was started directly instead of using 'start'.")
		return
	}
	// do not run the auto-update process if its disabled
	if !configs.Coordinator.Canopy.AutoUpdate {
		logger.Info("auto-update disabled, starting CLI directly")
		cli.Start()
		return
	}
	logger.Infof("auto-update enabled, starting coordinator on version %s", rpc.SoftwareVersion)
	// handle external shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// setup the dependencies
	updater := NewReleaseManager(configs.Updater, rpc.SoftwareVersion)
	snapshot := NewSnapshotManager(configs.Snapshot)

	// setup plugin updater and config if configured
	var pluginUpdater *ReleaseManager
	var pluginConfig *PluginReleaseConfig
	if configs.PluginUpdater != nil {
		pluginUpdater = NewReleaseManager(configs.PluginUpdater, "v0.0.0")
		pluginConfig = configs.PluginUpdater.PluginConfig
		logger.Infof("plugin auto-update enabled from %s/%s",
			configs.PluginUpdater.RepoOwner,
			configs.PluginUpdater.RepoName)
	}
	supervisor := NewSupervisor(logger, pluginConfig)

	coordinator := NewCoordinator(configs.Coordinator, updater, pluginUpdater, supervisor, snapshot, logger)
	// start the update loop
	err := coordinator.UpdateLoop(sigChan)
	if err != nil {
		exitCode, exitCodeStr := 1, "unknown"
		// try to extract the exit code from the error
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
			exitCodeStr = fmt.Sprintf("%d", exitCode)
		}
		// log the error
		logger.Errorf("canopy stopped with error: %v, exit code: %s", err, exitCodeStr)
		// exit
		os.Exit(exitCode)
	}
}

// Configs holds the configuration for the updater, snapshotter, and process supervisor.
type Configs struct {
	Updater       *ReleaseManagerConfig
	PluginUpdater *ReleaseManagerConfig
	Snapshot      *SnapshotConfig
	Coordinator   *CoordinatorConfig
	LoggerI       lib.LoggerI
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
	githubToken := envOrDefault("CANOPY_GITHUB_API_TOKEN", "")

	updater := &ReleaseManagerConfig{
		Type:           ReleaseTypeCLI,
		RepoName:       envOrDefault("REPO_NAME", defaultRepoName),
		RepoOwner:      envOrDefault("REPO_OWNER", defaultRepoOwner),
		GithubApiToken: githubToken,
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
		MaxDelayTime: envOrDefaultInt("AUTO_UPDATE_MAX_DELAY_MINUTES", defaultMaxDelayTime),
		CheckPeriod:  envOrDefaultDuration("AUTO_UPDATE_CHECK_PERIOD", defaultCheckPeriod),
		GracePeriod:  defaultGracePeriod,
	}

	// setup plugin updater config if plugin auto-update is enabled
	var pluginUpdater *ReleaseManagerConfig
	pluginAutoUpdate := canopyConfig.PluginAutoUpdate
	if pluginAutoUpdate.Enabled && canopyConfig.Plugin != "" {
		// lookup plugin release config from map
		pluginReleaseCfg, ok := pluginReleaseConfigs[canopyConfig.Plugin]
		if !ok {
			l.Warnf("unknown plugin type %q, plugin auto-update disabled", canopyConfig.Plugin)
		} else {
			// use configured repo or default to canopy-network/canopy
			repoOwner := pluginAutoUpdate.RepoOwner
			if repoOwner == "" {
				repoOwner = defaultRepoOwner
			}
			repoName := pluginAutoUpdate.RepoName
			if repoName == "" {
				repoName = defaultRepoName
			}
			pluginUpdater = &ReleaseManagerConfig{
				Type:           ReleaseTypePlugin,
				RepoOwner:      repoOwner,
				RepoName:       repoName,
				PluginDir:      fmt.Sprintf("plugin/%s", canopyConfig.Plugin),
				PluginConfig:   pluginReleaseCfg,
				GithubApiToken: githubToken,
			}
		}
	}

	return &Configs{
		Updater:       updater,
		PluginUpdater: pluginUpdater,
		Snapshot:      snapshot,
		Coordinator:   coordinator,
		LoggerI:       l,
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

// envOrDefaultInt returns the value of the environment variable as an int,
// or the default value if the variable is not set or invalid.
func envOrDefaultInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

// envOrDefaultDuration returns the value of the environment variable as a duration,
// or the default value if the variable is not set or invalid.
// Accepts formats like "30m", "1h", "30s", etc.
func envOrDefaultDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	result, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return result
}
