package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use: "OG tool for deployment",
	Short: "deploy helps you to create boot node address and init genesis consensus public keys",
}

func init() {
	onodeInit()
	rootCmd.AddCommand(onodeCmd)

	genInit()
	genCmd.AddCommand(genCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
