package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/canopy-network/canopy/lib"
)

// Supervisor manages the CLI process lifecycle, from start to stop,
// and notifies listeners when the process exits
type Supervisor struct {
	cmd            *exec.Cmd    // canopy sub-process
	mu             sync.RWMutex // mutex for concurrent access
	running        atomic.Bool  // flag indicating if process is running
	stopping       atomic.Bool  // flag indicating if process is stopping
	exit           chan error   // channel to notify listeners when process exits
	unexpectedExit chan error   // channel to notify listeners when process exits unexpectedly
	log            lib.LoggerI  // logger instance
}

// NewSupervisor creates a new ProcessSupervisor instance
func NewSupervisor(logger lib.LoggerI) *Supervisor {
	return &Supervisor{
		log:            logger,
		exit:           make(chan error, 1),
		unexpectedExit: make(chan error, 1),
	}
}

// Start starts the process and runs it until it exits
func (s *Supervisor) Start(binPath string) error {
	// hold the lock to prevent concurrent modifications
	s.mu.Lock()
	defer s.mu.Unlock()
	// check if process is already running
	if s.running.Load() && s.cmd != nil {
		return errors.New("process already running")
	}
	s.log.Infof("starting CLI process: %s", binPath)
	// setup the process to start
	s.cmd = exec.Command(binPath, "start")
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr
	// make sure the process is in a new process group, this is important for
	// ensuring that the process can be terminated by the coordinator
	s.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	// start the process
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start canopy binary: %s", err)
	}
	// set variables for monitoring and exit processing
	s.running.Store(true)
	s.stopping.Store(false)
	// start monitoring the process until it exits
	go s.Monitor()
	return nil
}

// Monitor runs the process while waiting for it to stop
func (s *Supervisor) Monitor() {
	err := s.cmd.Wait()
	s.running.Store(false)
	// route exit to the appropriate consumer
	if s.stopping.Load() {
		s.exit <- err
		return
	}
	s.unexpectedExit <- err
}

// Stop gracefully terminates the CLI process
func (s *Supervisor) Stop(ctx context.Context) error {
	// hold the lock to prevent concurrent modifications
	s.mu.Lock()
	defer s.mu.Unlock()
	// check if process exist and is running
	if !s.IsRunning() || s.cmd == nil {
		return nil
	}
	// store stopping status
	s.stopping.Store(true)
	defer s.stopping.Store(false)
	s.log.Info("stopping CLI process gracefully")
	// send SIGINT to the entire process group.
	pgid, err := syscall.Getpgid(s.cmd.Process.Pid)
	if err != nil {
		return fmt.Errorf("failed to get process group id: %w", err)
	}
	if err := syscall.Kill(-pgid, syscall.SIGINT); err != nil {
		return fmt.Errorf("failed to send stop signal: %w", err)
	}
	// wait for the monitoring goroutine to report exit
	select {
	case err := <-s.exit:
		return err
	case <-ctx.Done():
		s.log.Warn("graceful shutdown timed out, force killing")
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		return ctx.Err()
	}
}

// IsRunning is a concurrent-safe method to check if the Supervisor process is running
func (s *Supervisor) IsRunning() bool {
	return s.running.Load() == true
}

// IsStopping is a concurrent-safe method to check if the Supervisor process is stopping
func (s *Supervisor) IsStopping() bool {
	return s.stopping.Load() == true
}

// UnexpectedExit notifies when the process exits unexpectedly
func (s *Supervisor) UnexpectedExit() <-chan error {
	return s.unexpectedExit
}

// Coordinator code below

// CoordinatorConfig holds the configuration for the Coordinator
type CoordinatorConfig struct {
	Canopy       lib.Config    // Configuration for the canopy service
	BinPath      string        // Path to the binary file
	MaxDelayTime int           // Max time for delaying the update process (minutes)
	CheckPeriod  time.Duration // Period for checking updates
	GracePeriod  time.Duration // Grace period for tasks completion during shutdown
}

// Coordinator orchestrates the process of updating while managing CLI lifecycle
// handles the coordination between checking updates, stopping processes, and
// restarting
type Coordinator struct {
	updater          *UpdateManager     // updater instance reference
	supervisor       *Supervisor        // supervisor instance reference
	snapshot         *SnapshotManager   // snapshot instance reference
	config           *CoordinatorConfig // coordinator configuration
	updateInProgress atomic.Bool        // flag indicating if an update is in progress
	log              lib.LoggerI        // logger instance
}

// NewCoordinator creates a new Coordinator instance
func NewCoordinator(config *CoordinatorConfig, updater *UpdateManager,
	supervisor *Supervisor, snapshot *SnapshotManager, logger lib.LoggerI) *Coordinator {
	return &Coordinator{
		updater:          updater,
		supervisor:       supervisor,
		snapshot:         snapshot,
		config:           config,
		updateInProgress: atomic.Bool{},
		log:              logger,
	}
}

// UpdateLoop starts the update loop for the coordinator. This loop continuously checks
// for updates and applies them if necessary while also providing graceful shutdown for any
// termination signal received.
func (c *Coordinator) UpdateLoop(cancelSignal chan os.Signal) error {
	// start the process
	if err := c.supervisor.Start(c.config.BinPath); err != nil {
		return err
	}
	// create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// kick off an immediate check
	timer := time.NewTimer(0)
	defer timer.Stop()
	// update loop
	for {
		select {
		// unexpected process error
		case err := <-c.supervisor.UnexpectedExit():
			c.log.Warn("unexpected process exit, stopping program")
			// cancel the context to clean up resources
			cancel()
			// wait for context to clean up
			gracePeriodTimer := time.NewTimer(c.config.GracePeriod)
			defer gracePeriodTimer.Stop()
			<-gracePeriodTimer.C
			return err
		// externally closed the process (user input, container orchestrator, etc...)
		case sig := <-cancelSignal:
			c.log.Infof("received signal: %v, starting graceful shutdown", sig)
			// cancel the context to clean up resources
			cancel()
			err := c.GracefulShutdown()
			c.log.Info("completed graceful shutdown")
			return err
		// periodic check for updates
		case <-timer.C:
			// wrap it on a goroutine so it doesn't block the main loop
			go func() {
				c.log.Infof("checking for updates")
				if err := c.CheckAndApplyUpdate(ctx); err != nil {
					c.log.Errorf("update check failed: %v", err)
				}
				c.log.Infof("update check completed, performing next check in %s",
					c.config.CheckPeriod)
				// reset the timer to start the next check
				timer.Reset(c.config.CheckPeriod)
			}()
		}
	}
}

// GracefulShutdown stops the coordinator while giving a grace period to the
// canopy process to stop
func (c *Coordinator) GracefulShutdown() error {
	// stop any ongoing updates
	c.updateInProgress.Store(false)
	// check if the supervisor process is running
	if !c.supervisor.IsRunning() {
		return nil
	}
	// stop the supervised process
	shutdownCtx, cancel := context.WithTimeout(context.Background(), c.config.GracePeriod)
	defer cancel()
	return c.supervisor.Stop(shutdownCtx)
}

// CheckAndApplyUpdate performs a single update check and applies if needed
func (c *Coordinator) CheckAndApplyUpdate(ctx context.Context) error {
	// check if an update is already in progress
	if c.updateInProgress.Load() {
		c.log.Debug("update already in progress, skipping check")
		return nil
	}
	// check for new version
	release, err := c.updater.Check()
	if err != nil {
		return fmt.Errorf("failed to check for update: %w", err)
	}
	// check if an update is required
	if !release.ShouldUpdate {
		c.log.Debug("no update available")
		return nil
	}
	c.log.Infof("new version found: %s", release.Version)
	// download the new version
	if err := c.updater.Download(ctx, release); err != nil {
		return fmt.Errorf("failed to download release: %w", err)
	}
	// apply the update
	return c.ApplyUpdate(ctx, release)
}

// ApplyUpdate coordinates the update process, stopping the old process and starting the new one
// while applying a snapshot if required
func (c *Coordinator) ApplyUpdate(ctx context.Context, release *Release) error {
	canopy := c.config.Canopy
	// check if an update is already in progress
	if !c.updateInProgress.CompareAndSwap(false, true) {
		return fmt.Errorf("update already in progress")
	}
	defer c.updateInProgress.Store(false)
	c.log.Info("starting update process")
	// download snapshot if required
	var snapshotPath string
	if release.ApplySnapshot {
		snapshotPath = filepath.Join(canopy.DataDirPath, "snapshot")
		c.log.Info("downloading and extracting required snapshot")
		err := c.snapshot.DownloadAndExtract(ctx, snapshotPath, c.config.Canopy.ChainId)
		if err != nil {
			return fmt.Errorf("failed to download snapshot: %w", err)
		}
		c.log.Info("snapshot downloaded and extracted")
	}
	// add random delay for staggered updates
	if c.supervisor.IsRunning() {
		delay := time.Duration(rand.IntN(c.config.MaxDelayTime)+1) * time.Minute
		c.log.Infof("waiting %v before applying update", delay)
		timer := time.NewTimer(delay)
		// allow cancellation of timer if context is done
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	// stop current process if running
	if c.supervisor.IsRunning() {
		c.log.Info("stopping current CLI process for update")
		stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := c.supervisor.Stop(stopCtx); err != nil {
			// program may have exited with a non zero exit code due to forced close
			// this is to be expected so the update can still proceed
			c.log.Warnf("failed to stop process for update: %w", err)
		}
	}
	// replace current db with the snapshot if needed
	if snapshotPath != "" {
		c.log.Info("replacing current db with snapshot")
		dbPath := filepath.Join(canopy.DataDirPath, canopy.DBName)
		if err := c.snapshot.Replace(snapshotPath, dbPath); err != nil {
			c.log.Errorf("failed to replace db with snapshot: %v", err)
			// continue with update even if snapshot fails
		}
	}
	// restart with new version
	c.log.Infof("starting updated CLI process with version %s", release.Version)
	if err := c.supervisor.Start(c.config.BinPath); err != nil {
		return fmt.Errorf("failed to start updated process: %w", err)
	}
	c.log.Infof("update to version %s completed successfully", release.Version)
	// update UpdateManager to have the new version
	c.updater.Version = release.Version
	return nil
}
