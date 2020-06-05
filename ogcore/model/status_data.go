package model

import (
	"fmt"
	"github.com/annchain/OG/arefactor/og/types"
)

// OgStatusData is the network packet for the status message.
type OgStatusData struct {
	ProtocolVersion uint32
	NetworkId       uint64
	CurrentBlock    types.Hash
	GenesisBlock    types.Hash
	CurrentHeight   uint64
}

func (s OgStatusData) BasicMatch(o2 OgStatusData) bool {
	return s.NetworkId == o2.NetworkId &&
		s.GenesisBlock == o2.GenesisBlock
}

func (s OgStatusData) IsCompatible(o2 OgStatusData) bool {
	return s.BasicMatch(o2) && s.ProtocolVersion >= o2.ProtocolVersion
}

func (s OgStatusData) IsHeightNotLowerThan(o2 OgStatusData) bool {
	return s.CurrentHeight >= o2.CurrentHeight
}

func (s *OgStatusData) String() string {
	return fmt.Sprintf("ProtocolVersion  %d   NetworkId %d  CurrentBlock %s  GenesisBlock %s  CurrentHeight %d",
		s.ProtocolVersion, s.NetworkId, s.CurrentBlock, s.GenesisBlock, s.CurrentHeight)
}
