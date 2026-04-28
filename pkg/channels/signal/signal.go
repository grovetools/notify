// Package signal implements the Signal messaging channel using signal-cli daemon mode.
package signal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/grovetools/notify/pkg/channels"
)

// Config holds Signal channel configuration.
type Config struct {
	CLIPath   string   // Path to signal-cli binary
	Account   string   // Signal account phone number
	Allowlist []string // Authorized sender phone numbers
}

// Channel implements channels.Channel for Signal messaging via signal-cli.
type Channel struct {
	config    Config
	running   bool
	mu        sync.RWMutex
	cancel    context.CancelFunc
	allowlist map[string]bool
	daemonCmd *exec.Cmd

	restartCount  int
	lastRestartAt time.Time
	alive         bool
}

// NewChannel creates a new Signal channel with the given configuration.
func NewChannel(cfg Config) *Channel {
	allowmap := make(map[string]bool, len(cfg.Allowlist))
	for _, num := range cfg.Allowlist {
		allowmap[num] = true
	}
	return &Channel{
		config:    cfg,
		allowlist: allowmap,
	}
}

// killOrphanedSignalCLIDaemons terminates any running signal-cli daemon
// processes before a fresh one is spawned. When groved is killed
// ungracefully (SIGKILL, crash), its child signal-cli gets reparented to
// init and keeps the JSON-RPC socket bound, but its stdout goes nowhere.
// A new groved starting up would race against the orphan on socket bind
// and lose inbound routing. Killing orphans first guarantees the new
// signal-cli is a clean child of the current groved.
//
// Matches `signal-cli ... daemon --socket ...` specifically (the daemon-mode
// invocation) to avoid killing one-shot `signal-cli send` subprocesses.
func killOrphanedSignalCLIDaemons() {
	// pkill returns non-zero when no processes match; ignore error.
	_ = exec.Command("pkill", "-TERM", "-f", "signal-cli.*daemon.*--socket").Run()
	// Brief wait so the JVM has a chance to release its socket fd before
	// the new signal-cli tries to bind. signal-cli responds to SIGTERM
	// within a few hundred ms in practice.
	waitForSignalCLIExit(1 * time.Second)
}

// waitForSignalCLIExit polls pgrep until no matching daemon processes
// remain or the timeout elapses. Cheap and more reliable than a flat sleep.
func waitForSignalCLIExit(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := exec.Command("pgrep", "-f", "signal-cli.*daemon.*--socket").Run(); err != nil {
			// pgrep returns 1 when nothing matches — orphans are gone.
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Name returns the channel identifier.
func (c *Channel) Name() string { return "signal" }

// IsAllowed checks if a sender is in the allowlist.
func (c *Channel) IsAllowed(senderID string) bool {
	return c.allowlist[senderID]
}

// Status returns the supervision state of the signal-cli subprocess.
func (c *Channel) Status() channels.ChannelStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return channels.ChannelStatus{
		IsAlive:       c.alive,
		RestartCount:  c.restartCount,
		LastRestartAt: c.lastRestartAt,
	}
}

// Start begins the signal-cli daemon and routes inbound messages via the callback.
func (c *Channel) Start(ctx context.Context, onMessage func(channels.InboundMessage)) error {
	listenCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.mu.Lock()
	c.running = true
	c.mu.Unlock()

	killOrphanedSignalCLIDaemons()

	if socketPath := c.signalSocketPath(); socketPath != "" {
		os.Remove(socketPath)
	}

	go c.supervisorLoop(listenCtx, onMessage)
	return nil
}

func (c *Channel) supervisorLoop(ctx context.Context, onMessage func(channels.InboundMessage)) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		err := c.runDaemon(ctx, onMessage)

		select {
		case <-ctx.Done():
			c.mu.Lock()
			c.alive = false
			c.mu.Unlock()
			return
		default:
		}

		c.mu.Lock()
		c.alive = false
		c.restartCount++
		c.lastRestartAt = time.Now()
		c.mu.Unlock()

		log.Printf("[signal] signal-cli exited unexpectedly (err=%v), restarting in %v (restart #%d)", err, backoff, c.restartCount)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		// Kill any orphaned signal-cli before restarting to avoid config lock contention
		killOrphanedSignalCLIDaemons()
		if socketPath := c.signalSocketPath(); socketPath != "" {
			os.Remove(socketPath)
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (c *Channel) runDaemon(ctx context.Context, onMessage func(channels.InboundMessage)) error {
	cmd := exec.CommandContext(ctx, c.config.CLIPath, "-a", c.config.Account, "-o", "json", "daemon", "--socket", "--receive-mode", "on-start") //nolint:gosec // CLIPath is from trusted config
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	c.mu.Lock()
	c.daemonCmd = cmd
	c.alive = true
	c.mu.Unlock()

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var msg signalMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}

		if msg.Envelope.DataMessage == nil || msg.Envelope.DataMessage.Message == "" {
			continue
		}

		sender := msg.Envelope.Source
		if !c.IsAllowed(sender) {
			continue
		}

		inbound := channels.InboundMessage{
			Channel: c.Name(),
			Source:  sender,
			Message: msg.Envelope.DataMessage.Message,
		}

		if msg.Envelope.DataMessage.Quote != nil {
			inbound.Quote = &channels.Quote{
				ID:     msg.Envelope.DataMessage.Quote.ID,
				Author: msg.Envelope.DataMessage.Quote.Author,
				Text:   msg.Envelope.DataMessage.Quote.Text,
			}
		}

		onMessage(inbound)
	}

	return cmd.Wait()
}

// Send sends an outbound message via signal-cli's JSON-RPC socket.
// Falls back to spawning a separate signal-cli send process if the socket isn't available.
func (c *Channel) Send(ctx context.Context, req channels.OutboundMessage) (*channels.SendResult, error) {
	socketPath := c.signalSocketPath()
	if socketPath != "" {
		return c.sendViaSocket(socketPath, req.Recipient, req.Message)
	}

	// Fallback: spawn signal-cli send
	return c.sendViaCommand(req.Recipient, req.Message)
}

// Stop gracefully shuts down the Signal channel.
func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	if c.daemonCmd != nil && c.daemonCmd.Process != nil {
		_ = c.daemonCmd.Process.Kill()
		_ = c.daemonCmd.Wait()
	}

	c.running = false
	return nil
}

// signalMessage represents a message from signal-cli --json output.
type signalMessage struct {
	Envelope struct {
		Source      string `json:"source"`
		DataMessage *struct {
			Timestamp int64  `json:"timestamp"`
			Message   string `json:"message"`
			Quote     *struct {
				ID     int64  `json:"id"`     // Timestamp of the message being replied to
				Author string `json:"author"` // Who sent the original message
				Text   string `json:"text"`   // Quoted text (may be truncated)
			} `json:"quote"`
		} `json:"dataMessage"`
	} `json:"envelope"`
}

// signalSocketPath returns the signal-cli daemon socket path if it exists.
func (c *Channel) signalSocketPath() string {
	candidates := []string{}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		candidates = append(candidates, xdg+"/signal-cli/socket")
	}
	if tmpDir := os.TempDir(); tmpDir != "" {
		candidates = append(candidates, tmpDir+"/signal-cli/socket")
	}
	candidates = append(candidates, "/tmp/signal-cli/socket")
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// sendViaSocket sends a message through signal-cli's JSON-RPC unix socket.
func (c *Channel) sendViaSocket(socketPath, recipient, content string) (*channels.SendResult, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to signal-cli socket: %w", err)
	}
	defer conn.Close()

	request := map[string]any{
		"jsonrpc": "2.0",
		"method":  "send",
		"id":      "1",
		"params": map[string]any{
			"recipient": []string{recipient},
			"message":   content,
		},
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}
	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write to signal-cli socket: %w", err)
	}

	// Read response to get the timestamp for routing
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		var resp struct {
			Result struct {
				Timestamp int64 `json:"timestamp"`
			} `json:"result"`
			Error any `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err == nil {
			if resp.Error != nil {
				return nil, fmt.Errorf("signal-cli JSON-RPC error: %v", resp.Error)
			}
			return &channels.SendResult{Timestamp: resp.Result.Timestamp}, nil
		}
	}

	return &channels.SendResult{}, nil
}

// sendViaCommand sends a message by spawning signal-cli send (fallback).
func (c *Channel) sendViaCommand(recipient, content string) (*channels.SendResult, error) {
	cmd := exec.Command(c.config.CLIPath, "-a", c.config.Account, "send", "-m", content, recipient) //nolint:gosec // CLIPath is from trusted config
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("signal-cli send failed: %w (output: %s)", err, string(output))
	}
	return &channels.SendResult{}, nil
}

// SendDirect sends a message directly via signal-cli without requiring the daemon to be running.
// This is used by the notify CLI as a standalone fallback.
func SendDirect(cliPath, account, recipient, message string) error {
	cmd := exec.Command(cliPath, "-a", account, "send", "-m", message, recipient)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signal-cli send failed: %w (output: %s)", err, string(output))
	}
	return nil
}
