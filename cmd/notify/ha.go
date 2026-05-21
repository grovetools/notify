package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/grovetools/core/pkg/daemon"
	"github.com/grovetools/core/pkg/models"
	"github.com/grovetools/notify/pkg/channels"
	"github.com/grovetools/notify/pkg/channels/ha"
	"github.com/grovetools/notify/pkg/config"
	"github.com/spf13/cobra"
)

func newHACmd() *cobra.Command {
	var (
		satellite string
		direct    bool
	)

	cmd := &cobra.Command{
		Use:   "ha [message]",
		Short: "Send a voice announcement via Home Assistant",
		Long: `Send a message to a Home Assistant Voice satellite via assist_satellite.announce.

Routes through the grove daemon when running inside a flow job.
Falls back to direct HTTP call when daemon is unavailable or --direct is set.

Usage:
  notify ha "Build finished successfully"
  notify ha --satellite "assist_satellite.voice_pe_kitchen" "Dinner is ready"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := strings.Join(args, " ")
			cfg := config.Load()

			if !cfg.HomeAssistant.Enabled {
				return fmt.Errorf("home_assistant is not enabled in grove.toml [notifications.home_assistant]")
			}

			jobID := os.Getenv("GROVE_FLOW_JOB_ID")
			if jobID != "" && !direct {
				err := sendHAViaDaemon(jobID, satellite, message)
				if err == nil {
					ulog.Success("HA announcement sent via daemon").
						Field("job_id", jobID).
						Pretty("HA announcement sent via daemon").
						Emit()
					return nil
				}
				ulog.Info("Daemon unavailable, falling back to direct send").
					Field("error", err.Error()).
					Emit()
			}

			target := satellite
			if target == "" {
				target = cfg.HomeAssistant.DefaultSatellite
			}
			if target == "" {
				return fmt.Errorf("no satellite specified and no default configured")
			}

			ch := ha.NewChannel(ha.Config{
				HAURL:            cfg.HomeAssistant.URL,
				HAToken:          cfg.HomeAssistant.Token,
				DefaultSatellite: cfg.HomeAssistant.DefaultSatellite,
			})

			if _, err := ch.Send(context.Background(), channels.OutboundMessage{
				Recipient: target,
				Message:   message,
			}); err != nil {
				return fmt.Errorf("failed to send HA announcement: %w", err)
			}

			ulog.Success("HA announcement sent").
				Field("satellite", target).
				Pretty(fmt.Sprintf("HA announcement sent to %s", target)).
				Emit()
			return nil
		},
	}

	cmd.Flags().StringVar(&satellite, "satellite", "", "Target assist_satellite entity ID (omit to use default)")
	cmd.Flags().BoolVar(&direct, "direct", false, "Force direct HTTP send, bypassing daemon")

	return cmd
}

func sendHAViaDaemon(jobID, satellite, message string) error {
	client := daemon.NewWithAutoStart()
	defer client.Close()

	_, err := client.SendChannelMessage(context.Background(), models.ChannelSendRequest{
		Channel:   "ha",
		JobID:     jobID,
		JobTitle:  os.Getenv("GROVE_FLOW_JOB_TITLE"),
		Recipient: satellite,
		Message:   message,
	})
	return err
}
