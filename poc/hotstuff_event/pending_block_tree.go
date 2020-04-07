package hotstuff_event

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
)

// Stores state merkle tree.
type PendingBlockTree struct {
	MyId           int
	Logger         *logrus.Logger
	cache          map[string]*Block
	childRelations map[string][]string
}

func (t *PendingBlockTree) String() string {
	return fmt.Sprintf("[PBT: cache %d relation %d]", len(t.cache), len(t.childRelations))
}

func (t *PendingBlockTree) InitDefault() {
	t.cache = make(map[string]*Block)
	t.childRelations = make(map[string][]string)
}

func (t *PendingBlockTree) Add(p *Block) {
	t.cache[p.Id] = p

	vs, ok := t.childRelations[p.ParentQC.VoteData.Id]
	if !ok {
		vs = []string{}
	}
	vs = append(vs, p.Id)
	t.childRelations[p.ParentQC.VoteData.Id] = vs
}

func (t *PendingBlockTree) Commit(id string) {
	//fmt.Printf("[%d] Block %s\n", t.MyId, id)
	t.Logger.WithField("id", id).Debug("block commit")
}

func (l *PendingBlockTree) GetBlock(id string) (block *Block, err error) {
	if v, ok := l.cache[id]; ok {
		block = v
		return
	}
	err = errors.New("block not found")
	return
}
