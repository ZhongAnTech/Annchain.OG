//// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
////
//// Licensed under the Apache License, Version 2.0 (the "License");
//// you may not use this file except in compliance with the License.
//// You may obtain a copy of the License at
////
////     http://www.apache.org/licenses/LICENSE-2.0
////
//// Unless required by applicable law or agreed to in writing, software
//// distributed under the License is distributed on an "AS IS" BASIS,
//// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//// See the License for the specific language governing permissions and
//// limitations under the License.
package archive
//
//import (
//	"encoding/hex"
//	"encoding/json"
//	"fmt"
//	"github.com/annchain/OG/common"
//	"github.com/annchain/OG/common/hexutil"
//	dkg2 "github.com/annchain/OG/consensus/dkg"
//	"github.com/annchain/OG/types/tx_types"
//	"github.com/annchain/kyber/v3"
//
//	"github.com/annchain/kyber/v3/group/mod"
//	"github.com/annchain/kyber/v3/pairing/bn256"
//	"github.com/annchain/kyber/v3/share"
//	"github.com/annchain/kyber/v3/share/dkg/pedersen"
//	"io/ioutil"
//	"os"
//	"path/filepath"
//)
//
//type DkgConfig struct {
//	DKgSecretKey      hexutil.Bytes `json:"d_kg_secret_key"`
//	DKgJointPublicKey hexutil.Bytes `json:"d_kg_joint_public_key"`
//	jointPubKey       kyber.Point
//	secretKey         kyber.Scalar
//	keyShare          *dkg.DistKeyShare
//	CommitLen         []int         `json:"commit_len"`
//	PolyLen           []int         `json:"poly_len"`
//	CommitsData       hexutil.Bytes `json:"commits_data"`
//	PrivPolyData      hexutil.Bytes `json:"priv_poly_data"`
//	ShareData         hexutil.Bytes `json:"share_data"`
//	PartnerId         uint32        `json:"partner_id"`
//	SigSets           map[common.Address]*tx_types.SigSet
//}
//
//func (c DkgConfig) String() string {
//	if c.keyShare != nil {
//		return fmt.Sprintf("sk %s\n pk %s \n key share commit %v \n  key share poly %v \n key share %v \n commits %s \n privePoly %s", hex.EncodeToString(c.DKgSecretKey),
//			hex.EncodeToString(c.DKgJointPublicKey), c.keyShare.Commits, c.keyShare.PrivatePoly, c.keyShare.Share, hex.EncodeToString(c.CommitsData), hex.EncodeToString(c.PrivPolyData))
//	}
//	return fmt.Sprintf("sk %s\n pk %s \n key share  %v  \n commits %s \n privePoly %s", hex.EncodeToString(c.DKgSecretKey),
//		hex.EncodeToString(c.DKgJointPublicKey), c.keyShare, hex.EncodeToString(c.CommitsData), hex.EncodeToString(c.PrivPolyData))
//}
//
////SaveConsensusData
//func (d *archive.DkgPartner) SaveConsensusData() error {
//	config := d.generateConfig()
//	//
//	for i := 0; i < len(config.keyShare.Commits); i++ {
//		data, err := config.keyShare.Commits[i].MarshalBinary()
//		if err != nil {
//			panic(err)
//		}
//		config.CommitsData = append(config.CommitsData, data...)
//		config.CommitLen = append(config.CommitLen, len(data))
//	}
//	for i := 0; i < len(config.keyShare.PrivatePoly); i++ {
//		data, err := config.keyShare.PrivatePoly[i].MarshalBinary()
//		if err != nil {
//			panic(err)
//		}
//		config.PrivPolyData = append(config.PrivPolyData, data...)
//		config.PolyLen = append(config.PolyLen, len(data))
//	}
//
//	data, err := json.MarshalIndent(config.keyShare.Share, "", "\t")
//	if err != nil {
//		panic(err)
//	}
//
//	config.ShareData = data
//
//	data, err = json.MarshalIndent(config, "", "\t")
//	if err != nil {
//		panic(err)
//	}
//	absPath, err := filepath.Abs(d.ConfigFilePath)
//	if err != nil {
//		panic(fmt.Sprintf("Error on parsing config file path: %s %v err", absPath, err))
//	}
//	f, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
//	if err != nil {
//		panic(err)
//	}
//	defer f.Close()
//	f.Write(data)
//	return nil
//}
//
//func (d *archive.DkgPartner) generateConfig() DkgConfig {
//	var config DkgConfig
//	config.keyShare = d.context.KeyShare
//	pk, err := d.context.JointPubKey.MarshalBinary()
//	if err != nil {
//		dkg2.log.WithError(err).Error("joint publickey error")
//	}
//	config.DKgJointPublicKey = pk
//	sk, err := d.context.MyPartSec.MarshalBinary()
//	if err != nil {
//		dkg2.log.WithError(err).Error("joint publickey error")
//	}
//	config.DKgSecretKey = sk
//	config.PartnerId = d.context.Id
//	config.SigSets = d.blsSigSets
//	return config
//}
//
//func (d *archive.DkgPartner) LoadConsensusData() (*DkgConfig, error) {
//	suit := bn256.NewSuiteG2()
//	var config DkgConfig
//	keyShare := dkg.DistKeyShare{}
//
//	absPath, err := filepath.Abs(d.ConfigFilePath)
//	if err != nil {
//		return nil, fmt.Errorf("error on parsing config file path: %s %v", absPath, err)
//	}
//	data, err := ioutil.ReadFile(absPath)
//	if err != nil {
//		return nil, err
//	}
//	err = json.Unmarshal(data, &config)
//	if err != nil {
//		return nil, err
//	}
//	var k int
//	for i := 0; i < len(config.CommitLen); i++ {
//		data := make([]byte, config.CommitLen[i])
//		copy(data, config.CommitsData[k:k+config.CommitLen[i]])
//		k += config.CommitLen[i]
//		q, err := bn256.UnmarshalBinaryPointG2(data)
//		if err != nil {
//			dkg2.log.WithError(err).Error("unmarshal key share Commits error")
//			return nil, err
//		}
//		keyShare.Commits = append(keyShare.Commits, q)
//	}
//	k = 0
//	for i := 0; i < len(config.PolyLen); i++ {
//		data := make([]byte, config.PolyLen[i])
//		copy(data, config.PrivPolyData[k:k+config.PolyLen[i]])
//		k += config.PolyLen[i]
//		s := mod.NewInt64(0, bn256.Order)
//		dkg2.log.Debugln(k, config.PolyLen[i], hex.EncodeToString(data))
//		err = s.UnmarshalBinary(data)
//		if err != nil {
//			dkg2.log.WithError(err).Error("unmarshal key share Commits error")
//			return nil, err
//		}
//		keyShare.PrivatePoly = append(keyShare.PrivatePoly, s)
//	}
//
//	keyShare.Share = &share.PriShare{
//		I: 0,
//		V: suit.Scalar(),
//	}
//	err = json.Unmarshal(config.ShareData, keyShare.Share)
//	if err != nil {
//		dkg2.log.WithError(err).Error("unmarshal PrivatePoly  error")
//		return nil, err
//	}
//
//	if err != nil {
//		dkg2.log.WithError(err).Error("unmarshal key share key error")
//	} else {
//		config.keyShare = &keyShare
//	}
//	q, err := bn256.UnmarshalBinaryPointG2(config.DKgJointPublicKey)
//	if err != nil {
//		dkg2.log.WithError(err).Error("unmarshal public key error")
//		return nil, err
//	}
//	config.jointPubKey = q
//	g := suit.Scalar()
//	err = g.UnmarshalBinary(config.DKgSecretKey)
//	if err != nil {
//		dkg2.log.WithError(err).Error("unmarshal sk  error")
//		return nil, err
//	}
//	config.secretKey = g
//	return &config, nil
//}
//
//func (d *archive.DkgPartner) SetConfig(config *DkgConfig) {
//	d.context.KeyShare = config.keyShare
//	d.dkgOn = true
//	d.ready = true
//	d.context.MyPartSec = config.secretKey
//	d.context.JointPubKey = config.jointPubKey
//	d.context.Id = config.PartnerId
//	d.blsSigSets = config.SigSets
//	d.isValidPartner = true
//}
