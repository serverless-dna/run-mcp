package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// SignalHandler manages signal capture and forwarding
type SignalHandler interface {
	// Start begins signal monitoring for the given process
	Start(cmd *exec.Cmd) error
	
	// Stop stops signal monitoring and cleanup
	Stop() error
	
	// SetTimeout configures signal timeout values
	SetTimeout(sigint, sigterm time.Duration)
	
	// GetProcessManager returns the associated process manager
	GetProcessManager() ProcessManager
}

// DefaultSignalHandler implements SignalHandler
type DefaultSignalHandler struct {
	cmd            *exec.Cmd
	processManager ProcessManager
	signalChan     chan os.Signal
	done           chan bool
	sigintTimeout  time.Duration
	sigtermTimeout time.Duration
	config         *SignalConfig
	stopped        bool
}

// NewSignalHandler creates a new signal handler with the given process manager
func NewSignalHandler(processManager ProcessManager) SignalHandler {
	config := LoadSignalConfig()
	
	return &DefaultSignalHandler{
		processManager: processManager,
		signalChan:     make(chan os.Signal, 1),
		done:           make(chan bool, 1),
		sigintTimeout:  config.SigintTimeout,
		sigtermTimeout: config.SigtermTimeout,
		config:         config,
		stopped:        false,
	}
}

// Start begins signal monitoring for the given process
func (sh *DefaultSignalHandler) Start(cmd *exec.Cmd) error {
	if sh.stopped {
		return fmt.Errorf("signal handler has been stopped")
	}
	
	sh.cmd = cmd
	
	// Setup signal capture
	sh.setupSignalCapture()
	
	// Start signal handling goroutine
	go sh.handleSignals()
	
	return nil
}

// Stop stops signal monitoring and cleanup
func (sh *DefaultSignalHandler) Stop() error {
	if sh.stopped {
		return nil
	}
	
	sh.stopped = true
	
	// Stop signal notifications
	signal.Stop(sh.signalChan)
	
	// Signal completion
	select {
	case sh.done <- true:
	default:
		// Channel might be full or closed
	}
	
	return nil
}

// SetTimeout configures signal timeout values
func (sh *DefaultSignalHandler) SetTimeout(sigint, sigterm time.Duration) {
	sh.sigintTimeout = sigint
	sh.sigtermTimeout = sigterm
}

// GetProcessManager returns the associated process manager
func (sh *DefaultSignalHandler) GetProcessManager() ProcessManager {
	return sh.processManager
}

// setupSignalCapture configures platform-specific signal capture
func (sh *DefaultSignalHandler) setupSignalCapture() {
	// Platform-specific signal registration
	switch runtime.GOOS {
	case "windows":
		signal.Notify(sh.signalChan, os.Interrupt)
	case "linux", "darwin":
		signal.Notify(sh.signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	default:
		signal.Notify(sh.signalChan, os.Interrupt)
	}
}

// handleSignals processes incoming signals
func (sh *DefaultSignalHandler) handleSignals() {
	for {
		select {
		case sig := <-sh.signalChan:
			if sh.stopped {
				return
			}
			sh.handleSignal(sig)
		case <-sh.done:
			return
		}
	}
}

// handleSignal processes a single signal
func (sh *DefaultSignalHandler) handleSignal(sig os.Signal) {
	if sh.config.EnableLogging {
		fmt.Fprintf(os.Stderr, "[DEBUG] Received signal: %v at %s\n", sig, time.Now().Format(time.RFC3339))
	}
	
	// Forward signal to container
	if err := sh.processManager.ForwardSignal(sig); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to forward signal %v: %v\n", sig, err)
		return
	}
	
	if sh.config.EnableLogging {
		fmt.Fprintf(os.Stderr, "[DEBUG] Forwarded signal %v to container at %s\n", sig, time.Now().Format(time.RFC3339))
	}
	
	// Start timeout for graceful shutdown
	timeout := sh.getTimeoutForSignal(sig)
	timer := time.AfterFunc(timeout, func() {
		if sh.stopped {
			return
		}
		
		fmt.Fprintf(os.Stderr, "[WARN] Signal timeout exceeded (%v), forcing termination at %s\n", timeout, time.Now().Format(time.RFC3339))
		if err := sh.processManager.ForceKill(); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Force kill failed: %v\n", err)
		}
	})
	
	// Cancel timeout if process exits gracefully
	go func() {
		select {
		case <-sh.done:
			timer.Stop()
		case <-time.After(timeout + time.Second):
			// Timeout already fired, nothing to do
		}
	}()
}

// getTimeoutForSignal returns the appropriate timeout for a signal
func (sh *DefaultSignalHandler) getTimeoutForSignal(sig os.Signal) time.Duration {
	switch sig {
	case syscall.SIGINT:
		return sh.sigintTimeout
	case syscall.SIGTERM:
		return sh.sigtermTimeout
	default:
		return sh.sigtermTimeout // Default to SIGTERM timeout
	}
}

// SignalConfig holds signal handling configuration
type SignalConfig struct {
	SigintTimeout  time.Duration
	SigtermTimeout time.Duration
	EnableLogging  bool
}

// LoadSignalConfig loads signal configuration from environment variables
func LoadSignalConfig() *SignalConfig {
	config := &SignalConfig{
		SigintTimeout:  5 * time.Second,
		SigtermTimeout: 10 * time.Second,
		EnableLogging:  false,
	}
	
	// Load timeout from MCP_SIGNAL_TIMEOUT environment variable
	if timeoutStr := os.Getenv("MCP_SIGNAL_TIMEOUT"); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			config.SigtermTimeout = duration
			config.SigintTimeout = duration / 2 // SIGINT is faster
		}
	}
	
	// Enable debug logging if requested
	if os.Getenv("MCP_DEBUG") == "true" || os.Getenv("MCP_DEBUG") == "1" {
		config.EnableLogging = true
	}
	
	return config
}

// getSignalName converts a signal to its string name for container commands
func getSignalName(sig os.Signal) string {
	switch sig {
	case syscall.SIGINT:
		return "SIGINT"
	case syscall.SIGTERM:
		return "SIGTERM"
	case syscall.SIGKILL:
		return "SIGKILL"
	case syscall.SIGQUIT:
		return "SIGQUIT"
	default:
		return sig.String()
	}
}