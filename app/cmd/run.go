// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
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

	"github.com/annchain/OG/node"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start a full node",
	Long:  `Start a full node`,
	Run: func(cmd *cobra.Command, args []string) {
		// init logs and other facilities before the node starts
		readConfig()
		initLogger()
		startPerformanceMonitor()
		//fmt.Println(viper.GetString("title"))
		//fmt.Println(viper.GetStringSlice("database.ports"))
		//fmt.Println(viper.Get("clients.data"))
		log.Info("Node Starting")
		node := node.NewNode()
		node.Start()

		// prevent sudden stop. Do your clean up here

		var gracefulStop = make(chan os.Signal)

		signal.Notify(gracefulStop, syscall.SIGTERM)
		signal.Notify(gracefulStop, syscall.SIGINT)

		func() {
			sig := <-gracefulStop
			log.Warnf("caught sig: %+v", sig)
			log.Warn("Exiting... Please do no kill me")
			node.Stop()
			os.Exit(0)
		}()

	},
}

func readConfig() {
	absPath, err := filepath.Abs(viper.GetString("config"))
	panicIfError(err, fmt.Sprintf("Error on parsing config file path: %s", absPath))

	file, err := os.Open(absPath)
	panicIfError(err, fmt.Sprintf("Error on opening config file: %s", absPath))
	defer file.Close()

	viper.SetConfigType("toml")
	err = viper.MergeConfig(file)
	panicIfError(err, fmt.Sprintf("Error on reading config file: %s", absPath))
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
