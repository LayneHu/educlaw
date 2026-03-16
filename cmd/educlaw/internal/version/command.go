package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.1.0"

// Command returns the version cobra command.
func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the EduClaw version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("EduClaw v%s\n", Version)
		},
	}
}
