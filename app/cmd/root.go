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
	rootCmd.PersistentFlags().StringP("datadir", "d", "data", "Folder for ledger")
	rootCmd.PersistentFlags().StringP("configdir", "c", "config", "Folder for config")
	rootCmd.PersistentFlags().StringP("configurl", "u", "", "URL for online config")
	rootCmd.PersistentFlags().StringP("iddir", "i", "identity", "Folder for identity")
	rootCmd.PersistentFlags().StringP("logdir", "l", "log", "Folder for log")

	// identity generation
	rootCmd.PersistentFlags().BoolP("genkey", "g", false, "Automatically generate a private key if the privkey is missing.")

	// log
	rootCmd.PersistentFlags().BoolP("log_stdout", "s", false, "Whether the log will be printed to stdout")
	rootCmd.PersistentFlags().StringP("log_level", "v", "debug", "Logging verbosity, possible values:[panic, fatal, error, warn, info, debug]")
	rootCmd.PersistentFlags().BoolP("log_line_number", "n", false, "Whether the log will contain line number")
	rootCmd.PersistentFlags().BoolP("multifile_by_level", "m", false, "multifile_by_level")
	rootCmd.PersistentFlags().BoolP("multifile_by_module", "M", false, "multifile_by_module")

	_ = viper.BindPFlag("datadir", rootCmd.PersistentFlags().Lookup("datadir"))
	_ = viper.BindPFlag("configdir", rootCmd.PersistentFlags().Lookup("configdir"))
	_ = viper.BindPFlag("configurl", rootCmd.PersistentFlags().Lookup("configurl"))
	_ = viper.BindPFlag("iddir", rootCmd.PersistentFlags().Lookup("iddir"))
	_ = viper.BindPFlag("logdir", rootCmd.PersistentFlags().Lookup("logdir"))

	_ = viper.BindPFlag("genkey", rootCmd.PersistentFlags().Lookup("genkey"))

	_ = viper.BindPFlag("log.stdout", rootCmd.PersistentFlags().Lookup("log.stdout"))
	_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log.level"))
	_ = viper.BindPFlag("log.line_number", rootCmd.PersistentFlags().Lookup("log.line_number"))
	_ = viper.BindPFlag("multifile_by_level", rootCmd.PersistentFlags().Lookup("multifile_by_level"))
	_ = viper.BindPFlag("multifile_by_module", rootCmd.PersistentFlags().Lookup("multifile_by_module"))
	//_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log_level"))

	//viper.SetDefault("hub.outgoing_buffer_size", 100)
	//viper.SetDefault("hub.incoming_buffer_size", 100)
	//viper.SetDefault("hub.message_cache_expiration_seconds", 60)
	//viper.SetDefault("hub.message_cache_max_size", 30000)
	//viper.SetDefault("crypto.algorithm", "ed25519")
	//viper.SetDefault("tx_buffer.new_tx_queue_size", 1)
	//
	//viper.SetDefault("max_tx_hash", "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	//viper.SetDefault("max_mined_hash", "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	//
	//viper.SetDefault("debug.node_id", 0)
}
