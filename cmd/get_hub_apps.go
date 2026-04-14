package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dynatrace-oss/dtctl/pkg/resources/hub"
)

// getHubAppsCmd retrieves Hub catalog apps
var getHubAppsCmd = &cobra.Command{
	Use:     "hub-apps [id]",
	Aliases: []string{"hub-app"},
	Short:   "Get Dynatrace Hub catalog apps",
	Long: `Get Dynatrace Hub catalog apps.

Examples:
  # List all Hub apps
  dtctl get hub-apps

  # Get a specific Hub app by ID
  dtctl get hub-apps my-app-id

  # Output as JSON
  dtctl get hub-apps -o json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, c, printer, err := Setup()
		if err != nil {
			return err
		}

		handler := hub.NewHandler(c)

		if len(args) > 0 {
			app, err := handler.GetApp(args[0])
			if err != nil {
				return err
			}
			return printer.Print(app)
		}

		list, err := handler.ListApps(GetChunkSize())
		if err != nil {
			return err
		}
		return printer.PrintList(list.Items)
	},
}

// getHubAppReleasesCmd retrieves releases for a Hub catalog app
var getHubAppReleasesCmd = &cobra.Command{
	Use:     "hub-app-releases <id>",
	Aliases: []string{"hub-app-release"},
	Short:   "Get releases for a Dynatrace Hub app",
	Long: `Get releases for a Dynatrace Hub catalog app.

Examples:
  # List all releases for a Hub app
  dtctl get hub-app-releases my-app-id

  # Output as JSON
  dtctl get hub-app-releases my-app-id -o json
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires an app ID argument")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		_, c, printer, err := Setup()
		if err != nil {
			return err
		}

		handler := hub.NewHandler(c)

		list, err := handler.ListAppReleases(id, GetChunkSize())
		if err != nil {
			return err
		}
		return printer.PrintList(list.Items)
	},
}
