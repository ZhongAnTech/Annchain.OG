package archive

import "github.com/annchain/OG/og/protocol/ogmessage"

//go:generate msgp

//msgp:tuple RawArchive
type RawArchive struct {
	Archive
}

func (a *RawArchive) Txi() ogmessage.Txi {
	return &a.Archive
}

//msgp:tuple RawArchive
type RawArchives []*RawArchive