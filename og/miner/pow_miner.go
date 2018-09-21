package miner

import (
	"github.com/annchain/OG/types"
	"math"
)

type PoWMiner struct {
}

func (m *PoWMiner) StartMine(tx types.Txi, targetMax types.Hash, start uint64, responseChan chan uint64) {
	// do brute force
	var i uint64
	base := tx.GetBase()
	for i = start; i <= math.MaxUint64; i++ {
		base.MineNonce = i
		//logrus.Debugf("%10d %s %s", i, tx.Hash().Hex(), targetMax.Hex())
		if tx.CalcMinedHash().Cmp(targetMax) < 0 {
			//logrus.Debugf("Hash found: %s with %d", tx.CalcMinedHash().Hex(), i)
			responseChan <- i
			return
		}
	}

}
func (m *PoWMiner) Stop() {
}
