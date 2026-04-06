// Package signal implements the Signal messaging channel using signal-cli daemon mode.
package signal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"

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

// Name returns the channel identifier.
func (c *Channel) Name() string { return "signal" }

// IsAllowed checks if a sender is in the allowlist.
func (c *Channel) IsAllowed(senderID string) bool {
	return c.allowlist[senderID]
}

// Start begins the signal-cli daemon and routes inbound messages via the callback.
func (c *Channel) Start(ctx context.Context, onMessage func(channels.InboundMessage)) error {
	listenCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.mu.Lock()
	c.running = true
	c.mu.Unlock()

	// Clean up stale signal-cli socket from previous runs
	if socketPath := c.signalSocketPath(); socketPath != "" {
		os.Remove(socketPath)
	}

	// Start signal-cli daemon mode — stays connected, streams received messages to stdout,
	// and exposes a JSON-RPC socket for sending.
	c.daemonCmd = exec.CommandContext(listenCtx, c.config.CLIPath, "-a", c.config.Account, "-o", "json", "daemon", "--socket", "--receive-mode", "on-start")
	stdout, err := c.daemonCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create signal-cli stdout pipe: %w", err)
	}

	if err := c.daemonCmd.Start(); err != nil {
		return fmt.Errorf("failed to start signal-cli daemon: %w", err)
	}

	// Read incoming messages from daemon stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			select {
			case <-listenCtx.Done():
				return
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

			// Parse quote for reply-based routing
			if msg.Envelope.DataMessage.Quote != nil {
				inbound.Quote = &channels.Quote{
					ID:     msg.Envelope.DataMessage.Quote.ID,
					Author: msg.Envelope.DataMessage.Quote.Author,
					Text:   msg.Envelope.DataMessage.Quote.Text,
				}
			}

			onMessage(inbound)
		}
	}()

	return nil
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
		c.daemonCmd.Process.Kill()
		c.daemonCmd.Wait()
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
	cmd := exec.Command(c.config.CLIPath, "-a", c.config.Account, "send", "-m", content, recipient)
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
