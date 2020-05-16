package cmd

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/annchain/OG/poc/hotstuff_event"
	"github.com/latifrons/soccerdash"
	core "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func MakeStandalonePartner(myIdIndex int, N int, F int, hub hotstuff_event.Hub, peerIds []string) *hotstuff_event.Partner {
	//logger := hotstuff_event.SetupOrderedLog(myIdIndex)
	logger := logrus.StandardLogger()
	reporter := &soccerdash.Reporter{
		Name:          fmt.Sprintf("Peer-%d", myIdIndex),
		TargetAddress: "127.0.0.1:2088",
	}
	ledger := &hotstuff_event.Ledger{
		Logger: logger,
	}
	ledger.InitDefault()

	safety := &hotstuff_event.Safety{
		Ledger: ledger,
		Logger: logger,
	}

	blockTree := &hotstuff_event.BlockTree{
		Logger:    logger,
		Ledger:    ledger,
		F:         F,
		MyIdIndex: myIdIndex,
		Report:    reporter,
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
		Report:           reporter,
	}
	partner.InitDefault()

	paceMaker.Partner = partner
	safety.Partner = partner

	return partner
}

// runCmd represents the run command
var standaloneCmd = &cobra.Command{
	Use:   "std",
	Short: "Start a standalone node",
	Long:  `Start a standalone node and communicate with other standalone nodes`,
	Run: func(cmd *cobra.Command, args []string) {
		//setupLogger()
		peers := readList(viper.GetString("list"))
		total := len(peers)

		priv, id := loadPrivateKey()

		p2p := &hotstuff_event.PhysicalCommunicator{
			Port:       viper.GetInt("port"),
			PrivateKey: priv,
		}
		p2p.InitDefault()
		// init me before init peers
		p2p.Start()

		peerIds := make([]string, len(peers))
		// preconnect peers
		for i, peer := range peers {
			peerId := p2p.SuggestConnection(peer)
			peerIds[i] = peerId
		}
		hotstuff_event.SetPeers(peerIds)

		// get my index in the peer list
		myIdIndex, err := getMyIdIndexInList(id, peerIds)
		if err != nil {
			panic(err)
		}

		logrus.Info("setup logger")
		hotstuff_event.SetupOrderedLog(myIdIndex)

		hub := &hotstuff_event.LogicalCommunicator{
			PhysicalCommunicator: p2p,
			MyId:                 id,
		}

		hub.InitDefault()
		hub.Start()

		// Debugging broadcasting and printing
		//go func() {
		//	messageChannel, _ := hub.GetChannel(hub.MyId)
		//	for {
		//		v := <-messageChannel
		//		fmt.Println("I received " + v.Content.String())
		//	}
		//}()
		//go func() {
		//	for {
		//		// now broadcast constantly
		//		hub.Broadcast(&hotstuff_event.Msg{
		//			Typev:    hotstuff_event.String,
		//			Sig:      hotstuff_event.Signature{},
		//			SenderId: hub.MyId,
		//			Content: &hotstuff_event.ContentString{
		//				Content: fmt.Sprintf("MSG %s->%s", hub.MyId, time.Now().String())},
		//		}, "")
		//		time.Sleep(time.Second * 2)
		//	}
		//}()
		go func() {
			for {
				logrus.WithField("number", runtime.NumGoroutine()).Info("go rountine numbers")
				time.Sleep(time.Second * 10)
			}
		}()

		partner := MakeStandalonePartner(myIdIndex, total, total/3, hub, peerIds)
		go partner.Start()

		//go func() {
		//	for {
		//		// dump contstantly
		//		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		//		pprof.Lookup("block").WriteTo(os.Stdout, 1)
		//
		//		time.Sleep(time.Second * 30)
		//	}
		//}()

		// prevent sudden stop. Do your clean up here
		var gracefulStop = make(chan os.Signal)

		signal.Notify(gracefulStop, syscall.SIGTERM)
		signal.Notify(gracefulStop, syscall.SIGINT)

		func() {
			sig := <-gracefulStop
			log.Warnf("caught sig: %+v", sig)
			log.Warn("Exiting... Please do no kill me")
			//for _, partner := range partners {
			//	partner.Stop()
			//}
			os.Exit(0)
		}()

	},
}

func getMyIdIndexInList(id string, peers []string) (myIdIndex int, err error) {
	// just for simple
	for i, peer := range peers {
		if peer == id {
			return i, nil
		}
	}
	logrus.WithField("peer", id).WithField("list", peers).Warn("peer not found in list")

	return -1, errors.New("peer not found in list")
}

func readList(filename string) (peers []string) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		peers = append(peers, line)

	}
	return

}

func loadPrivateKey() (core.PrivKey, string) {
	// read key file
	keyFile := viper.GetString("file")
	bytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		panic(err)
	}

	pi := &hotstuff_event.PrivateInfo{}
	err = json.Unmarshal(bytes, pi)
	if err != nil {
		panic(err)
	}

	privb, err := hex.DecodeString(pi.PrivateKey)
	if err != nil {
		panic(err)
	}

	priv, err := core.UnmarshalPrivateKey(privb)
	if err != nil {
		panic(err)
	}
	return priv, pi.Id
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
