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
	"path/filepath"
	"strings"
	"sync"
	"time"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/notify/pkg/channels"
)

var ulog = grovelogging.NewUnifiedLogger("groved.signal")

const (
	// stderrTailBytes bounds the retained signal-cli stderr (last N bytes).
	stderrTailBytes = 4096
	// fastExitThreshold: a daemon exit with less uptime than this counts as
	// a fast failure for the circuit breaker.
	fastExitThreshold = 10 * time.Second
	// maxFastExits: consecutive fast failures before supervision stops.
	maxFastExits = 5
)

// Config holds Signal channel configuration.
type Config struct {
	CLIPath   string   // Path to signal-cli binary
	Account   string   // Signal account phone number
	Allowlist []string // Authorized sender phone numbers
	Groups    []string // Authorized Signal group IDs (base64)
}

// Channel implements channels.Channel for Signal messaging via signal-cli.
type Channel struct {
	config    Config
	running   bool
	mu        sync.RWMutex
	cancel    context.CancelFunc
	allowlist map[string]bool
	groupMap  map[string]bool
	daemonCmd *exec.Cmd

	restartCount  int
	lastRestartAt time.Time
	alive         bool
	lastError     string // most recent exit cause (with short stderr tail)
	stopped       bool   // supervision permanently stopped (breaker tripped)

	// Supervisor backoff bounds; overridable in tests.
	initialBackoff time.Duration
	maxBackoff     time.Duration
}

// tailBuffer is an io.Writer that keeps only the last max bytes written.
// Used to capture the tail of signal-cli's stderr for failure logs.
type tailBuffer struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.max {
		t.buf = t.buf[len(t.buf)-t.max:]
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.TrimSpace(string(t.buf))
}

// lastErrorTailBytes bounds the stderr tail included in the recorded
// lastError so Status() reporting stays to roughly one line.
const lastErrorTailBytes = 200

// formatExitError combines a daemon exit error with a short tail of its
// stderr into a single human-readable cause for Status() reporting.
func formatExitError(err error, stderrTail string) string {
	tail := strings.TrimSpace(stderrTail)
	if len(tail) > lastErrorTailBytes {
		tail = tail[len(tail)-lastErrorTailBytes:]
	}
	switch {
	case err != nil && tail != "":
		return fmt.Sprintf("%v: %s", err, tail)
	case err != nil:
		return err.Error()
	case tail != "":
		return tail
	default:
		return "signal-cli exited without error"
	}
}

// NewChannel creates a new Signal channel with the given configuration.
func NewChannel(cfg Config) *Channel {
	allowmap := make(map[string]bool, len(cfg.Allowlist))
	for _, num := range cfg.Allowlist {
		allowmap[num] = true
	}
	groupmap := make(map[string]bool, len(cfg.Groups))
	for _, g := range cfg.Groups {
		groupmap[g] = true
	}
	return &Channel{
		config:         cfg,
		allowlist:      allowmap,
		groupMap:       groupmap,
		initialBackoff: time.Second,
		maxBackoff:     30 * time.Second,
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
		LastError:     c.lastError,
		Stopped:       c.stopped,
	}
}

// preflight validates the channel configuration before any supervision
// starts: the signal-cli binary must exist and be executable, an account
// must be configured, and (when determinable) the account must be
// registered with signal-cli. A descriptive error here prevents the
// supervisor loop from churning forever on a setup that can never work.
func (c *Channel) preflight() error {
	if c.config.Account == "" {
		return fmt.Errorf("no signal account configured")
	}
	info, err := os.Stat(c.config.CLIPath)
	if err != nil {
		return fmt.Errorf("signal-cli binary not found at %q: %w", c.config.CLIPath, err)
	}
	if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("signal-cli path %q is not an executable file", c.config.CLIPath)
	}
	return c.checkRegistration()
}

// accountsJSONPath returns the path of signal-cli's account registry
// (<data-dir>/data/accounts.json for the default data dir), or "" if the
// home directory cannot be resolved.
func accountsJSONPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "signal-cli", "data", "accounts.json")
}

// checkRegistration verifies that the configured account is registered
// with signal-cli. Primary source: signal-cli's accounts.json (format
// {"accounts":[{"number":"+1...","path":...},...],"version":2}, verified
// against signal-cli 0.14.5). Fallback when the file is missing or
// unparseable: a best-effort `signal-cli -o json listAccounts` probe
// (subcommand verified via --help). If neither source is conclusive the
// check passes — the supervisor circuit breaker bounds any resulting churn.
func (c *Channel) checkRegistration() error {
	if path := accountsJSONPath(); path != "" {
		if data, err := os.ReadFile(path); err == nil { //nolint:gosec // fixed well-known path
			var f struct {
				Accounts []struct {
					Number string `json:"number"`
					Path   string `json:"path"`
				} `json:"accounts"`
			}
			if jerr := json.Unmarshal(data, &f); jerr == nil {
				for _, a := range f.Accounts {
					if a.Number == c.config.Account || a.Path == c.config.Account {
						return nil
					}
				}
				return fmt.Errorf("account %s is not registered with signal-cli (%s lists %d account(s)); run `%s -a %s register` or link a device, then restart the daemon",
					c.config.Account, path, len(f.Accounts), c.config.CLIPath, c.config.Account)
			}
			// Unparseable accounts.json: fall through to the CLI probe.
		}
	}

	probeCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(probeCtx, c.config.CLIPath, "-o", "json", "listAccounts").Output() //nolint:gosec // CLIPath is from trusted config
	if err != nil {
		// Inconclusive (probe failed to run); do not block startup.
		return nil
	}
	if !strings.Contains(string(out), c.config.Account) {
		return fmt.Errorf("account %s is not registered according to `signal-cli listAccounts`; run `%s -a %s register` or link a device, then restart the daemon",
			c.config.Account, c.config.CLIPath, c.config.Account)
	}
	return nil
}

// Start begins the signal-cli daemon and routes inbound messages via the callback.
// It fails fast (without launching the supervisor loop) when preflight
// validation shows signal-cli can never start successfully.
func (c *Channel) Start(ctx context.Context, onMessage func(channels.InboundMessage)) error {
	if err := c.preflight(); err != nil {
		return fmt.Errorf("signal channel preflight failed: %w", err)
	}

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
	backoff := c.initialBackoff
	firstRun := true
	fastExits := 0 // consecutive exits with uptime < fastExitThreshold

	for {
		started := time.Now()
		stderrTail, err := c.runDaemon(ctx, onMessage, firstRun)
		uptime := time.Since(started)
		firstRun = false

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
		c.lastError = formatExitError(err, stderrTail)
		restarts := c.restartCount
		c.mu.Unlock()

		if uptime < fastExitThreshold {
			fastExits++
		} else {
			fastExits = 0
			backoff = c.initialBackoff
		}

		// Circuit breaker: signal-cli dying immediately means a config or
		// environment problem restarting can never fix (unregistered
		// account, bad binary...). Stop supervising after repeated fast
		// failures; the breaker re-arms on daemon restart or config reload.
		if fastExits >= maxFastExits {
			c.mu.Lock()
			c.stopped = true
			c.mu.Unlock()
			ulog.Error("signal-cli failing repeatedly; supervision stopped").Err(err).
				Field("event", "channel.down").
				Field("consecutive_fast_exits", fastExits).
				Field("restart_count", restarts).
				Field("stderr_tail", stderrTail).
				StructuredOnly().Log(ctx)
			return
		}

		ulog.Warn("signal-cli exited unexpectedly").Err(err).
			Field("backoff", backoff.String()).
			Field("restart_count", restarts).
			Field("stderr_tail", stderrTail).
			StructuredOnly().Log(ctx)

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
		if backoff > c.maxBackoff {
			backoff = c.maxBackoff
		}
	}
}

// runDaemon runs one signal-cli daemon process to completion, returning the
// tail of its stderr (for failure logs) alongside the exit error.
func (c *Channel) runDaemon(ctx context.Context, onMessage func(channels.InboundMessage), firstRun bool) (string, error) {
	cmd := exec.CommandContext(ctx, c.config.CLIPath, "-a", c.config.Account, "-o", "json", "daemon", "--socket", "--receive-mode", "on-start") //nolint:gosec // CLIPath is from trusted config
	stderr := &tailBuffer{max: stderrTailBytes}
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return stderr.String(), fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return stderr.String(), fmt.Errorf("start: %w", err)
	}

	startedLog := ulog.Debug("signal-cli started")
	if firstRun {
		startedLog = ulog.Info("signal-cli started").Field("event", "channel.up")
	}
	startedLog.Field("pid", cmd.Process.Pid).StructuredOnly().Log(ctx)

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
			return stderr.String(), ctx.Err()
		default:
		}

		var msg signalMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			ulog.Debug("unparseable line from signal-cli").
				Field("line", scanner.Text()[:min(len(scanner.Text()), 200)]).
				StructuredOnly().Log(ctx)
			continue
		}

		if msg.Envelope.DataMessage == nil || msg.Envelope.DataMessage.Message == "" {
			continue
		}

		sender := msg.Envelope.Source
		if !c.IsAllowed(sender) {
			ulog.Warn("dropped message from unlisted sender").
				Field("sender", sender).StructuredOnly().Log(ctx)
			continue
		}

		var groupID string
		if msg.Envelope.DataMessage.GroupInfo != nil {
			groupID = msg.Envelope.DataMessage.GroupInfo.GroupID
		}
		if groupID != "" && !c.groupMap[groupID] {
			ulog.Warn("dropped message from unlisted group").
				Field("sender", sender).
				Field("group_id", groupID).
				StructuredOnly().Log(ctx)
			continue
		}

		ulog.Info("inbound message received").
			Field("sender", sender).
			Field("group_id", groupID).
			Field("len", len(msg.Envelope.DataMessage.Message)).
			StructuredOnly().Log(ctx)

		inbound := channels.InboundMessage{
			Channel: c.Name(),
			Source:  sender,
			GroupID: groupID,
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

	ulog.Debug("signal-cli stdout scanner exited").
		Field("scanner_err", scanner.Err()).
		StructuredOnly().Log(ctx)

	waitErr := cmd.Wait()
	return stderr.String(), waitErr
}

// Send sends an outbound message via signal-cli's JSON-RPC socket.
// Falls back to spawning a separate signal-cli send process if the socket isn't available.
func (c *Channel) Send(ctx context.Context, req channels.OutboundMessage) (*channels.SendResult, error) {
	socketPath := c.signalSocketPath()
	ulog.Info("Send called").
		Field("recipient", req.Recipient).
		Field("group_id", req.GroupID).
		Field("msg_len", len(req.Message)).
		Field("socket", socketPath).
		StructuredOnly().Log(ctx)

	if socketPath != "" {
		result, err := c.sendViaSocket(socketPath, req.Recipient, req.GroupID, req.Message)
		if err != nil {
			ulog.Error("sendViaSocket failed").Err(err).StructuredOnly().Log(ctx)
		} else {
			ulog.Info("sendViaSocket succeeded").StructuredOnly().Log(ctx)
		}
		return result, err
	}

	ulog.Info("falling back to sendViaCommand").StructuredOnly().Log(ctx)
	return c.sendViaCommand(req.Recipient, req.GroupID, req.Message)
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
			GroupInfo *struct {
				GroupID string `json:"groupId"`
			} `json:"groupInfo"`
			Quote *struct {
				ID     int64  `json:"id"`
				Author string `json:"author"`
				Text   string `json:"text"`
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
func (c *Channel) sendViaSocket(socketPath, recipient, groupID, content string) (*channels.SendResult, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to signal-cli socket: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	params := map[string]any{"message": content}
	if groupID != "" {
		params["groupId"] = groupID
	} else {
		params["recipient"] = []string{recipient}
	}

	request := map[string]any{
		"jsonrpc": "2.0",
		"method":  "send",
		"id":      "1",
		"params":  params,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}
	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write to signal-cli socket: %w", err)
	}
	ulog.Debug("sendViaSocket: wrote request, waiting for response").StructuredOnly().Log(context.Background())

	// Read response to get the timestamp for routing
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		ulog.Debug("sendViaSocket: got response").StructuredOnly().Log(context.Background())
		var resp struct {
			Result struct {
				Timestamp int64 `json:"timestamp"`
				Results   []struct {
					Timestamp int64 `json:"timestamp"`
				} `json:"results"`
			} `json:"result"`
			Error any `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err == nil {
			if resp.Error != nil {
				return nil, fmt.Errorf("signal-cli JSON-RPC error: %v", resp.Error)
			}
			ts := resp.Result.Timestamp
			if ts == 0 && len(resp.Result.Results) > 0 {
				ts = resp.Result.Results[0].Timestamp
			}
			return &channels.SendResult{Timestamp: ts}, nil
		}
	}

	return &channels.SendResult{}, nil
}

// sendViaCommand sends a message by spawning signal-cli send (fallback).
func (c *Channel) sendViaCommand(recipient, groupID, content string) (*channels.SendResult, error) {
	args := []string{"-a", c.config.Account, "send", "-m", content}
	if groupID != "" {
		args = append(args, "-g", groupID)
	} else {
		args = append(args, recipient)
	}
	cmd := exec.Command(c.config.CLIPath, args...) //nolint:gosec // CLIPath is from trusted config
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("signal-cli send failed: %w (output: %s)", err, string(output))
	}
	return &channels.SendResult{}, nil
}

// SendDirect sends a message directly via signal-cli without requiring the daemon to be running.
// This is used by the notify CLI as a standalone fallback.
func SendDirect(cliPath, account, recipient, groupID, message string) error {
	args := []string{"-a", account, "send", "-m", message}
	if groupID != "" {
		args = append(args, "-g", groupID)
	} else {
		args = append(args, recipient)
	}
	cmd := exec.Command(cliPath, args...) //nolint:gosec // CLIPath is from trusted config
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signal-cli send failed: %w (output: %s)", err, string(output))
	}
	return nil
}
