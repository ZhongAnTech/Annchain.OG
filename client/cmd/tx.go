package cmd

import "github.com/spf13/cobra"

var (
	txCmd = &cobra.Command{
		Use:   "tx",
		Short: "new transaction",
		Run:   newTx,
	}
)

func newTx(cmd *cobra.Command, args []string) {

}
