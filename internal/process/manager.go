package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"netbird-coredns/internal/config"
	"netbird-coredns/internal/logger"
)

// Manager handles multiple processes and their lifecycle
type Manager struct {
	config    *config.Config
	processes []*Process
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// Process represents a managed process
type Process struct {
	name    string
	cmd     *exec.Cmd
	running bool
	mu      sync.RWMutex
}

// NewManager creates a new process manager
func NewManager(cfg *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		config:    cfg,
		processes: make([]*Process, 0),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// StartNetBird starts the NetBird daemon to register this service as a peer
func (m *Manager) StartNetBird() error {
	// First, ensure NetBird service is installed and started
	logger.Info("Installing NetBird service...")
	installCmd := exec.CommandContext(m.ctx, "netbird", "service", "install")
	if err := installCmd.Run(); err != nil {
		// Service might already be installed, which is fine
		logger.Debug("Note: NetBird service install returned: %v (may already be installed)", err)
	}

	logger.Info("Starting NetBird service...")
	startCmd := exec.CommandContext(m.ctx, "netbird", "service", "start")
	var startStderr bytes.Buffer
	startCmd.Stderr = &startStderr
	if err := startCmd.Run(); err != nil {
		errorOutput := startStderr.String()
		// In Docker containers, service commands may not work (no systemd)
		// Log a warning and continue - netbird up will start the daemon directly
		logger.Warn("Failed to start NetBird service (this is expected in Docker containers): %v", err)
		if errorOutput != "" {
			logger.Debug("Service start error output: %s", errorOutput)
		}
		logger.Info("Proceeding with direct NetBird startup (netbird up will start daemon automatically)...")
		// Ensure socket directory exists and is writable
		socketDir := "/var/run"
		if err := os.MkdirAll(socketDir, 0755); err != nil {
			logger.Warn("Failed to create socket directory %s: %v", socketDir, err)
		}
		logger.Debug("Ensured socket directory exists: %s", socketDir)
	} else {
		// Service started successfully, wait a moment for it to be ready
		logger.Debug("NetBird service started successfully, waiting for readiness...")
		time.Sleep(2 * time.Second)
	}

	// Now connect to the network using netbird up in foreground mode
	logger.Info("Connecting to NetBird network...")
	args := []string{
		"up",
		"--foreground-mode", // Run in foreground for Docker containers
		"--setup-key=" + m.config.SetupKey,
		"--management-url=" + m.config.ManagementURL,
		"--hostname=" + m.config.Hostname,
		"--log-level=" + m.config.LogLevel,
	}

	// Add DNS labels - critical for service discovery
	if len(m.config.DNSLabels) > 0 {
		labelsStr := strings.Join(m.config.DNSLabels, ",")
		args = append(args, "--extra-dns-labels", labelsStr)
		logger.Info("Setting DNS labels: %s", labelsStr)
		logger.Info("This DNS service will be discoverable at: %s.<netbird-domain>", m.config.DNSLabels[0])
	}

	cmd := exec.CommandContext(m.ctx, "netbird", args...)

	// Log the exact command for debugging
	logger.Debug("Executing command: netbird %s", strings.Join(args, " "))

	// Capture both stdout and stderr to detect errors
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start NetBird: %w", err)
	}

	// Wait briefly to detect immediate failures
	time.Sleep(2 * time.Second)
	errOutput := stderr.String()

	// Check for known error patterns in stderr
	if strings.Contains(errOutput, "unknown flag") && strings.Contains(errOutput, "extra-dns-labels") {
		cmd.Process.Kill()
		return fmt.Errorf("NetBird version does not support --extra-dns-labels flag. Please ensure you're using NetBird v0.32.0 or later")
	}

	// Check if process is still running
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		if errOutput != "" {
			return fmt.Errorf("NetBird process exited immediately: %s", errOutput)
		}
		return fmt.Errorf("NetBird process exited immediately: %v", err)
	}

	process := &Process{
		name:    "netbird",
		cmd:     cmd,
		running: true,
	}

	m.mu.Lock()
	m.processes = append(m.processes, process)
	m.mu.Unlock()

	logger.Info("Started NetBird with PID: %d", cmd.Process.Pid)

	// Monitor the process
	go m.monitorProcess(process)

	// Give NetBird a moment to stabilize
	time.Sleep(2 * time.Second)

	// Final check that the process is still running
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		output := stderr.String()
		if output == "" {
			output = stdout.String()
		}
		logger.Error("NetBird process failed to stay running. Output: %s", output)
		return fmt.Errorf("NetBird process failed to stay running")
	}

	return nil
}

// WaitForNetBirdConnection waits for NetBird connection to be established
func (m *Manager) WaitForNetBirdConnection() error {
	logger.Info("Waiting for NetBird connection to be established...")

	// In foreground mode, NetBird runs directly - wait for initial connection setup
	logger.Info("NetBird is running in foreground mode, waiting for initial connection setup...")

	// Check that the NetBird process is still running
	m.mu.RLock()
	var netbirdProcess *Process
	for _, p := range m.processes {
		if p.name == "netbird" {
			netbirdProcess = p
			break
		}
	}
	m.mu.RUnlock()

	if netbirdProcess == nil {
		return fmt.Errorf("NetBird process not found")
	}

	// Check if process is still running
	if err := netbirdProcess.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("NetBird process is not running")
	}

	// Wait for NetBird to establish its initial connections
	waitTime := 5 * time.Second
	logger.Info("Waiting %v for NetBird to establish connections...", waitTime)
	time.Sleep(waitTime)

	logger.Info("NetBird process is running, proceeding with CoreDNS startup")

	return nil
}

// StartCoreDNS starts the CoreDNS server with the specified config file
func (m *Manager) StartCoreDNS(corefilePath string) error {
	cmd := exec.CommandContext(m.ctx, "coredns", "-conf", corefilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start CoreDNS: %w", err)
	}

	process := &Process{
		name:    "coredns",
		cmd:     cmd,
		running: true,
	}

	m.mu.Lock()
	m.processes = append(m.processes, process)
	m.mu.Unlock()

	logger.Info("Started CoreDNS with PID: %d", cmd.Process.Pid)

	// Monitor the process
	go m.monitorProcess(process)

	return nil
}

// monitorProcess monitors a process and handles its lifecycle
func (m *Manager) monitorProcess(process *Process) {
	// Check if ProcessState is already set (meaning Wait() was already called)
	var err error
	if process.cmd.ProcessState != nil {
		// Wait() was already called, check if process exited with error
		waitStatus := process.cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitStatus := waitStatus.ExitStatus()
		if exitStatus != 0 {
			err = fmt.Errorf("process exited with status %d", exitStatus)
		}
		// If exitStatus is 0, err remains nil (successful exit)
	} else {
		// Wait() hasn't been called yet, call it now
		err = process.cmd.Wait()
	}

	process.mu.Lock()
	process.running = false
	process.mu.Unlock()

	if err != nil && m.ctx.Err() == nil {
		logger.Error("Process %s exited unexpectedly: %v", process.name, err)
		// Trigger shutdown
		m.cancel()
	}
}

// Stop gracefully stops all managed processes
func (m *Manager) Stop() error {
	logger.Info("Initiating graceful shutdown of all managed processes...")

	// Cancel context to stop all processes
	logger.Debug("Cancelling process manager context...")
	m.cancel()

	m.mu.RLock()
	processes := make([]*Process, len(m.processes))
	copy(processes, m.processes)
	m.mu.RUnlock()

	logger.Debug("Sending TERM signals to managed processes...")

	// Send TERM signal to all running processes
	for _, process := range processes {
		process.mu.RLock()
		if process.running && process.cmd.Process != nil {
			logger.Debug("Sending TERM signal to %s (PID: %d)", process.name, process.cmd.Process.Pid)
			if err := process.cmd.Process.Signal(syscall.SIGTERM); err != nil {
				logger.Warn("Failed to send TERM signal to %s: %v", process.name, err)
			}
		} else {
			logger.Debug("Process %s is not running or has no PID", process.name)
		}
		process.mu.RUnlock()
	}

	// Wait for graceful shutdown with timeout to stay within Docker's grace period
	logger.Info("Waiting for processes to shut down gracefully...")
	timeout := 2 * time.Second
	deadline := time.Now().Add(timeout)

	gracefulShutdown := false
	for time.Now().Before(deadline) {
		allStopped := true
		runningProcesses := []string{}
		for _, process := range processes {
			process.mu.RLock()
			if process.running {
				allStopped = false
				runningProcesses = append(runningProcesses, process.name)
			}
			process.mu.RUnlock()
		}
		if allStopped {
			logger.Info("All processes shut down gracefully")
			gracefulShutdown = true
			break
		}
		if len(runningProcesses) > 0 {
			logger.Debug("Still waiting for processes: %v", runningProcesses)
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !gracefulShutdown {
		logger.Warn("Graceful shutdown timeout reached, force killing remaining processes...")
		// Force kill any remaining processes
		for _, process := range processes {
			process.mu.RLock()
			if process.running && process.cmd.Process != nil {
				logger.Warn("Force killing %s (PID: %d)", process.name, process.cmd.Process.Pid)
				if err := process.cmd.Process.Kill(); err != nil {
					logger.Error("Failed to force kill %s: %v", process.name, err)
				}
			}
			process.mu.RUnlock()
		}
	}

	logger.Info("Process shutdown sequence completed")
	return nil
}

// RunWithSignalHandling runs the process manager with signal handling
func (m *Manager) RunWithSignalHandling() error {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	logger.Debug("Process manager is now waiting for signals...")

	// Wait for either termination signal or context cancellation
	select {
	case sig := <-sigChan:
		logger.Info("Received termination signal: %v - initiating graceful shutdown", sig)
	case <-m.ctx.Done():
		logger.Info("Process manager context cancelled - initiating shutdown")
	}

	logger.Info("Beginning shutdown sequence...")

	// Stop all processes
	if err := m.Stop(); err != nil {
		logger.Error("Error during shutdown: %v", err)
		return err
	}

	logger.Info("Shutdown sequence completed successfully")
	return nil
}

// GetRunningProcesses returns a list of currently running process names
func (m *Manager) GetRunningProcesses() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var running []string
	for _, process := range m.processes {
		process.mu.RLock()
		if process.running {
			running = append(running, process.name)
		}
		process.mu.RUnlock()
	}

	return running
}

// GetContext returns the manager's context
func (m *Manager) GetContext() context.Context {
	return m.ctx
}
