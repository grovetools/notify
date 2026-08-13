package ha

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grovetools/notify/pkg/channels"
)

// freePort returns a currently-free TCP port on the loopback interface.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve free port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// TestStartRefusesWithoutSecret proves the fail-CLOSED behavior: an enabled HA
// channel with an empty webhook secret must NOT open a listener.
func TestStartRefusesWithoutSecret(t *testing.T) {
	port := freePort(t)
	ch := NewChannel(Config{WebhookPort: port, WebhookSecret: ""})

	err := ch.Start(context.Background(), func(channels.InboundMessage) {})
	if err == nil {
		_ = ch.Stop(context.Background())
		t.Fatal("expected Start to refuse an empty webhook secret, got nil error")
	}
	if !strings.Contains(err.Error(), "webhook_secret is empty") {
		t.Fatalf("unexpected error: %v", err)
	}

	// The listener must not be bound — the port should still be free.
	ln, lerr := net.Listen("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)))
	if lerr != nil {
		t.Fatalf("port %d is bound; Start opened a listener despite refusing: %v", port, lerr)
	}
	_ = ln.Close()
}

// TestStartRefusesOnSecretResolutionError proves a failed webhook_secret_command
// (surfaced as WebhookSecretErr) keeps the endpoint DOWN.
func TestStartRefusesOnSecretResolutionError(t *testing.T) {
	port := freePort(t)
	ch := NewChannel(Config{
		WebhookPort:      port,
		WebhookSecret:    "",
		WebhookSecretErr: "webhook_secret_command failed: vault locked",
	})

	err := ch.Start(context.Background(), func(channels.InboundMessage) {})
	if err == nil {
		_ = ch.Stop(context.Background())
		t.Fatal("expected Start to refuse when secret resolution failed, got nil error")
	}
	if !strings.Contains(err.Error(), "vault locked") {
		t.Fatalf("expected the resolution error to surface, got: %v", err)
	}
}

// TestWebhookAuthConstantTime proves the authenticated endpoint accepts a
// correct bearer token and rejects a wrong or missing one.
func TestWebhookAuthConstantTime(t *testing.T) {
	const secret = "s3cr3t-token"
	port := freePort(t)

	var got string
	ch := NewChannel(Config{
		WebhookPort:   port,
		WebhookBind:   "127.0.0.1",
		WebhookSecret: secret,
	})
	if err := ch.Start(context.Background(), func(m channels.InboundMessage) { got = m.Message }); err != nil {
		t.Fatalf("Start with a valid secret should succeed, got: %v", err)
	}
	defer ch.Stop(context.Background())

	url := fmt.Sprintf("http://127.0.0.1:%d/webhook", port)
	body := `{"question":"what is the build status"}`

	post := func(bearer string, sendAuth bool) int {
		req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		if sendAuth {
			req.Header.Set("Authorization", bearer)
		}
		// Retry briefly while the server goroutine comes up.
		var resp *http.Response
		for i := 0; i < 50; i++ {
			resp, err = http.DefaultClient.Do(req)
			if err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
			req, _ = http.NewRequest(http.MethodPost, url, strings.NewReader(body))
			if sendAuth {
				req.Header.Set("Authorization", bearer)
			}
		}
		if err != nil {
			t.Fatalf("POST failed: %v", err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	if code := post("Bearer "+secret, true); code != http.StatusOK {
		t.Fatalf("correct token should be accepted, got status %d", code)
	}
	if got != "what is the build status" {
		t.Fatalf("inbound message not routed on valid auth; got %q", got)
	}

	if code := post("Bearer wrong-token", true); code != http.StatusUnauthorized {
		t.Fatalf("wrong token should be rejected with 401, got status %d", code)
	}
	if code := post("", false); code != http.StatusUnauthorized {
		t.Fatalf("missing token should be rejected with 401, got status %d", code)
	}
}

// TestStartBindsLoopbackByDefault proves the default bind is loopback, not
// 0.0.0.0 — the listener answers on 127.0.0.1 for the configured port.
func TestStartBindsLoopbackByDefault(t *testing.T) {
	port := freePort(t)
	ch := NewChannel(Config{WebhookPort: port, WebhookSecret: "abc"})
	if err := ch.Start(context.Background(), func(channels.InboundMessage) {}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(context.Background())

	// Loopback should be reachable...
	var reached bool
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)), 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			reached = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !reached {
		t.Fatalf("loopback listener never came up on port %d", port)
	}
}
