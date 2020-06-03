package cmd

import (
	"github.com/annchain/OG/arefactor/core"
	log "github.com/sirupsen/logrus"
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
		ensureFolder()
		initLogger()
		// init logs and other facilities before the node starts
		readConfig()
		startPerformanceMonitor()
		pid := os.Getpid()
		writeConfig()

		log.WithField("pid", pid).Info("Node Starting")

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
