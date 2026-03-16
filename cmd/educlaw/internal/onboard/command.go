package onboard

import (
	"github.com/spf13/cobra"
)

var configPath string

// Command returns the onboard cobra command.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Register a new student, parent, or teacher",
		Long:  "Interactively create a new actor (student/parent/teacher) and initialize their workspace.",
		RunE:  runOnboard,
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")
	return cmd
}
