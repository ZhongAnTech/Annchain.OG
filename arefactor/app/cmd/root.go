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
	"github.com/annchain/commongo/program"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "net/http/pprof"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "OG",
	Short: "OG: The next generation of DLT",
	Long:  `OG to da moon`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer program.DumpStack(true)
	_ = rootCmd.Execute()
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// folders
	rootCmd.PersistentFlags().StringP("dir-root", "r", "nodedata", "Folder for all data of one node")
	rootCmd.PersistentFlags().String("dir-log", "", "Log folder. Default to {dir.root}/log")
	rootCmd.PersistentFlags().String("dir-data", "", "Data folder. Default to {dir.root}/data")
	rootCmd.PersistentFlags().String("dir-config", "", "Config folder. Default to {dir.root}/config")
	rootCmd.PersistentFlags().String("dir-private", "", "Private folder. Default to {dir.root}/private")
	rootCmd.PersistentFlags().String("url-config", "", "URL for online config")

	rootCmd.PersistentFlags().StringP("configurl", "u", "", "URL for online config")
	// identity generation
	rootCmd.PersistentFlags().BoolP("gen-key", "g", false, "Automatically generate a private key if the privkey is missing.")

	// log
	rootCmd.PersistentFlags().BoolP("log-stdout", "", true, "Whether the log will be printed to stdout")
	rootCmd.PersistentFlags().BoolP("log-file", "", false, "Whether the log will be printed to file")
	rootCmd.PersistentFlags().BoolP("log-line-number", "n", false, "Whether the log will contain line number")
	rootCmd.PersistentFlags().StringP("log-level", "v", "trace", "Logging verbosity, possible values:[panic, fatal, error, warn, info, debug]")

	rootCmd.PersistentFlags().Bool("multifile-by-level", false, "Output separate log files according to their level")
	rootCmd.PersistentFlags().Bool("multifile-by-module", false, "Output separate log files according to their module")

	_ = viper.BindPFlag("dir.root", rootCmd.PersistentFlags().Lookup("dir-root"))
	_ = viper.BindPFlag("dir.log", rootCmd.PersistentFlags().Lookup("dir-log"))
	_ = viper.BindPFlag("dir.data", rootCmd.PersistentFlags().Lookup("dir-data"))
	_ = viper.BindPFlag("dir.config", rootCmd.PersistentFlags().Lookup("dir-config"))
	_ = viper.BindPFlag("dir.private", rootCmd.PersistentFlags().Lookup("dir-private"))
	_ = viper.BindPFlag("url.config", rootCmd.PersistentFlags().Lookup("url-config"))

	_ = viper.BindPFlag("gen.key", rootCmd.PersistentFlags().Lookup("gen-key"))

	_ = viper.BindPFlag("log.stdout", rootCmd.PersistentFlags().Lookup("log-stdout"))
	_ = viper.BindPFlag("log.file", rootCmd.PersistentFlags().Lookup("log-file"))
	_ = viper.BindPFlag("log.line_number", rootCmd.PersistentFlags().Lookup("log-line-number"))
	_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))

	_ = viper.BindPFlag("multifile_by_level", rootCmd.PersistentFlags().Lookup("multifile-by-level"))
	_ = viper.BindPFlag("multifile_by_module", rootCmd.PersistentFlags().Lookup("multifile-by-module"))

	rootCmd.PersistentFlags().Int("id", 0, "Node Id for debugging")
	_ = viper.BindPFlag("id", rootCmd.PersistentFlags().Lookup("id"))
}
