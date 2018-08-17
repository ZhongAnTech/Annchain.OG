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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sirupsen/logrus"
	"path"
	"io"
	"path/filepath"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "OG",
	Short: "OG: The next generation of DLT",
	Long: `OG to da moon`,
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringP("datadir", "d", "data", fmt.Sprintf("Runtime directory for storage and configurations"))
	rootCmd.PersistentFlags().StringP("config", "c", "config.toml", "Path for configuration file")
	rootCmd.PersistentFlags().StringP("log_dir", "l", "", "Path for configuration file. Not enabled by default")
	//rootCmd.PersistentFlags().BoolP("log_stdout", "ls", true, "Whether the log will be printed to stdout")
	rootCmd.PersistentFlags().StringP("log_verbosity", "v", "debug", "Logging verbosity, possible values:[panic, fatal, error, warn, info, debug]")

	viper.BindPFlag("datadir", rootCmd.PersistentFlags().Lookup("datadir"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("log_dir", rootCmd.PersistentFlags().Lookup("log_dir"))
	//viper.BindPFlag("log_stdout", rootCmd.PersistentFlags().Lookup("log_stdout"))
	viper.BindPFlag("log_verbosity", rootCmd.PersistentFlags().Lookup("log_verbosity"))
}

func panicIfError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// initLogger uses viper to get the log path and level. It should be called by all other commands
func initLogger() {
	logdir := viper.GetString("log_dir")
	stdout := viper.GetBool("log_stdout")

	var writer io.Writer

	if logdir != "" {
		folderPath, err := filepath.Abs(logdir)
		panicIfError(err, fmt.Sprintf("Error on parsing log path: %s", logdir))

		abspath, err := filepath.Abs(path.Join(logdir, "run.log"))
		panicIfError(err, fmt.Sprintf("Error on parsing log file path: %s", logdir))

		err = os.MkdirAll(folderPath, os.ModePerm)
		panicIfError(err, fmt.Sprintf("Error on creating log dir: %s", folderPath))

		logFile, err := os.OpenFile(abspath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		panicIfError(err, fmt.Sprintf("Error on creating log file: %s", abspath))

		if stdout {
			fmt.Println("Will be logged to stdout and ", abspath)
			writer = io.MultiWriter(os.Stdout, logFile)
		} else {
			fmt.Println("Will be logged to ", abspath)
			writer = logFile
		}
	} else {
		// stdout only
		fmt.Println("Will be logged to stdout")
		writer = os.Stdout
	}

	logrus.SetOutput(writer)

	// Only log the warning severity or above.
	switch viper.GetString("log_verbosity") {
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	default:
		fmt.Println("Unknown level", viper.GetString("log_verbosity"), "Set to INFO")
		logrus.SetLevel(logrus.InfoLevel)
	}

	Formatter := new(logrus.TextFormatter)
	Formatter.ForceColors = true
	//Formatter.DisableColors = true
	Formatter.TimestampFormat = "2006-01-02 15:04:05.000000"
	Formatter.FullTimestamp = true

	logrus.SetFormatter(Formatter)

	// redirect standard log to logrus
	//log.SetOutput(logrus.StandardLogger().Writer())
	//log.Println("Standard logger. Am I here?")

	logrus.Debug("Logger initialized.")
}
