package annsensus

import (
	"crypto/rand"
	"github.com/annchain/OG/poc/vrf"
	"github.com/annchain/OG/types"

)
//go:generate msgp

func (as *AnnSensus)GenerateVrf(campaign *types.Campaign) (ok bool){
	sk,err := vrf.GenerateKey(rand.Reader)
	if err!=nil {
		panic(err)
	}
	pk, _ := sk.Public()
	data := as.GetProofData()
	 Vrf  := sk.Compute(data)
	VRFFromProof, proof := sk.Prove(data)
	if Vrf[0] <0xf0 {
		return false
		campaign.Vrf = VRFFromProof  ///todo ???
	}
	campaign.Vrf = Vrf
	campaign.PkVrf = pk
	campaign.Proof = proof
	campaign.Data = data
	return true
}

type VrfData struct {
	SeqHash types.Hash
	Height  uint64
	TxNum   int
}

func (as *AnnSensus)GetProofData() []byte{
	var vd VrfData
	sq := as.Idag.LatestSequencer()
	txs := as.Idag.GetTxsByNumber(sq.Height)
	vd.SeqHash = sq.Hash
	vd.Height = sq.Height
	vd.TxNum = len(txs)
	data ,_ := vd.MarshalMsg(nil)
	return data
}
