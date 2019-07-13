// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Host string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "OGtool",
	Short: "OGtool: The next generation of DLT",
	Long:  `OG to da moon`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&Host, "server", "s", "http://127.0.0.1:8000", fmt.Sprintf("serverurl,default: http://127.0.0.1:8000"))
	InfoCmd.AddCommand(netInfoCmd)
	rootCmd.AddCommand(InfoCmd)
	txInit()
	rootCmd.AddCommand(txCmd)
	accountInit()
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(tokenCmd)
	tokenInit()
	rootCmd.AddCommand(tpsCmd)
	tpsInit()

}

func panicIfError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

