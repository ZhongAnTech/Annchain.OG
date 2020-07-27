package cmd

import (
	"github.com/annchain/OG/arefactor/core"
	"github.com/annchain/commongo/mylog"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

//func setupSoloEngine(myNode *node.Node2) {
//	myNode.Setup()
//}

// runCmd represents the run command
var soloCmd = &cobra.Command{
	Use:   "solo",
	Short: "Start a solo node",
	Long:  `Start a solo node. No consensus will be involved.`,
	Run: func(cmd *cobra.Command, args []string) {

		logrus.WithField("pid", os.Getpid()).Info("OG Solo Starting")
		folderConfigs := ensureFolders()
		readConfig(folderConfigs.Config)
		mylog.InitLogger(logrus.StandardLogger(), mylog.LogConfig{
			MaxSize:    10,
			MaxBackups: 100,
			MaxAgeDays: 90,
			Compress:   true,
			LogDir:     folderConfigs.Log,
			OutputFile: "ogsolo",
		})

		node := &core.SoloNode{}
		node.InitDefault()
		node.Setup()
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

func init() {
	rootCmd.AddCommand(soloCmd)
}
