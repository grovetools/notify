package config

import (
	"log"

	"github.com/grovetools/core/config"
)

//go:generate sh -c "cd ../.. && go run ./tools/schema-generator/"

// NotificationsConfig represents the structure of the 'notifications' section in grove.yml
type NotificationsConfig struct {
	Ntfy   NtfyConfig   `yaml:"ntfy" jsonschema:"description=ntfy.sh push notification settings" jsonschema_extras:"x-layer=global,x-priority=70"`
	System SystemConfig `yaml:"system" jsonschema:"description=Native system notification settings" jsonschema_extras:"x-layer=global,x-priority=71"`
}

// NtfyConfig holds settings for ntfy.sh notifications.
type NtfyConfig struct {
	Enabled bool   `yaml:"enabled" jsonschema:"description=Enable ntfy.sh push notifications,default=false" jsonschema_extras:"x-layer=global,x-priority=70,x-important=true"`
	Topic   string `yaml:"topic" jsonschema:"description=ntfy.sh topic name for notifications" jsonschema_extras:"x-layer=global,x-priority=71,x-important=true"`
	URL     string `yaml:"url" jsonschema:"description=ntfy.sh server URL,default=https://ntfy.sh" jsonschema_extras:"x-layer=global,x-priority=72,x-important=true"`
}

// SystemConfig holds settings for native system notifications.
type SystemConfig struct {
	// Levels specifies which notification levels should trigger a system notification.
	// e.g., ["error", "warning"]
	Levels []string `yaml:"levels" jsonschema:"description=Notification levels that trigger system notifications" jsonschema_extras:"x-layer=global,x-priority=73"`
}

// Load reads the merged grove configuration and parses the 'notifications' extension.
func Load() *NotificationsConfig {
	// Start with a safe default configuration.
	cfg := defaultConfig()

	// Use grove-core to load the complete, merged configuration from the environment.
	coreCfg, err := config.LoadDefault()
	if err != nil {
		// It's common for no config to exist; this is not a fatal error.
		// We'll proceed with the defaults.
		log.Printf("Debug: No grove config found, using default notification settings: %v", err)
		return cfg
	}

	// Unmarshal the 'notifications' key from the Extensions map into our struct.
	var userCfg NotificationsConfig
	if err := coreCfg.UnmarshalExtension("notifications", &userCfg); err != nil {
		log.Printf("Warning: could not parse 'notifications' config section: %v. Using defaults.", err)
		return cfg
	}

	// Merge user-provided values over the defaults.
	// For bools, we need to check if the user explicitly set them
	// Since Go's zero value for bool is false, we check other fields to determine intent
	if userCfg.Ntfy.Topic != "" || userCfg.Ntfy.URL != "" {
		cfg.Ntfy.Enabled = userCfg.Ntfy.Enabled
	}
	if userCfg.Ntfy.Topic != "" {
		cfg.Ntfy.Topic = userCfg.Ntfy.Topic
	}
	if userCfg.Ntfy.URL != "" {
		cfg.Ntfy.URL = userCfg.Ntfy.URL
	}
	if len(userCfg.System.Levels) > 0 {
		cfg.System.Levels = userCfg.System.Levels
	}

	return cfg
}

func defaultConfig() *NotificationsConfig {
	return &NotificationsConfig{
		Ntfy: NtfyConfig{
			Enabled: false,
			Topic:   "",
			URL:     "https://ntfy.sh",
		},
		System: SystemConfig{
			Levels: []string{"error", "warning"},
		},
	}
}
