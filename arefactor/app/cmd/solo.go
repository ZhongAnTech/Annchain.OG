package cmd

import (
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
	Use:   "run",
	Short: "Start a solo node",
	Long:  `Start a solo node`,
	Run: func(cmd *cobra.Command, args []string) {
		// init logs and other facilities before the node starts
		readConfig()
		initLogger()
		startPerformanceMonitor()

		pid := os.Getpid()
		log.WithField("pid", pid).Info("Node Starting")

		//ogEngine := &engine.Engine{}
		//ogEngine.InitDefault()

		writeConfig()
		//ogEngine.Start()

		// prevent sudden stop. Do your clean up here
		var gracefulStop = make(chan os.Signal)

		signal.Notify(gracefulStop, syscall.SIGTERM)
		signal.Notify(gracefulStop, syscall.SIGINT)

		func() {
			sig := <-gracefulStop
			log.Warnf("caught sig: %+v", sig)
			log.Warn("Exiting... Please do no kill me")
			//ogEngine.Stop()
			os.Exit(0)
		}()

	},
}
