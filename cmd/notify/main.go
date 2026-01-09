package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mattsolo1/grove-core/cli"
	grovelogging "github.com/mattsolo1/grove-core/logging"
	"github.com/mattsolo1/grove-notifications"
	"github.com/mattsolo1/grove-notifications/cmd"
	"github.com/spf13/cobra"
)

var ulog = grovelogging.NewUnifiedLogger("grove-notifications")

func main() {
	rootCmd := cli.NewStandardCommand(
		"notify",
		"Notification system for Grove ecosystem",
	)
	
	// Add subcommands
	rootCmd.AddCommand(newSystemCmd())
	rootCmd.AddCommand(newNtfyCmd())
	rootCmd.AddCommand(cmd.NewVersionCmd())
	
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newSystemCmd() *cobra.Command {
	var level string
	
	cmd := &cobra.Command{
		Use:   "system [args...]",
		Short: "Send a system notification",
		Long: `Send a system notification with title and message.
Usage:
  notify system --title "Title" "Message text"
  notify system --title "Title" --level info "Message text"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			title, _ := cmd.Flags().GetString("title")
			if title == "" {
				title = "Grove Notification"
			}

			message := strings.Join(args, " ")

			if err := notifications.SendSystem(title, message, level); err != nil {
				return fmt.Errorf("failed to send system notification: %w", err)
			}

			ulog.Success("System notification sent").
				Field("title", title).
				Field("message", message).
				Field("level", level).
				Pretty(fmt.Sprintf("System notification sent: %s - %s", title, message)).
				Log(ctx)
			return nil
		},
	}
	
	cmd.Flags().String("title", "Grove Notification", "Notification title")
	cmd.Flags().StringVar(&level, "level", "info", "Notification level (info, warning, error)")
	
	return cmd
}

func newNtfyCmd() *cobra.Command {
	var (
		topic    string
		title    string
		priority string
		tags     []string
		url      string
	)
	
	cmd := &cobra.Command{
		Use:   "ntfy [args...]",
		Short: "Send a notification via ntfy",
		Long: `Send a notification via ntfy service.
Usage:
  notify ntfy --topic mytopic --title "Title" "Message text"
  notify ntfy --topic mytopic --title "Title" --priority high --tags tag1,tag2 "Message text"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			message := strings.Join(args, " ")

			if url == "" {
				url = "https://ntfy.sh" // Default ntfy server
			}

			if err := notifications.SendNtfy(url, topic, title, message, priority, tags); err != nil {
				return fmt.Errorf("failed to send ntfy notification: %w", err)
			}

			ulog.Success("Ntfy notification sent").
				Field("topic", topic).
				Field("title", title).
				Field("message", message).
				Field("priority", priority).
				Field("tags", tags).
				Field("url", url).
				Pretty(fmt.Sprintf("Ntfy notification sent to topic '%s': %s - %s", topic, title, message)).
				Log(ctx)
			return nil
		},
	}
	
	cmd.Flags().StringVar(&topic, "topic", "", "Ntfy topic (required)")
	cmd.Flags().StringVar(&title, "title", "Grove Notification", "Notification title")
	cmd.Flags().StringVar(&priority, "priority", "default", "Priority (min, low, default, high, urgent)")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "Comma-separated tags")
	cmd.Flags().StringVar(&url, "url", "", "Ntfy server URL (default: https://ntfy.sh)")
	cmd.MarkFlagRequired("topic")
	
	return cmd
}