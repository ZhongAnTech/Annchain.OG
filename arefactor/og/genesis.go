package og

import (
	"github.com/annchain/OG/arefactor/og_interface"
)

type Genesis struct {
	RootSequencerHash og_interface.Hash
	FirstCommittee    *Committee
}
