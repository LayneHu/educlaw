package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/pingjie/educlaw/cmd/educlaw/internal/onboard"
	"github.com/pingjie/educlaw/cmd/educlaw/internal/serve"
	"github.com/pingjie/educlaw/cmd/educlaw/internal/version"
)

const banner = `
  ___    _         ___  _
 | __|__| |_  _ / __|/ |_____ __ __
 | _|/ _` + "`" + ` | || | || (__| |/ _ \ V  V /
 |___\__,_|\_,_|\___\_|\___/\_/\_/

 AI-powered Education Platform v%s
`

func main() {
	root := &cobra.Command{
		Use:   "educlaw",
		Short: "EduClaw - AI Education Platform",
		Long:  fmt.Sprintf(banner, version.Version),
	}

	root.AddCommand(serve.Command())
	root.AddCommand(onboard.Command())
	root.AddCommand(version.Command())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
