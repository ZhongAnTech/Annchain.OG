package cmd

import (
	"github.com/annchain/OG/poc/hotstuff_event"
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

func MakePartner(myId int, N int, F int, hub *hotstuff_event.Hub) *hotstuff_event.Partner {
	ledger := &hotstuff_event.Ledger{}
	safety := &hotstuff_event.Safety{
		Ledger: ledger,
	}

	blockTree := &hotstuff_event.BlockTree{
		Ledger: ledger,
		F:      F,
	}
	proposerElection := &hotstuff_event.ProposerElection{N: N}

	paceMaker := &hotstuff_event.PaceMaker{
		MyId:             myId,
		CurrentRound:     0,
		Safety:           safety,
		MessageHub:       hub,
		BlockTree:        blockTree,
		ProposerElection: proposerElection,
		Partner:          nil,
	}

	blockTree.PaceMaker = paceMaker

	partner := &hotstuff_event.Partner{
		MessageHub:       hub,
		Ledger:           ledger,
		MyId:             myId,
		N:                N,
		F:                F,
		PaceMaker:        nil,
		Safety:           safety,
		BlockTree:        blockTree,
		ProposerElection: proposerElection,
	}
	paceMaker.Partner = partner

	partner.InitDefault()
	return partner
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start a full node",
	Long:  `Start a full node`,
	Run: func(cmd *cobra.Command, args []string) {
		num := viper.GetInt("number")

		// prepare partners
		hub := &hotstuff_event.Hub{Channels: map[int]chan *hotstuff_event.Msg{}}
		partners := make([]*hotstuff_event.Partner, num)

		for i := 0; i < num; i++ {
			hub.Channels[i] = make(chan *hotstuff_event.Msg, 30)
			partners[i] = MakePartner(i, num, num/3, hub)

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
