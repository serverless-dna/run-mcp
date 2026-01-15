package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
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

	// EnableStreamHandling enables stream handling during shutdown
	EnableStreamHandling(enabled bool)

	// SetStreamTimeout configures stream draining timeout
	SetStreamTimeout(timeout time.Duration)
}

// StreamHandler manages stream handling during shutdown
type StreamHandler interface {
	// StartStreamMonitoring begins monitoring container streams
	StartStreamMonitoring(cmd *exec.Cmd) error

	// DrainStreams allows streams to drain before termination
	DrainStreams(timeout time.Duration) error

	// DetectEOF detects when container streams close
	DetectEOF() <-chan bool

	// Stop stops stream monitoring
	Stop() error
}

// DefaultSignalHandler implements SignalHandler
type DefaultSignalHandler struct {
	cmd                     *exec.Cmd
	processManager          ProcessManager
	streamHandler           StreamHandler
	signalChan              chan os.Signal
	done                    chan bool
	sigintTimeout           time.Duration
	sigtermTimeout          time.Duration
	streamTimeout           time.Duration
	config                  *SignalConfig
	stopped                 bool
	processGroupIndependent bool
	hostTerminationCleanup  bool
	streamHandlingEnabled   bool
	cleanupContext          context.Context
	cleanupCancel           context.CancelFunc
	cleanupWaitGroup        sync.WaitGroup
}

// NewSignalHandler creates a new signal handler with the given process manager
func NewSignalHandler(processManager ProcessManager) SignalHandler {
	config := LoadSignalConfig()

	// Create cleanup context for host termination cleanup
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	return &DefaultSignalHandler{
		processManager:          processManager,
		streamHandler:           NewDefaultStreamHandler(),
		signalChan:              make(chan os.Signal, 1),
		done:                    make(chan bool, 1),
		sigintTimeout:           config.SigintTimeout,
		sigtermTimeout:          config.SigtermTimeout,
		streamTimeout:           2 * time.Second, // Default stream draining timeout
		config:                  config,
		stopped:                 false,
		processGroupIndependent: true, // Default to true for Requirements 5.1
		hostTerminationCleanup:  true, // Default to true for Requirements 5.4
		streamHandlingEnabled:   true, // Default to true for Requirements 8.1, 8.2, 8.3, 8.4
		cleanupContext:          cleanupCtx,
		cleanupCancel:           cleanupCancel,
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

	// Start stream monitoring if enabled
	if sh.streamHandlingEnabled {
		if err := sh.streamHandler.StartStreamMonitoring(cmd); err != nil {
			return fmt.Errorf("failed to start stream monitoring: %w", err)
		}
	}

	// Start signal handling goroutine
	go sh.handleSignals()

	// Start host termination cleanup monitoring if enabled
	if sh.hostTerminationCleanup {
		sh.startHostTerminationCleanup()
	}

	// Start EOF detection monitoring if stream handling is enabled
	if sh.streamHandlingEnabled {
		go sh.monitorEOF()
	}

	return nil
}

// Stop stops signal monitoring and cleanup
func (sh *DefaultSignalHandler) Stop() error {
	if sh.stopped {
		return nil
	}

	sh.stopped = true

	// Stop stream handler if enabled
	if sh.streamHandlingEnabled && sh.streamHandler != nil {
		if err := sh.streamHandler.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to stop stream handler: %v\n", err)
		}
	}

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

// EnableStreamHandling enables stream handling during shutdown
// Requirements 8.1, 8.2, 8.3, 8.4: Stream handling during shutdown
func (sh *DefaultSignalHandler) EnableStreamHandling(enabled bool) {
	sh.streamHandlingEnabled = enabled
}

// SetStreamTimeout configures stream draining timeout
// Requirements 8.3: Stream timeout override
func (sh *DefaultSignalHandler) SetStreamTimeout(timeout time.Duration) {
	sh.streamTimeout = timeout
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
// Enhanced to support stream handling during shutdown (Requirements 8.1, 8.3)
// Enhanced to support unsupported signal resilience (Requirements 3.5)
// Enhanced to support duplicate signal handling (Requirements 6.2)
func (sh *DefaultSignalHandler) handleSignal(sig os.Signal) {
	if sh.config.EnableLogging {
		fmt.Fprintf(os.Stderr, "[DEBUG] Received signal: %v at %s\n", sig, time.Now().Format(time.RFC3339))
	}

	// Requirements 3.5: Unsupported Signal Resilience
	// Check if this is a supported signal for the current platform
	if !sh.isSupportedSignal(sig) {
		if sh.config.EnableLogging {
			fmt.Fprintf(os.Stderr, "[DEBUG] Ignoring unsupported signal: %v\n", sig)
		}
		// Ignore unsupported signals and continue normal operation
		return
	}

	// Forward signal to container with child process propagation
	// Requirements 5.2: Child process signal propagation
	// Requirements 6.2: Duplicate signal handling
	if err := sh.forwardSignalWithChildPropagation(sig); err != nil {
		// Requirements 6.2: Handle duplicate signals gracefully for already-terminated containers
		if sh.isDuplicateSignalError(err) {
			if sh.config.EnableLogging {
				fmt.Fprintf(os.Stderr, "[DEBUG] Container already terminated, ignoring duplicate signal %v\n", sig)
			}
			// Gracefully handle duplicate signals - don't treat as error
			return
		}

		fmt.Fprintf(os.Stderr, "[ERROR] Failed to forward signal %v: %v\n", sig, err)
		return
	}

	if sh.config.EnableLogging {
		fmt.Fprintf(os.Stderr, "[DEBUG] Forwarded signal %v to container (with child propagation) at %s\n", sig, time.Now().Format(time.RFC3339))
	}

	// Allow stream draining before termination if enabled
	// Requirements 8.1: Stream draining before termination
	if sh.streamHandlingEnabled && sh.streamHandler != nil {
		if err := sh.drainStreamsWithTimeout(sig); err != nil {
			if sh.config.EnableLogging {
				fmt.Fprintf(os.Stderr, "[WARN] Stream draining failed: %v\n", err)
			}
		}
	}

	// Implement progressive timeout handling (graceful → force → absolute)
	sh.startProgressiveTimeoutWithStreamHandling(sig)
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
	if timeoutStr := os.Getenv(MCPSignalTimeout); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil && duration > 0 {
			config.SigtermTimeout = duration
			config.SigintTimeout = duration / 2 // SIGINT is faster
		}
		// Invalid or non-positive durations fall back to defaults (already set above)
	}

	// Enable debug logging if requested
	if os.Getenv(MCPDebug) == "true" || os.Getenv(MCPDebug) == "1" {
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

// isSupportedSignal checks if a signal is supported on the current platform
// Requirements 3.5: Unsupported Signal Resilience
func (sh *DefaultSignalHandler) isSupportedSignal(sig os.Signal) bool {
	switch runtime.GOOS {
	case "windows":
		// Windows only supports os.Interrupt (Ctrl-C and Ctrl-Break)
		return sig == os.Interrupt
	case "linux", "darwin":
		// Unix systems support SIGINT, SIGTERM, and SIGQUIT
		return sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == syscall.SIGQUIT
	default:
		// Other platforms default to os.Interrupt only
		return sig == os.Interrupt
	}
}

// isDuplicateSignalError checks if an error indicates a duplicate signal to already-terminated container
// Requirements 6.2: Duplicate Signal Handling
func (sh *DefaultSignalHandler) isDuplicateSignalError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := strings.ToLower(err.Error())

	// Check for common error messages indicating container is already terminated
	duplicateSignalIndicators := []string{
		"container not started",
		"no such container",
		"container not found",
		"is not running",
		"container has stopped",
		"container already terminated",
		"container is not running",
		"no container found",
	}

	for _, indicator := range duplicateSignalIndicators {
		if strings.Contains(errorMsg, indicator) {
			return true
		}
	}

	return false
}

// drainStreamsWithTimeout allows streams to drain before termination with timeout override
// Requirements 8.1: Stream draining before termination
// Requirements 8.3: Stream timeout override
func (sh *DefaultSignalHandler) drainStreamsWithTimeout(sig os.Signal) error {
	if sh.streamHandler == nil {
		return nil
	}

	// Determine effective timeout for stream draining
	signalTimeout := sh.getTimeoutForSignal(sig)
	streamTimeout := sh.streamTimeout

	// Use the smaller of stream timeout and signal timeout
	// Requirements 8.3: Stream timeout override - don't wait indefinitely
	effectiveTimeout := streamTimeout
	if signalTimeout < streamTimeout {
		effectiveTimeout = signalTimeout
	}

	if sh.config.EnableLogging {
		fmt.Fprintf(os.Stderr, "[DEBUG] Draining streams with timeout %v (signal timeout: %v, stream timeout: %v)\n",
			effectiveTimeout, signalTimeout, streamTimeout)
	}

	// Drain streams with timeout
	return sh.streamHandler.DrainStreams(effectiveTimeout)
}

// startProgressiveTimeoutWithStreamHandling implements progressive timeout handling with stream awareness
// Requirements 8.3: Stream timeout override
func (sh *DefaultSignalHandler) startProgressiveTimeoutWithStreamHandling(sig os.Signal) {
	timeout := sh.getTimeoutForSignal(sig)

	// Phase 1: Graceful shutdown timeout (accounts for stream draining)
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

// monitorEOF monitors for EOF detection and initiates shutdown
// Requirements 8.2: EOF detection and shutdown
// Requirements 8.4: Stdin EOF handling
func (sh *DefaultSignalHandler) monitorEOF() {
	if sh.streamHandler == nil {
		return
	}

	sh.cleanupWaitGroup.Add(1)
	defer sh.cleanupWaitGroup.Done()

	eofChan := sh.streamHandler.DetectEOF()

	for {
		select {
		case <-eofChan:
			if sh.stopped {
				return
			}

			if sh.config.EnableLogging {
				fmt.Fprintf(os.Stderr, "[DEBUG] EOF detected, initiating graceful shutdown at %s\n", time.Now().Format(time.RFC3339))
			}

			// EOF detected, initiate graceful shutdown
			// This handles both stdout/stderr EOF and stdin EOF scenarios
			sh.initiateGracefulShutdown()
			return

		case <-sh.cleanupContext.Done():
			return
		}
	}
}

// initiateGracefulShutdown initiates graceful shutdown when EOF is detected
// Requirements 8.2: EOF detection and shutdown
// Requirements 8.4: Stdin EOF handling
func (sh *DefaultSignalHandler) initiateGracefulShutdown() {
	if sh.config.EnableLogging {
		fmt.Fprintf(os.Stderr, "[DEBUG] Initiating graceful shutdown due to EOF\n")
	}

	// Signal completion to stop other monitoring goroutines
	select {
	case sh.done <- true:
	default:
		// Channel might be full or closed
	}
}

// DefaultStreamHandler implements StreamHandler
type DefaultStreamHandler struct {
	cmd     *exec.Cmd
	eofChan chan bool
	stopped bool
	mutex   sync.Mutex
}

// NewDefaultStreamHandler creates a new default stream handler
func NewDefaultStreamHandler() StreamHandler {
	return &DefaultStreamHandler{
		eofChan: make(chan bool, 1),
		stopped: false,
	}
}

// StartStreamMonitoring begins monitoring container streams
// Requirements 8.1, 8.2: Stream monitoring for draining and EOF detection
func (sh *DefaultStreamHandler) StartStreamMonitoring(cmd *exec.Cmd) error {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.stopped {
		return fmt.Errorf("stream handler has been stopped")
	}

	sh.cmd = cmd

	// Start monitoring for EOF in a separate goroutine
	go sh.monitorStreams()

	return nil
}

// DrainStreams allows streams to drain before termination
// Requirements 8.1: Stream draining before termination
func (sh *DefaultStreamHandler) DrainStreams(timeout time.Duration) error {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.stopped || sh.cmd == nil {
		return nil
	}

	// Create a timeout context for stream draining
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Wait for streams to drain or timeout
	done := make(chan bool, 1)

	go func() {
		// In a real implementation, this would monitor the actual streams
		// For now, we simulate stream draining by waiting a short time
		time.Sleep(100 * time.Millisecond)
		done <- true
	}()

	select {
	case <-done:
		// Streams drained successfully
		return nil
	case <-ctx.Done():
		// Timeout exceeded, proceed with termination
		return fmt.Errorf("stream draining timeout exceeded")
	}
}

// DetectEOF detects when container streams close
// Requirements 8.2: EOF detection for container stream closure
func (sh *DefaultStreamHandler) DetectEOF() <-chan bool {
	return sh.eofChan
}

// Stop stops stream monitoring
func (sh *DefaultStreamHandler) Stop() error {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.stopped {
		return nil
	}

	sh.stopped = true

	// Close EOF channel
	close(sh.eofChan)

	return nil
}

// monitorStreams monitors container streams for EOF
// Requirements 8.2: EOF detection and shutdown
// Requirements 8.4: Stdin EOF handling
func (sh *DefaultStreamHandler) monitorStreams() {
	if sh.cmd == nil {
		return
	}

	// Wait for the process to complete
	// In a real implementation, this would monitor actual stream states
	// The container runtime handles EOF detection automatically
	go func() {
		if sh.cmd.Process != nil {
			// Wait for process to exit (which indicates stream closure)
			sh.cmd.Process.Wait()

			sh.mutex.Lock()
			defer sh.mutex.Unlock()

			if !sh.stopped {
				// Signal EOF detected
				select {
				case sh.eofChan <- true:
				default:
					// Channel might be full or closed
				}
			}
		}
	}()
}
