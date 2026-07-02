package signal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grovetools/notify/pkg/channels"
)

// writeFakeCLI writes an executable shell script to dir and returns its path.
// The name deliberately does not contain "signal-cli" so the orphan-killing
// pkill pattern can never match test processes.
func writeFakeCLI(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-cli")
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeAccountsJSON creates <dataHome>/signal-cli/data/accounts.json and
// points XDG_DATA_HOME at dataHome for the duration of the test.
func writeAccountsJSON(t *testing.T, numbers ...string) {
	t.Helper()
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	dir := filepath.Join(dataHome, "signal-cli", "data")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	entries := make([]string, 0, len(numbers))
	for _, n := range numbers {
		entries = append(entries, fmt.Sprintf(`{"number":%q,"path":%q}`, n, n))
	}
	content := fmt.Sprintf(`{"accounts":[%s],"version":2}`, strings.Join(entries, ","))
	if err := os.WriteFile(filepath.Join(dir, "accounts.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTailBufferKeepsTail(t *testing.T) {
	tb := &tailBuffer{max: 8}
	if _, err := tb.Write([]byte("0123456789abcdef")); err != nil {
		t.Fatal(err)
	}
	if got := tb.String(); got != "89abcdef" {
		t.Fatalf("tail = %q, want %q", got, "89abcdef")
	}
	if _, err := tb.Write([]byte("XY")); err != nil {
		t.Fatal(err)
	}
	if got := tb.String(); got != "abcdefXY" {
		t.Fatalf("tail after second write = %q, want %q", got, "abcdefXY")
	}
}

func TestPreflightNoAccount(t *testing.T) {
	c := NewChannel(Config{CLIPath: "/usr/bin/true", Account: ""})
	if err := c.preflight(); err == nil || !strings.Contains(err.Error(), "account") {
		t.Fatalf("want account error, got %v", err)
	}
}

func TestPreflightMissingBinary(t *testing.T) {
	c := NewChannel(Config{CLIPath: filepath.Join(t.TempDir(), "nope"), Account: "+15550001111"})
	if err := c.preflight(); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestPreflightNonExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cli")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewChannel(Config{CLIPath: path, Account: "+15550001111"})
	if err := c.preflight(); err == nil || !strings.Contains(err.Error(), "not an executable") {
		t.Fatalf("want non-executable error, got %v", err)
	}
}

func TestPreflightUnregisteredAccount(t *testing.T) {
	writeAccountsJSON(t) // empty accounts list
	cli := writeFakeCLI(t, t.TempDir(), "exit 0")
	c := NewChannel(Config{CLIPath: cli, Account: "+15550001111"})
	err := c.preflight()
	if err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("want not-registered error, got %v", err)
	}
}

func TestPreflightRegisteredAccount(t *testing.T) {
	writeAccountsJSON(t, "+15550001111")
	cli := writeFakeCLI(t, t.TempDir(), "exit 0")
	c := NewChannel(Config{CLIPath: cli, Account: "+15550001111"})
	if err := c.preflight(); err != nil {
		t.Fatalf("preflight should pass for registered account, got %v", err)
	}
}

func TestCheckRegistrationCLIFallback(t *testing.T) {
	// No accounts.json anywhere under this XDG_DATA_HOME.
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	dir := t.TempDir()
	registered := writeFakeCLI(t, dir, `echo '[{"number":"+15550001111"}]'`)
	c := NewChannel(Config{CLIPath: registered, Account: "+15550001111"})
	if err := c.checkRegistration(); err != nil {
		t.Fatalf("CLI probe lists account, want nil, got %v", err)
	}

	empty := writeFakeCLI(t, t.TempDir(), `echo '[]'`)
	c2 := NewChannel(Config{CLIPath: empty, Account: "+15550001111"})
	if err := c2.checkRegistration(); err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("CLI probe omits account, want not-registered error, got %v", err)
	}

	// Probe fails to run: inconclusive, must not block startup.
	failing := writeFakeCLI(t, t.TempDir(), "exit 3")
	c3 := NewChannel(Config{CLIPath: failing, Account: "+15550001111"})
	if err := c3.checkRegistration(); err != nil {
		t.Fatalf("inconclusive probe should pass, got %v", err)
	}
}

func TestRunDaemonCapturesStderrTail(t *testing.T) {
	cli := writeFakeCLI(t, t.TempDir(), `echo "boom: account not registered" >&2; exit 1`)
	c := NewChannel(Config{CLIPath: cli, Account: "+15550001111"})
	tail, err := c.runDaemon(context.Background(), func(channels.InboundMessage) {}, true)
	if err == nil {
		t.Fatal("want exit error from fake CLI")
	}
	if !strings.Contains(tail, "boom: account not registered") {
		t.Fatalf("stderr tail = %q, want it to contain the fake CLI's stderr", tail)
	}
}

func TestCircuitBreakerStopsSupervision(t *testing.T) {
	cli := writeFakeCLI(t, t.TempDir(), "exit 1")
	c := NewChannel(Config{CLIPath: cli, Account: "+15550001111"})
	c.initialBackoff = time.Millisecond
	c.maxBackoff = 4 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		c.supervisorLoop(ctx, func(channels.InboundMessage) {})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("supervisorLoop did not stop; circuit breaker never tripped")
	}

	st := c.Status()
	if st.IsAlive {
		t.Fatal("channel reported alive after breaker tripped")
	}
	if st.RestartCount != maxFastExits {
		t.Fatalf("restart count = %d, want %d", st.RestartCount, maxFastExits)
	}
}
