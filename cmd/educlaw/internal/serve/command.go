package serve

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var configPath string

// Command returns the serve cobra command.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the EduClaw web server",
		Long:  "Start the EduClaw HTTP server with AI agent backend.",
		RunE:  runServe,
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")
	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	srv, addr, err := SetupServer(configPath)
	if err != nil {
		return fmt.Errorf("server setup: %w", err)
	}

	fmt.Printf("\n🎓 EduClaw is running!\n")
	fmt.Printf("   Student portal:  http://%s/student\n", addr)
	fmt.Printf("   Parent portal:   http://%s/parent\n", addr)
	fmt.Printf("   Teacher portal:  http://%s/teacher\n", addr)
	fmt.Printf("\nPress Ctrl+C to stop.\n\n")

	log.Printf("Starting server on %s", addr)
	return srv.Run(addr)
}
