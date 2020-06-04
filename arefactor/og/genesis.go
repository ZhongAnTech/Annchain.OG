package og

import "github.com/annchain/OG/arefactor/og/types"

type Genesis struct {
	RootSequencerHash types.Hash
	FirstCommittee    *Committee
}
