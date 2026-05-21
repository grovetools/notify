// Package ha implements the Home Assistant Voice channel.
// Inbound: receives webhook POSTs from HA sentence triggers.
// Outbound: calls HA's assist_satellite.announce service.
package ha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/notify/pkg/channels"
)

var ulog = grovelogging.NewUnifiedLogger("groved.ha")

// Config holds Home Assistant channel configuration.
type Config struct {
	WebhookPort      int
	WebhookSecret    string
	HAURL            string
	HAToken          string
	DefaultSatellite string
}

// webhookPayload is the JSON body HA sends to the webhook.
type webhookPayload struct {
	Question        string `json:"question"`
	SourceSatellite string `json:"source_satellite,omitempty"`
}

// announcePayload is the JSON body sent to HA's announce service.
type announcePayload struct {
	EntityID string `json:"entity_id"`
	Message  string `json:"message"`
}

// Channel implements channels.Channel for Home Assistant Voice.
type Channel struct {
	config   Config
	mu       sync.RWMutex
	server   *http.Server
	listener net.Listener
	running  bool
	cancel   context.CancelFunc

	restartCount  int
	lastRestartAt time.Time
	alive         bool
	httpClient    *http.Client
}

// NewChannel creates a new Home Assistant channel with the given configuration.
func NewChannel(cfg Config) *Channel {
	return &Channel{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Channel) Name() string { return "ha" }

func (c *Channel) Start(ctx context.Context, onMessage func(channels.InboundMessage)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("ha channel already running")
	}

	ctx, c.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if c.config.WebhookSecret != "" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+c.config.WebhookSecret {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		var payload webhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if payload.Question == "" {
			http.Error(w, "missing question field", http.StatusBadRequest)
			return
		}

		source := payload.SourceSatellite
		if source == "" {
			source = c.config.DefaultSatellite
		}

		ulog.Info("Inbound HA webhook").
			Field("source", source).
			Field("text_len", len(payload.Question)).
			Log(ctx)

		onMessage(channels.InboundMessage{
			Channel: "ha",
			Source:  source,
			Message: payload.Question,
		})

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	addr := fmt.Sprintf(":%d", c.config.WebhookPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	c.listener = ln

	c.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	c.running = true
	c.alive = true

	go func() {
		ulog.Info("HA webhook server started").Field("addr", addr).Log(ctx)
		if err := c.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			ulog.Error("HA webhook server error").Err(err).Log(ctx)
			c.mu.Lock()
			c.alive = false
			c.mu.Unlock()
		}
	}()

	go func() {
		<-ctx.Done()
		_ = c.server.Close()
	}()

	return nil
}

func (c *Channel) Send(ctx context.Context, req channels.OutboundMessage) (*channels.SendResult, error) {
	satellite := req.Recipient
	if satellite == "" {
		satellite = c.config.DefaultSatellite
	}
	if satellite == "" {
		return nil, fmt.Errorf("no satellite entity specified and no default configured")
	}

	payload := announcePayload{
		EntityID: satellite,
		Message:  req.Message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal announce payload: %w", err)
	}

	url := c.config.HAURL + "/api/services/assist_satellite/announce"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.HAToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HA announce request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HA announce returned %d", resp.StatusCode)
	}

	ulog.Info("HA announce sent").
		Field("satellite", satellite).
		Field("msg_len", len(req.Message)).
		Log(ctx)

	return &channels.SendResult{Timestamp: time.Now().UnixMilli()}, nil
}

func (c *Channel) Stop(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	if c.server != nil {
		_ = c.server.Close()
	}
	c.running = false
	c.alive = false
	return nil
}

func (c *Channel) Status() channels.ChannelStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return channels.ChannelStatus{
		IsAlive:       c.alive,
		RestartCount:  c.restartCount,
		LastRestartAt: c.lastRestartAt,
	}
}
