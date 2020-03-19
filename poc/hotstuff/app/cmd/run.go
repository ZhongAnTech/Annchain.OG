package cmd

import (
	"github.com/annchain/OG/poc/hotstuff"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"syscall"
)

func pickLeader(n int, viewNumber int) int {
	return viewNumber % n
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start a full node",
	Long:  `Start a full node`,
	Run: func(cmd *cobra.Command, args []string) {
		num := viper.GetInt("number")

		// prepare partners
		hub := &hotstuff.Hub{Channels: map[int]chan *hotstuff.Msg{}}
		partners := make([]*hotstuff.Partner, num)

		for i := 0; i < num; i++ {
			hub.Channels[i] = make(chan *hotstuff.Msg, 30)
			partners[i] = &hotstuff.Partner{
				MessageHub:  hub,
				NodeCache:   make(map[string]*hotstuff.Node),
				LockedQC:    nil,
				PrepareQC:   nil,
				PreCommitQC: nil,
				CommitQC:    nil,
				Id:          i,
				N:           num,
				F:           num / 3,
				LeaderFunc:  pickLeader,
			}
			partners[i].InitDefault()
		}
		for i := 0; i < num; i++ {
			go partners[i].Start()
		}

		// prevent sudden stop. Do your clean up here
		var gracefulStop = make(chan os.Signal)

		signal.Notify(gracefulStop, syscall.SIGTERM)
		signal.Notify(gracefulStop, syscall.SIGINT)

		func() {
			sig := <-gracefulStop
			log.Warnf("caught sig: %+v", sig)
			log.Warn("Exiting... Please do no kill me")
			for _, partner := range partners {
				partner.Stop()
			}
			os.Exit(0)
		}()

	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().IntP("number", "n", 4, "Partners to be started")
	_ = viper.BindPFlag("number", runCmd.Flags().Lookup("number"))
}
