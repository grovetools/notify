package notifications

import (
	"github.com/grovetools/notify/pkg/channels/signal"
	"github.com/grovetools/notify/pkg/config"
)

// AgentInstructions returns the agent instructions for all enabled channels,
// respecting config overrides (replace or append).
func AgentInstructions(cfg *config.NotificationsConfig, enabledChannels []string) string {
	var instructions string

	for _, ch := range enabledChannels {
		switch ch {
		case "signal":
			instructions += signalInstructions(cfg)
		}
	}

	return instructions
}

func signalInstructions(cfg *config.NotificationsConfig) string {
	// Full replacement if configured
	if cfg.Signal.AgentInstructions != "" {
		return cfg.Signal.AgentInstructions
	}

	// Default instructions with account number
	result := signal.DefaultAgentInstructions(cfg.Signal.Account)

	// Append extra instructions if configured
	if cfg.Signal.AppendInstructions != "" {
		result += "\n\n" + cfg.Signal.AppendInstructions
	}

	return result
}
