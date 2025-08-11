package notifications

import (
	"fmt"
	"github.com/mattsolo1/grove-notifications/internal/notifiers"
)

// NotificationType represents the type of notification service
type NotificationType string

const (
	TypeSystem NotificationType = "system"
	TypeNtfy   NotificationType = "ntfy"
)

// Config holds configuration for notifications
type Config struct {
	Type NotificationType
	// For ntfy
	NtfyURL   string
	NtfyTopic string
}

// Send sends a notification using the specified configuration
func Send(cfg Config, title, message, priority string, tags []string) error {
	switch cfg.Type {
	case TypeSystem:
		notifier := notifiers.NewSystemNotifier()
		// System notifier doesn't use tags, just level from priority
		return notifier.Send(title, message, priority)
	case TypeNtfy:
		notifier := notifiers.NewNtfyNotifier(cfg.NtfyURL, cfg.NtfyTopic)
		return notifier.Send(title, message, priority, tags)
	default:
		return fmt.Errorf("unsupported notification type: %s", cfg.Type)
	}
}

// SendSystem is a convenience function for sending system notifications
func SendSystem(title, message, level string) error {
	notifier := notifiers.NewSystemNotifier()
	return notifier.Send(title, message, level)
}

// SendNtfy is a convenience function for sending ntfy notifications
func SendNtfy(url, topic, title, message, priority string, tags []string) error {
	notifier := notifiers.NewNtfyNotifier(url, topic)
	return notifier.Send(title, message, priority, tags)
}