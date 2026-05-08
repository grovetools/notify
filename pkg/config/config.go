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
	Signal SignalConfig `yaml:"signal" jsonschema:"description=Signal messaging channel settings" jsonschema_extras:"x-layer=global,x-priority=72"`
}

// NtfyConfig holds settings for ntfy.sh notifications.
type NtfyConfig struct {
	Enabled bool   `yaml:"enabled" jsonschema:"description=Enable ntfy.sh push notifications,default=false" jsonschema_extras:"x-layer=global,x-priority=70,x-important=true"`
	Topic   string `yaml:"topic" jsonschema:"description=ntfy.sh topic name for notifications" jsonschema_extras:"x-layer=global,x-priority=71,x-important=true"`
	URL     string `yaml:"url" jsonschema:"description=ntfy.sh server URL,default=https://ntfy.sh" jsonschema_extras:"x-layer=global,x-priority=72,x-important=true"`
}

// SignalContact represents a named Signal contact.
type SignalContact struct {
	Phone        string `yaml:"phone" jsonschema:"description=Phone number for this contact"`
	Instructions string `yaml:"instructions" jsonschema:"description=Custom agent instructions when targeting this contact"`
}

// SignalGroup represents a named Signal group.
type SignalGroup struct {
	ID           string `yaml:"id" jsonschema:"description=Base64 group ID"`
	Instructions string `yaml:"instructions" jsonschema:"description=Custom agent instructions when targeting this group"`
}

// SignalConfig holds settings for Signal messaging channel.
type SignalConfig struct {
	Enabled            bool                      `yaml:"enabled" jsonschema:"description=Enable Signal messaging channel,default=false" jsonschema_extras:"x-layer=global,x-priority=72,x-important=true"`
	CLIPath            string                    `yaml:"cli_path" jsonschema:"description=Path to signal-cli binary,default=/usr/local/bin/signal-cli" jsonschema_extras:"x-layer=global,x-priority=73"`
	Account            string                    `yaml:"account" jsonschema:"description=Signal account phone number" jsonschema_extras:"x-layer=global,x-priority=74,x-important=true"`
	Allowlist          []string                  `yaml:"allowlist" jsonschema:"description=Authorized sender phone numbers" jsonschema_extras:"x-layer=global,x-priority=75"`
	Groups             []string                  `yaml:"groups" jsonschema:"description=Authorized Signal group IDs (base64)" jsonschema_extras:"x-layer=global,x-priority=75"`
	Contacts           map[string]SignalContact  `yaml:"contacts" jsonschema:"description=Named contacts mapping name to phone number" jsonschema_extras:"x-layer=global,x-priority=75"`
	NamedGroups        map[string]SignalGroup    `yaml:"named_groups,omitempty" json:"named_groups,omitempty" jsonschema:"-"`
	AgentInstructions  string                    `yaml:"agent_instructions" jsonschema:"description=Custom agent instructions for Signal (replaces default if set)" jsonschema_extras:"x-layer=global,x-priority=76"`
	AppendInstructions string                    `yaml:"append_instructions" jsonschema:"description=Additional instructions appended to the default Signal agent instructions" jsonschema_extras:"x-layer=global,x-priority=77"`
}

// ContactsFlat returns a flat name→phone map for use by the channel manager.
func (c *SignalConfig) ContactsFlat() map[string]string {
	m := make(map[string]string, len(c.Contacts))
	for name, contact := range c.Contacts {
		m[name] = contact.Phone
	}
	return m
}

// TargetInstructions returns custom instructions for a named target, or empty string.
func (c *SignalConfig) TargetInstructions(name string) string {
	if g, ok := c.NamedGroups[name]; ok && g.Instructions != "" {
		return g.Instructions
	}
	if ct, ok := c.Contacts[name]; ok && ct.Instructions != "" {
		return ct.Instructions
	}
	return ""
}

// NamedGroupsFlat returns a flat name→groupID map for use by the channel manager.
func (c *SignalConfig) NamedGroupsFlat() map[string]string {
	m := make(map[string]string, len(c.NamedGroups))
	for name, group := range c.NamedGroups {
		m[name] = group.ID
	}
	return m
}

// AutonomousDefaults holds default settings for autonomous idle pinging.
type AutonomousDefaults struct {
	DefaultPrompt string `yaml:"default_prompt" jsonschema:"description=Default idle ping prompt when not specified per-job" jsonschema_extras:"x-layer=global,x-priority=78"`
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

	// Signal config
	if userCfg.Signal.Account != "" || userCfg.Signal.CLIPath != "" {
		cfg.Signal.Enabled = userCfg.Signal.Enabled
	}
	if userCfg.Signal.CLIPath != "" {
		cfg.Signal.CLIPath = userCfg.Signal.CLIPath
	}
	if userCfg.Signal.Account != "" {
		cfg.Signal.Account = userCfg.Signal.Account
	}
	if len(userCfg.Signal.Allowlist) > 0 {
		cfg.Signal.Allowlist = userCfg.Signal.Allowlist
	}
	if len(userCfg.Signal.Groups) > 0 {
		cfg.Signal.Groups = userCfg.Signal.Groups
	}
	if len(userCfg.Signal.Contacts) > 0 {
		cfg.Signal.Contacts = userCfg.Signal.Contacts
	}
	if len(userCfg.Signal.NamedGroups) > 0 {
		cfg.Signal.NamedGroups = userCfg.Signal.NamedGroups
	}

	// Auto-populate allowlist/groups from named entries (dedup-safe)
	existing := make(map[string]bool)
	for _, v := range cfg.Signal.Allowlist {
		existing[v] = true
	}
	for _, contact := range cfg.Signal.Contacts {
		if contact.Phone != "" && !existing[contact.Phone] {
			cfg.Signal.Allowlist = append(cfg.Signal.Allowlist, contact.Phone)
			existing[contact.Phone] = true
		}
	}
	existingGroups := make(map[string]bool)
	for _, v := range cfg.Signal.Groups {
		existingGroups[v] = true
	}
	for _, group := range cfg.Signal.NamedGroups {
		if group.ID != "" && !existingGroups[group.ID] {
			cfg.Signal.Groups = append(cfg.Signal.Groups, group.ID)
			existingGroups[group.ID] = true
		}
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
		Signal: SignalConfig{
			Enabled: false,
			CLIPath: "/usr/local/bin/signal-cli",
		},
	}
}
