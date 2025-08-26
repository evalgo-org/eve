package cli

import (
	eve "eve.evalgo.org/common"
	"eve.evalgo.org/assets"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(invCmd)
	invCmd.Flags().String("url", "http://inventory.platform.px", "inventory base url to be used for all requests")
	invCmd.Flags().String("token", "", "token to authorize agains the inventory system")
	invCmd.Flags().String("elements", "components", "list all components")
}

var invCmd = &cobra.Command{
	Use:   "inv",
	Short: "read and write to inventory",
	Long:  `utilize the api of the inventory system to read and write information to it`,
	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")
		token, _ := cmd.Flags().GetString("token")
		elements, _ := cmd.Flags().GetString("elements")
		if elements == "components" {
			eve.Logger.Info(assets.InvComponents(url, token))
		}
	},
}
