package archive

import (
	"github.com/annchain/OG/og/protocol_message"
)

//go:generate msgp

//msgp:tuple RawArchive
type RawArchive struct {
	Archive
}

func (a *RawArchive) Txi() protocol_message.Txi {
	return &a.Archive
}

//msgp:tuple RawArchive
type RawArchives []*RawArchive