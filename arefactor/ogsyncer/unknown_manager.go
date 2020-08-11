package ogsyncer

import (
	"container/list"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"sync"
)

type DefaultUnknownManager struct {
	unknowns   list.List
	unknownMap map[ogsyncer_interface.Unknown]ogsyncer_interface.SourceHint
	mu         sync.RWMutex
}

func (d *DefaultUnknownManager) Enqueue(task ogsyncer_interface.Unknown, hint ogsyncer_interface.SourceHint) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.unknownMap[task]; !ok {
		d.unknowns.PushBack(task)
		d.unknownMap[task] = hint
	}
}

func (d *DefaultUnknownManager) InitDefault() {
	d.unknownMap = make(map[ogsyncer_interface.Unknown]ogsyncer_interface.SourceHint)
}
