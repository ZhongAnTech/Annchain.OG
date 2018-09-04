package miner

import (
	"github.com/annchain/OG/types"
	"math"
	"github.com/sirupsen/logrus"
)

type PoWMiner struct {
	tx   types.Txi
	quit bool
}

func (m *PoWMiner) StartMine(tx types.Txi, targetMax types.Hash, responseChan chan uint64) {
	m.quit = false
	defer close(responseChan)
	// do brute force
	var i uint64
	for i = 0; i <= math.MaxUint64; i++ {
		tx.SetMineNonce(i)
		//logrus.Debugf("%10d %s %s", i, tx.Hash().Hex(), targetMax.Hex())
		if tx.MinedHash().Cmp(targetMax) < 0 {
			logrus.Debugf("Hash found: %s with %d", tx.MinedHash().Hex(), i)
			responseChan <- i
			return
		}
	}

}
func (m *PoWMiner) Stop() {
	m.quit = true
}
