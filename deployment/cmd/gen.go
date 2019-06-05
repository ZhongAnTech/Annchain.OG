package cmd

import "github.com/spf13/cobra"

var (
	genCmd = &cobra.Command{
		Use: "gen",
		Short: "generate config.toml files for deployment",
		Run: gen,
	}
	normal bool
	solo bool
	private bool
)

func genInit() {
	genCmd.PersistentFlags().BoolVarP(&normal,"normal", "m", true, "normal node that connect to main network")
	genCmd.PersistentFlags().BoolVarP(&solo,"solo", "s", false, "solo node that use auto client to produce sequencer")
	genCmd.PersistentFlags().BoolVarP(&private,"normal", "m", false, "private nodes that use your own boot-strap nodes")

}

func gen(cmd *cobra.Command, args []string) {


}
