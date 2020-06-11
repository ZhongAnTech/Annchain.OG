package og

import (
	"github.com/annchain/OG/arefactor/og_interface"
)

type Ledger interface {
	CurrentHeight() int64
	CurrentCommittee() *Committee
}

type DefaultLedger struct {
	height  int64
	genesis *Genesis
}

// StaticSetup supposely will load ledger from disk.
func (d *DefaultLedger) StaticSetup() {
	d.height = 1
	rootHash := &og_interface.Hash32{}
	rootHash.FromHexNoError("0x00")

	d.genesis = &Genesis{
		RootSequencerHash: rootHash,
		FirstCommittee: &Committee{
			Peers: []*PeerMember{
				{
					PeerId:    "16Uiu2HAmSPLF68qLtob31r3hYga7Qs84TPas9wNkanKQtKezjRne",
					PublicKey: nil,
				},
				{
					PeerId:    "16Uiu2HAmKK8gG13RdesZenARuPmsYnXQH5iLad85eMoepRQS6Pi9",
					PublicKey: nil,
				},
				{
					PeerId:    "16Uiu2HAmGWF1gqXsJ5oZzy213bQRh1aWqSRdiowRecNNH2DEqkYL",
					PublicKey: nil,
				},
				{
					PeerId:    "16Uiu2HAmTb7wFyTqjWTMrjdUNZXRUdXh4byQi6nBNGGnYe8vhdG3",
					PublicKey: nil,
				},
			},
			Version: 1,
		},
	}
}

func (d *DefaultLedger) CurrentHeight() int64 {
	return d.height
}

func (d *DefaultLedger) CurrentCommittee() *Committee {
	// currently no general election. always return committee in genesis
	return d.genesis.FirstCommittee
}
