package cmd

import (
	"encoding/json"
	"fmt"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/version"
	"github.com/spf13/cobra"
)

var ulog = grovelogging.NewUnifiedLogger("grove-notifications")

func NewVersionCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information for this binary",
		RunE: func(cmd *cobra.Command, args []string) error {
	info := version.GetInfo()

			if jsonOutput {
				jsonData, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal version info to JSON: %w", err)
				}
				ulog.Info("Version information (JSON)").
					Field("version", info.Version).
					Field("commit", info.Commit).
					Field("build_date", info.BuildDate).
					Pretty(string(jsonData)).
					PrettyOnly().
					Emit()
			} else {
				ulog.Info("Version information").
					Field("version", info.Version).
					Field("commit", info.Commit).
					Field("build_date", info.BuildDate).
					Pretty(info.String()).
					PrettyOnly().
					Emit()
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output version information in JSON format")

	return cmd
}