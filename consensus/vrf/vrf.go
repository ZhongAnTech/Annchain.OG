// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package vrf

import (
	"crypto/rand"
	"fmt"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/poc/vrf"
	"github.com/sirupsen/logrus"
)

//go:generate msgp

type Vrf struct {
}

func (as *Vrf) GenerateVrf() *VrfInfo {
	sk, err := vrf.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	pk, _ := sk.Public()
	//TODO: recover this. currently just comment out for compiler
	//_, data := as.GetProofData(as.Idag.LatestSequencer().Height)
	var data []byte

	Vrf := sk.Compute(data)
	ok := as.VrfCondition(Vrf)
	if !ok {
		return nil
	}
	VRFFromProof, proof := sk.Prove(data)
	_ = VRFFromProof ///todo ???
	var VrfInfo VrfInfo
	VrfInfo.Vrf = Vrf
	VrfInfo.PublicKey = pk
	VrfInfo.Proof = proof
	VrfInfo.Message = data
	return &VrfInfo
}

//msgp:tuple VrfData
type VrfData struct {
	SeqHash types.Hash
	Height  uint64
	TxNum   int
}

//GetProofData get data
//func (as *Vrf) GetProofData(height uint64) (*VrfData, []byte) {
//	var sq *Sequencer
//	if height == 0 {
//		sq = as.Idag.LatestSequencer()
//	} else {
//		sq = as.Idag.GetSequencerByHeight(height)
//	}
//	if sq == nil {
//		logrus.WithField("height ", height).Warn("we don't have this sequencer yet")
//		return nil, nil
//	}
//	txs := as.Idag.GetTxisByHeight(sq.Height)
//	vd := &VrfData{}
//	vd.SeqHash = sq.Hash
//	vd.Height = sq.Height
//	vd.TxNum = len(txs)
//	data, _ := vd.MarshalMsg(nil)
//	return vd, data
//}

func (as *Vrf) VrfCondition(Vrf []byte) bool {
	if len(Vrf) != vrf.Size {
		logrus.WithField("len", len(Vrf)).Warn("vrf length error")
		return false
	}
	//for test  return true , we need more node
	//todo remove this later
	return true
	if Vrf[0] < 0x80 {
		return false
	}
	return true

}

func (as *Vrf) VerifyVrfData(data []byte) error {
	var vd VrfData
	_, err := vd.UnmarshalMsg(data)
	if err != nil {
		return err
	}
	//todo need more condition

	//shouldVD, _ := as.GetProofData(vd.Height)
	//
	//if shouldVD == nil || *shouldVD != vd {
	//	logrus.WithField("vrf data ", vd).WithField("want ", shouldVD).Debug("vrf data mismatch")
	//	return fmt.Errorf("vfr data mismatch")
	//}
	return nil
}

func (as *Vrf) VrfVerify(Vrf []byte, pk []byte, data []byte, proof []byte) (err error) {
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
