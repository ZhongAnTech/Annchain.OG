package archive

import "github.com/annchain/OG/og/types"

//go:generate msgp

//msgp:tuple RawArchive
type RawArchive struct {
	Archive
}

func (a *RawArchive) Txi() types.Txi {
	return &a.Archive
}

//msgp:tuple RawArchive
type RawArchives []*RawArchive