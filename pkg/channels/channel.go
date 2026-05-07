// Package channels provides bidirectional communication channel abstractions
// for external messaging services (Signal, Telegram, Slack, etc.).
package channels

import (
	"context"
	"time"
)

// Quote represents a reply/quote reference in an inbound message.
type Quote struct {
	ID     int64  // Primary routing key (e.g., Signal message timestamp)
	Author string // Who sent the original quoted message
	Text   string // Quoted text (may be truncated by the channel)
}

// InboundMessage represents a message received from an external channel.
type InboundMessage struct {
	Channel string // Channel name, e.g. "signal"
	Source  string // Channel-native sender ID (e.g., phone number)
	GroupID string // Non-empty if message came from a group
	Message string // Message text
	Quote   *Quote // Non-nil if this is a reply to a previous message
}

// OutboundMessage represents a message to be sent via an external channel.
type OutboundMessage struct {
	Recipient string // Channel-native recipient ID
	GroupID   string // Signal group ID; when set, sends to group instead of recipient
	Message   string // Message text
}

// SendResult contains metadata from a successful send operation.
type SendResult struct {
	Timestamp int64 // Channel-native message ID (e.g., Signal timestamp) for routing table
}

// ChannelStatus reports the health and supervision state of a channel.
type ChannelStatus struct {
	IsAlive       bool
	RestartCount  int
	LastRestartAt time.Time
}

// Channel is a bidirectional communication channel with an external messaging service.
// Implementations handle the raw IPC with the external service and convert
// native message formats into the generic types above.
type Channel interface {
	Name() string
	Start(ctx context.Context, onMessage func(InboundMessage)) error
	Send(ctx context.Context, req OutboundMessage) (*SendResult, error)
	Stop(ctx context.Context) error
	Status() ChannelStatus
}
