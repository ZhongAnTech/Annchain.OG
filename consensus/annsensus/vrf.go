package annsensus

import (
	"crypto/rand"
	"fmt"
	"github.com/annchain/OG/poc/vrf"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

//go:generate msgp

func (as *AnnSensus) GenerateVrf() ( *types.VrfInfo ) {
	sk, err := vrf.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	pk, _ := sk.Public()
	_, data := as.GetProofData(0)
	Vrf := sk.Compute(data)
	ok := as.VrfCondition(Vrf)
	if !ok {
		return nil
	}
	VRFFromProof, proof := sk.Prove(data)
	_ = VRFFromProof ///todo ???
	var VrfInfo  types.VrfInfo
	VrfInfo.Vrf = Vrf
	VrfInfo.PublicKey = pk
	VrfInfo.Proof = proof
	VrfInfo.Message = data
	return &VrfInfo
}

type VrfData struct {
	SeqHash types.Hash
	Height  uint64
	TxNum   int
}

//GetProofData get data
func (as *AnnSensus) GetProofData(height uint64) (*VrfData,  []byte) {
	var sq *types.Sequencer
	if height == 0 {
		sq = as.Idag.LatestSequencer()
	} else {
		sq = as.Idag.GetSequencerByHeight(height)
	}
	if sq ==nil {
		logrus.WithField("height ",height).Warn("we don't have this sequencer yet")
		return nil ,nil
	}
	txs := as.Idag.GetTxsByNumber(sq.Height)
	vd := &VrfData{}
	vd.SeqHash = sq.Hash
	vd.Height = sq.Height
	vd.TxNum = len(txs)
	data, _ := vd.MarshalMsg(nil)
	return vd, data
}

func (as *AnnSensus) VrfCondition(Vrf []byte) bool {
	if len(Vrf) != vrf.Size {
		logrus.WithField("len", len(Vrf)).Warn("vrf length error")
		return false
	}
	//for test  return true , we need more node
	//todo remove this later
	return  true
	if Vrf[0] < 0x80 {
		return false
	}
	return true

}

func (as *AnnSensus) VerifyVrfData(data []byte) error {
	var vd VrfData
	_, err := vd.UnmarshalMsg(data)
	if err != nil {
		return err
	}
	//todo need more condition

	shouldVD, _ := as.GetProofData(vd.Height)

	if shouldVD ==nil || *shouldVD != vd {
		logrus.WithField("vrf data ", vd).WithField("want ", shouldVD).Debug("vrf data mismatch")
		return fmt.Errorf("vfr data mismatch")
	}
	return nil
}

func (as *AnnSensus) VrfVerify(Vrf []byte, pk []byte, data []byte, proof []byte) (err error) {
	if !as.VrfCondition(Vrf) {
		return fmt.Errorf("not your turn ; vrf condition mismatch")
	}
	if len(pk) != vrf.PublicKeySize {
		return fmt.Errorf("publik ley size error %d", len(pk))
	}
	err = as.VerifyVrfData(data)
	if err != nil {
		return err
	}
	pubKey := vrf.PublicKey(pk)
	if !pubKey.Verify(data, Vrf, proof) {
		err = fmt.Errorf("vrf verifr error")
		return err
	}
	return nil
}
