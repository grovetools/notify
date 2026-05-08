package notifications

import (
	"github.com/grovetools/notify/pkg/channels/signal"
	"github.com/grovetools/notify/pkg/config"
)

// AgentInstructions returns the agent instructions for all enabled channels,
// respecting config overrides (replace or append). signalTarget is the
// human-readable name of the configured signal target for this session.
func AgentInstructions(cfg *config.NotificationsConfig, enabledChannels []string, signalTarget string) string {
	var instructions string

	for _, ch := range enabledChannels {
		switch ch {
		case "signal":
			instructions += signalInstructions(cfg, signalTarget)
		}
	}

	return instructions
}

func signalInstructions(cfg *config.NotificationsConfig, signalTarget string) string {
	// Full replacement if configured
	if cfg.Signal.AgentInstructions != "" {
		return cfg.Signal.AgentInstructions
	}

	result := signal.DefaultAgentInstructions(signalTarget)

	// Per-target instructions (e.g. "this group has non-technical members")
	if signalTarget != "" {
		if targetInstr := cfg.Signal.TargetInstructions(signalTarget); targetInstr != "" {
			result += "\n\n" + targetInstr
		}
	}

	// Append extra instructions if configured
	if cfg.Signal.AppendInstructions != "" {
		result += "\n\n" + cfg.Signal.AppendInstructions
	}

	return result
}
