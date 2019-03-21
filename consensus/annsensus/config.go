package annsensus

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/group/mod"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/OG/types"
	"io/ioutil"
	"os"
	"path/filepath"
)

type AnnSensusConfig struct {
	DKgSecretKey      []byte `json:"d_kg_secret_key"`
	DKgJointPublicKey []byte `json:"d_kg_joint_public_key"`
	jointPubKey       kyber.Point
	secretKey         kyber.Scalar
	keyShare          *dkg.DistKeyShare
	CommitLen         []int  `json:"commit_len"`
	PolyLen           []int  `json:"poly_len"`
	CommitsData       []byte `json:"commits_data"`
	PrivPolyData      []byte `json:"priv_poly_data"`
	ShareData         []byte `json:"share_data"`
	PartnerId         uint32 `json:"partner_id"`
	SigSets           map[types.Address]*types.SigSet
}

func (c AnnSensusConfig) String() string {
	if c.keyShare != nil {
		return fmt.Sprintf("sk %s\n pk %s \n key share commit %v \n  key share poly %v \n key share %v \n commits %s \n privePoly %s", hex.EncodeToString(c.DKgSecretKey),
			hex.EncodeToString(c.DKgJointPublicKey), c.keyShare.Commits, c.keyShare.PrivatePoly, c.keyShare.Share, hex.EncodeToString(c.CommitsData), hex.EncodeToString(c.PrivPolyData))
	}
	return fmt.Sprintf("sk %s\n pk %s \n key share  %v  \n commits %s \n privePoly %s", hex.EncodeToString(c.DKgSecretKey),
		hex.EncodeToString(c.DKgJointPublicKey), c.keyShare, hex.EncodeToString(c.CommitsData), hex.EncodeToString(c.PrivPolyData))
}

//SaveConsensusData
func (a *AnnSensus) SaveConsensusData() error {
	config := a.generateConfig()
	//
	for i := 0; i < len(config.keyShare.Commits); i++ {
		data, err := config.keyShare.Commits[i].MarshalBinary()
		if err != nil {
			panic(err)
		}
		config.CommitsData = append(config.CommitsData, data...)
		config.CommitLen = append(config.CommitLen, len(data))
	}
	for i := 0; i < len(config.keyShare.PrivatePoly); i++ {
		data, err := config.keyShare.PrivatePoly[i].MarshalBinary()
		if err != nil {
			panic(err)
		}
		config.PrivPolyData = append(config.PrivPolyData, data...)
		config.PolyLen = append(config.PolyLen, len(data))
	}

	data, err := json.MarshalIndent(config.keyShare.Share, "", "\t")
	if err != nil {
		panic(err)
	}

	config.ShareData = data

	data, err = json.MarshalIndent(config, "", "\t")
	if err != nil {
		panic(err)
	}
	absPath, err := filepath.Abs(a.ConfigFilePath)
	if err != nil {
		panic(fmt.Sprintf("Error on parsing config file path: %s %v err", absPath, err))
	}
	f, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.Write(data)
	return nil
}

func (a *AnnSensus) generateConfig() AnnSensusConfig {
	var config AnnSensusConfig
	config.keyShare = a.dkg.partner.KeyShare
	pk, err := a.dkg.partner.jointPubKey.MarshalBinary()
	if err != nil {
		log.WithError(err).Error("joint publickey error")
	}
	config.DKgJointPublicKey = pk
	sk, err := a.dkg.partner.MyPartSec.MarshalBinary()
	if err != nil {
		log.WithError(err).Error("joint publickey error")
	}
	config.DKgSecretKey = sk
	config.PartnerId = a.dkg.partner.Id
	config.SigSets = a.dkg.blsSigSets
	return config
}

func (a *AnnSensus) LoadConsensusData() (*AnnSensusConfig, error) {
	suit := bn256.NewSuiteG2()
	var config AnnSensusConfig
	keyShare := dkg.DistKeyShare{}

	absPath, err := filepath.Abs(a.ConfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("error on parsing config file path: %s %v", absPath, err)
	}
	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	var k int
	for i := 0; i < len(config.CommitLen); i++ {
		data := make([]byte, config.CommitLen[i])
		copy(data, config.CommitsData[k:k+config.CommitLen[i]])
		k += config.CommitLen[i]
		q, err := bn256.UnmarshalBinaryPointG2(data)
		if err != nil {
			log.WithError(err).Error("unmarshal key share Commits error")
			return nil, err
		}
		keyShare.Commits = append(keyShare.Commits, q)
	}
	k = 0
	for i := 0; i < len(config.PolyLen); i++ {
		data := make([]byte, config.PolyLen[i])
		copy(data, config.PrivPolyData[k:k+config.PolyLen[i]])
		k += config.PolyLen[i]
		s := mod.NewInt64(0, bn256.Order)
		log.Debugln(k, config.PolyLen[i], hex.EncodeToString(data))
		err = s.UnmarshalBinary(data)
		if err != nil {
			log.WithError(err).Error("unmarshal key share Commits error")
			return nil, err
		}
		keyShare.PrivatePoly = append(keyShare.PrivatePoly, s)
	}

	keyShare.Share = &share.PriShare{
		I: 0,
		V: suit.Scalar(),
	}
	err = json.Unmarshal(config.ShareData, keyShare.Share)
	if err != nil {
		log.WithError(err).Error("unmarshal PrivatePoly  error")
		return nil, err
	}

	if err != nil {
		log.WithError(err).Error("unmarshal key share key error")
	} else {
		config.keyShare = &keyShare

	}
	q, err := bn256.UnmarshalBinaryPointG2(config.DKgJointPublicKey)
	if err != nil {
		log.WithError(err).Error("unmarshal public key error")
		return nil, err
	}
	config.jointPubKey = q
	g := suit.Scalar()
	err = g.UnmarshalBinary(config.DKgSecretKey)
	if err != nil {
		log.WithError(err).Error("unmarshal sk  error")
		return nil, err
	}
	config.secretKey = g
	return &config, nil
}

func (a *AnnSensus) SetConfig(config *AnnSensusConfig) {
	a.dkg.partner.KeyShare = config.keyShare
	a.dkg.dkgOn = true
	a.dkg.ready = true
	a.dkg.partner.MyPartSec = config.secretKey
	a.dkg.partner.jointPubKey = config.jointPubKey
	a.dkg.partner.Id = config.PartnerId
	a.dkg.blsSigSets = config.SigSets

}
