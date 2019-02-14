package annsensus

import (
	"github.com/annchain/OG/types"
)

type AnnSensus struct {
}

// Commit takes a list of campaigns as input and record these
// camps' information It checks if the number of camps reaches
// the threshold. If so, start term changing flow.
func (as *AnnSensus) Commit(camps []*types.Campaign) {

}

// func (as *AnnSensus)
