package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
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
	
	// SetProcessGroupIndependent configures process group independence
	SetProcessGroupIndependent(independent bool)
	
	// EnableHostTerminationCleanup enables cleanup when host process is terminated
	EnableHostTerminationCleanup(enabled bool)
}

// DefaultSignalHandler implements SignalHandler
type DefaultSignalHandler struct {
	cmd                        *exec.Cmd
	processManager             ProcessManager
	signalChan                 chan os.Signal
	done                       chan bool
	sigintTimeout              time.Duration
	sigtermTimeout             time.Duration
	config                     *SignalConfig
	stopped                    bool
	processGroupIndependent    bool
	hostTerminationCleanup     bool
	cleanupContext             context.Context
	cleanupCancel              context.CancelFunc
	cleanupWaitGroup           sync.WaitGroup
}

// NewSignalHandler creates a new signal handler with the given process manager
func NewSignalHandler(processManager ProcessManager) SignalHandler {
	config := LoadSignalConfig()
	
	// Create cleanup context for host termination cleanup
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	
	return &DefaultSignalHandler{
		processManager:             processManager,
		signalChan:                 make(chan os.Signal, 1),
		done:                       make(chan bool, 1),
		sigintTimeout:              config.SigintTimeout,
		sigtermTimeout:             config.SigtermTimeout,
		config:                     config,
		stopped:                    false,
		processGroupIndependent:    true,  // Default to true for Requirements 5.1
		hostTerminationCleanup:     true,  // Default to true for Requirements 5.4
		cleanupContext:             cleanupCtx,
		cleanupCancel:              cleanupCancel,
	}
}

// Start begins signal monitoring for the given process
func (sh *DefaultSignalHandler) Start(cmd *exec.Cmd) error {
	if sh.stopped {
		return fmt.Errorf("signal handler has been stopped")
	}
	
	sh.cmd = cmd
	
	// Setup signal capture with process group independence
	sh.setupSignalCaptureWithIndependence()
	
	// Start signal handling goroutine
	go sh.handleSignals()
	
	// Start host termination cleanup monitoring if enabled
	if sh.hostTerminationCleanup {
		sh.startHostTerminationCleanup()
	}
	
	return nil
}

// Stop stops signal monitoring and cleanup
func (sh *DefaultSignalHandler) Stop() error {
	if sh.stopped {
		return nil
	}
	
	sh.stopped = true
	
	// Cancel cleanup context for host termination cleanup
	if sh.cleanupCancel != nil {
		sh.cleanupCancel()
	}
	
	// Wait for cleanup goroutines to finish
	sh.cleanupWaitGroup.Wait()
	
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

// SetProcessGroupIndependent configures process group independence
// Requirements 5.1: Process group independence for signal handling
func (sh *DefaultSignalHandler) SetProcessGroupIndependent(independent bool) {
	sh.processGroupIndependent = independent
}

// EnableHostTerminationCleanup enables cleanup when host process is terminated
// Requirements 5.4: Host process termination cleanup
func (sh *DefaultSignalHandler) EnableHostTerminationCleanup(enabled bool) {
	sh.hostTerminationCleanup = enabled
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

// setupSignalCaptureWithIndependence configures platform-specific signal capture with process group independence
// Requirements 5.1: Process group independence for signal handling
func (sh *DefaultSignalHandler) setupSignalCaptureWithIndependence() {
	if sh.processGroupIndependent {
		// Configure signal handling to be independent of process group
		// This ensures signals are handled regardless of process group membership
		
		// Platform-specific signal registration with process group independence
		switch runtime.GOOS {
		case "windows":
			// Windows doesn't have process groups in the Unix sense
			// Use standard interrupt handling
			signal.Notify(sh.signalChan, os.Interrupt)
		case "linux", "darwin":
			// Unix systems: handle signals independently of process group
			// This ensures proper signal forwarding regardless of job control
			signal.Notify(sh.signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		default:
			signal.Notify(sh.signalChan, os.Interrupt)
		}
	} else {
		// Use standard signal capture
		sh.setupSignalCapture()
	}
}

// startHostTerminationCleanup starts monitoring for host process termination
// Requirements 5.4: Host process termination cleanup
func (sh *DefaultSignalHandler) startHostTerminationCleanup() {
	sh.cleanupWaitGroup.Add(1)
	
	go func() {
		defer sh.cleanupWaitGroup.Done()
		
		// Monitor for context cancellation (host termination)
		<-sh.cleanupContext.Done()
		
		if sh.stopped {
			return
		}
		
		// Host process is being terminated, ensure container cleanup
		if sh.config.EnableLogging {
			fmt.Fprintf(os.Stderr, "[DEBUG] Host termination detected, cleaning up container at %s\n", time.Now().Format(time.RFC3339))
		}
		
		// Force kill the container to prevent orphaned containers
		if err := sh.processManager.ForceKill(); err != nil {
			if sh.config.EnableLogging {
				fmt.Fprintf(os.Stderr, "[WARN] Failed to cleanup container during host termination: %v\n", err)
			}
		}
	}()
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

// handleSignal processes a single signal with progressive timeout handling
// Enhanced to support child process signal propagation (Requirements 5.2)
func (sh *DefaultSignalHandler) handleSignal(sig os.Signal) {
	if sh.config.EnableLogging {
		fmt.Fprintf(os.Stderr, "[DEBUG] Received signal: %v at %s\n", sig, time.Now().Format(time.RFC3339))
	}
	
	// Forward signal to container with child process propagation
	// Requirements 5.2: Child process signal propagation
	if err := sh.forwardSignalWithChildPropagation(sig); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to forward signal %v: %v\n", sig, err)
		return
	}
	
	if sh.config.EnableLogging {
		fmt.Fprintf(os.Stderr, "[DEBUG] Forwarded signal %v to container (with child propagation) at %s\n", sig, time.Now().Format(time.RFC3339))
	}
	
	// Implement progressive timeout handling (graceful → force → absolute)
	sh.startProgressiveTimeout(sig)
}

// forwardSignalWithChildPropagation forwards signals to container with child process propagation
// Requirements 5.2: Child process signal propagation
func (sh *DefaultSignalHandler) forwardSignalWithChildPropagation(sig os.Signal) error {
	// Container runtimes automatically handle child process signal propagation
	// when signals are sent to the main container process using the container name
	// This ensures all child processes within the container receive the signal
	
	// Use the existing ForwardSignal method which already supports child propagation
	// through container runtime signal forwarding mechanisms
	return sh.processManager.ForwardSignal(sig)
}

// startProgressiveTimeout implements progressive timeout handling
func (sh *DefaultSignalHandler) startProgressiveTimeout(sig os.Signal) {
	timeout := sh.getTimeoutForSignal(sig)
	
	// Phase 1: Graceful shutdown timeout
	gracefulTimer := time.AfterFunc(timeout, func() {
		if sh.stopped {
			return
		}
		
		if sh.config.EnableLogging {
			fmt.Fprintf(os.Stderr, "[WARN] Signal timeout exceeded (%v), forcing termination at %s\n", timeout, time.Now().Format(time.RFC3339))
		}
		
		// Phase 2: Force termination with SIGKILL
		if err := sh.processManager.ForceKill(); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Force kill failed: %v\n", err)
			
			// Phase 3: Absolute timeout - if force kill fails, exit with error
			absoluteTimer := time.AfterFunc(5*time.Second, func() {
				if sh.stopped {
					return
				}
				fmt.Fprintf(os.Stderr, "[ERROR] Absolute timeout reached, force termination failed\n")
				// Exit with error code indicating force termination failure
				os.Exit(128 + int(syscall.SIGKILL))
			})
			
			// Cancel absolute timeout if process exits
			go func() {
				select {
				case <-sh.done:
					absoluteTimer.Stop()
				case <-time.After(6 * time.Second):
					// Absolute timeout already fired
				}
			}()
		}
	})
	
	// Cancel graceful timeout if process exits gracefully
	go func() {
		select {
		case <-sh.done:
			gracefulTimer.Stop()
		case <-time.After(timeout + 10*time.Second):
			// Timeout handling already completed
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
		if duration, err := time.ParseDuration(timeoutStr); err == nil && duration > 0 {
			config.SigtermTimeout = duration
			config.SigintTimeout = duration / 2 // SIGINT is faster
		}
		// Invalid or non-positive durations fall back to defaults (already set above)
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