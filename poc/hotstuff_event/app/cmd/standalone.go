package cmd

import (
	"bufio"
	"github.com/annchain/OG/poc/hotstuff_event"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func MakeStandalonePartner(myId int, N int, F int, hub hotstuff_event.Hub, peerIds []string) *hotstuff_event.Partner {
	logger := hotstuff_event.SetupOrderedLog(myId)
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
		MyIdIndex: myId,
	}
	blockTree.InitDefault()
	blockTree.InitGenesisOrLatest()

	proposerElection := &hotstuff_event.ProposerElection{N: N}

	paceMaker := &hotstuff_event.PaceMaker{
		PeerIds:          peerIds,
		MyIdIndex:        myId,
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
		MyIdIndex:        myId,
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
var standaloneCmd = &cobra.Command{
	Use:   "standalone",
	Short: "Start a standalone node",
	Long:  `Start a standalone node and communicate with other standalone nodes`,
	Run: func(cmd *cobra.Command, args []string) {
		setupLogger()

		peers := readList(viper.GetString("list"))
		total := len(peers)

		hub := &hotstuff_event.LogicalCommunicator{
			Port: viper.GetInt("port"),
		}
		hub.InitDefault()
		// init me before init peers
		go hub.Listen()
		//go hub.Start()

		go hub.InitPeers(peers)

		logrus.Info("waiting for connection...")
		partners := make([]*hotstuff_event.Partner, total)

		//partner := MakeStandalonePartner(viper.GetInt("mei"), total, total/3, hub)
		//go partner.Start()

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

func readList(filename string) (peers []string) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		s := strings.TrimSpace(line)
		peers = append(peers, s)
	}
	return

}

func init() {
	rootCmd.AddCommand(standaloneCmd)
	standaloneCmd.Flags().StringP("list", "l", "peers.lst", "Partners to be started in file list")
	_ = viper.BindPFlag("list", standaloneCmd.Flags().Lookup("list"))

	standaloneCmd.Flags().StringP("file", "f", "id.key", "my key file")
	_ = viper.BindPFlag("file", standaloneCmd.Flags().Lookup("file"))

	standaloneCmd.Flags().IntP("port", "p", 3301, "Local IO port")
	_ = viper.BindPFlag("port", standaloneCmd.Flags().Lookup("port"))
}
