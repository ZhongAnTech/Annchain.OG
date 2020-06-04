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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "net/http/pprof"
	"os"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "OG",
	Short: "OG: The next generation of DLT",
	Long:  `OG to da moon`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer DumpStack()
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Fatalf("Fatal error occurred. Program will exit")

		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// folders
	rootCmd.PersistentFlags().StringP("rootdir", "r", "nodedata", "Folder for all data of one node")
	rootCmd.PersistentFlags().StringP("configurl", "u", "", "URL for online config")
	// identity generation
	rootCmd.PersistentFlags().BoolP("genkey", "g", false, "Automatically generate a private key if the privkey is missing.")

	// log
	rootCmd.PersistentFlags().BoolP("log-stdout", "", true, "Whether the log will be printed to stdout")
	rootCmd.PersistentFlags().BoolP("log-file", "", false, "Whether the log will be printed to file")
	rootCmd.PersistentFlags().StringP("log-level", "v", "trace", "Logging verbosity, possible values:[panic, fatal, error, warn, info, debug]")
	rootCmd.PersistentFlags().BoolP("log-line-number", "n", false, "Whether the log will contain line number")

	rootCmd.PersistentFlags().BoolP("multifile_by_level", "m", false, "multifile_by_level")
	rootCmd.PersistentFlags().BoolP("multifile_by_module", "M", false, "multifile_by_module")

	_ = viper.BindPFlag("rootdir", rootCmd.PersistentFlags().Lookup("rootdir"))
	_ = viper.BindPFlag("configurl", rootCmd.PersistentFlags().Lookup("configurl"))
	_ = viper.BindPFlag("genkey", rootCmd.PersistentFlags().Lookup("genkey"))

	_ = viper.BindPFlag("log-stdout", rootCmd.PersistentFlags().Lookup("log-stdout"))
	_ = viper.BindPFlag("log-file", rootCmd.PersistentFlags().Lookup("log-file"))
	_ = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("log-line-number", rootCmd.PersistentFlags().Lookup("log-line-number"))

	_ = viper.BindPFlag("multifile_by_level", rootCmd.PersistentFlags().Lookup("multifile_by_level"))
	_ = viper.BindPFlag("multifile_by_module", rootCmd.PersistentFlags().Lookup("multifile_by_module"))

	rootCmd.PersistentFlags().Int("id", 0, "Node Id for debugging")
	_ = viper.BindPFlag("id", rootCmd.PersistentFlags().Lookup("id"))
}
