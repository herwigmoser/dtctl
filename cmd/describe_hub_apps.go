package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dynatrace-oss/dtctl/pkg/resources/hub"
)

// describeHubAppCmd shows detailed info about a Hub catalog app
var describeHubAppCmd = &cobra.Command{
	Use:     "hub-apps <id>",
	Aliases: []string{"hub-app"},
	Short:   "Show details of a Dynatrace Hub app",
	Long: `Show detailed information about a Dynatrace Hub catalog app.

Examples:
  # Describe a Hub app (JSON output by default)
  dtctl describe hub-apps my-app-id

  # Output as YAML
  dtctl describe hub-apps my-app-id -o yaml
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		_, c, printer, err := Setup()
		if err != nil {
			return err
		}

		handler := hub.NewHandler(c)

		app, err := handler.GetApp(id)
		if err != nil {
			return err
		}
		return printer.Print(app)
	},
}
