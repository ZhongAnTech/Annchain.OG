package cmd

import (
	"github.com/annchain/OG/poc/hotstuff_event"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func MakeLocalPartner(myIdIndex int, N int, F int, hub *hotstuff_event.LocalCommunicator) *hotstuff_event.Partner {
	// peer info should be loaded separately.
	// either from file, from ledger or hard coded.
	// for local testing, we hard coded. parter id is the string value starts from 0.
	peerIds := make([]string, N)
	for i := 0; i < N; i++ {
		peerIds[i] = strconv.Itoa(i)
	}

	logger := hotstuff_event.SetupOrderedLog(myIdIndex)
	ledger := &hotstuff_event.Ledger{
		Logger: logger,
	}
	ledger.InitDefault()

	safety := &hotstuff_event.Safety{
		Ledger: ledger,
		Logger: logger,
	}

	blockTree := &hotstuff_event.BlockTree{
		Ledger:    ledger,
		F:         F,
		Logger:    logger,
		MyIdIndex: myIdIndex,
	}
	blockTree.InitDefault()
	blockTree.InitGenesisOrLatest()

	proposerElection := &hotstuff_event.ProposerElection{N: N}

	paceMaker := &hotstuff_event.PaceMaker{
		PeerIds:          peerIds,
		MyIdIndex:        myIdIndex,
		CurrentRound:     1, // must be 1 which is AFTER GENESIS
		Safety:           safety,
		MessageHub:       hub,
		BlockTree:        blockTree,
		ProposerElection: proposerElection,
		Logger:           logger,
		Partner:          nil, // fill later
	}
	paceMaker.InitDefault()

	blockTree.PaceMaker = paceMaker

	partner := &hotstuff_event.Partner{
		PeerIds:          peerIds,
		MessageHub:       hub,
		Ledger:           ledger,
		MyIdIndex:        myIdIndex,
		N:                N,
		F:                F,
		PaceMaker:        paceMaker,
		Safety:           safety,
		BlockTree:        blockTree,
		ProposerElection: proposerElection,
		Logger:           logger,
	}
	partner.InitDefault()

	paceMaker.Partner = partner
	safety.Partner = partner

	return partner
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start multiple nodes locally.",
	Long:  `Start multiple nodes locally. They will communicate inside the program.`,
	Run: func(cmd *cobra.Command, args []string) {
		setupLogger()
		num := viper.GetInt("total")

		// prepare partners
		hub := &hotstuff_event.LocalCommunicator{Channels: map[string]chan *hotstuff_event.Msg{}}
		partners := make([]*hotstuff_event.Partner, num)

		for i := 0; i < num; i++ {
			hub.Channels[strconv.Itoa(i)] = make(chan *hotstuff_event.Msg, 30)
			partners[i] = MakeLocalPartner(i, num, num/3, hub)

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

func setupLogger() {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "15:04:05.000000",
	})
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().IntP("total", "t", 4, "Partners to be started")
	_ = viper.BindPFlag("total", runCmd.Flags().Lookup("total"))
}
