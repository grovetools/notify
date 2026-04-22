package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/grovetools/core/pkg/daemon"
	"github.com/grovetools/core/pkg/models"
	"github.com/grovetools/notify/pkg/channels/signal"
	"github.com/grovetools/notify/pkg/config"
	"github.com/spf13/cobra"
)

func newSignalCmd() *cobra.Command {
	var (
		to     string
		direct bool
	)

	cmd := &cobra.Command{
		Use:   "signal [message]",
		Short: "Send a message via Signal",
		Long: `Send a message via Signal messaging.

Routes through the grove daemon for reply-tracking when running inside a flow job.
Falls back to direct signal-cli invocation when daemon is unavailable or --direct is set.

Usage:
  notify signal "Hello from the agent"
  notify signal --to "+1234567890" "Targeted message"
  notify signal --direct "Bypass daemon"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := strings.Join(args, " ")
			cfg := config.Load()

			if !cfg.Signal.Enabled {
				return fmt.Errorf("signal is not enabled in grove.toml [notifications.signal]")
			}

			// If running inside a flow job and not forced direct, route through daemon
			jobID := os.Getenv("GROVE_FLOW_JOB_ID")
			if jobID != "" && !direct {
				err := sendViaDaemon(jobID, to, message)
				if err == nil {
					ulog.Success("Signal message sent via daemon").
						Field("job_id", jobID).
						Pretty("Signal message sent via daemon").
						Emit()
					return nil
				}
				// Fall through to direct send if daemon unreachable
				ulog.Info("Daemon unavailable, falling back to direct send").
					Field("error", err.Error()).
					Emit()
			}

			// Direct send via signal-cli
			recipient := to
			if recipient == "" {
				// Broadcast to all allowlisted contacts
				for _, contact := range cfg.Signal.Allowlist {
					if err := signal.SendDirect(cfg.Signal.CLIPath, cfg.Signal.Account, contact, message); err != nil {
						ulog.Error("Failed to send to contact").
							Field("recipient", contact).
							Field("error", err.Error()).
							Emit()
					}
				}
				ulog.Success("Signal message broadcast").
					Pretty("Signal message sent to all contacts").
					Emit()
				return nil
			}

			if err := signal.SendDirect(cfg.Signal.CLIPath, cfg.Signal.Account, recipient, message); err != nil {
				return fmt.Errorf("failed to send Signal message: %w", err)
			}

			ulog.Success("Signal message sent").
				Field("recipient", recipient).
				Pretty(fmt.Sprintf("Signal message sent to %s", recipient)).
				Emit()
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Recipient phone number (omit to broadcast to allowlist)")
	cmd.Flags().BoolVar(&direct, "direct", false, "Force direct signal-cli send, bypassing daemon")

	return cmd
}

// sendViaDaemon sends a message through the grove daemon's channel API.
// Uses daemon.NewWithAutoStart so GROVE_SCOPE routes to the correct scoped
// daemon (or auto-starts one) instead of hitting the legacy unscoped socket.
func sendViaDaemon(jobID, recipient, message string) error {
	client := daemon.NewWithAutoStart()
	defer client.Close()

	_, err := client.SendChannelMessage(context.Background(), models.ChannelSendRequest{
		JobID:     jobID,
		Recipient: recipient,
		Message:   message,
	})
	return err
}
