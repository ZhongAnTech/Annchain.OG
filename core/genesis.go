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
package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"crypto/sha256"
	"path/filepath"
)

const MaxAccountCount = 255

func DefaultGenesis(cryptoType crypto.CryptoType, genesisPath string) (*types.Sequencer, map[types.Address]*math.BigInt, string) {

	//crypto.SignerSecp256k1{},
	seq := newUnsignedSequencer(0, 0)
	seq.GetBase().Signature = common.FromHex("3044022012302bd7c951fcbfef2646d996fa42709a3cc35dfcaf480fa4f0f8782645585d0220424d7102da89f447b28c53aae388acf0ba57008c8048f5e34dc11765b1cab7f6")
	seq.GetBase().PublicKey = common.FromHex("b3e1b8306e1bab15ed51a4c24b086550677ba99cd62835965316a36419e8f59ce6a232892182da7401a329066e8fe2af607287139e637d314bf0d61cb9d1c7ee")
	hash := seq.CalcTxHash()
	seq.SetHash(hash)

	addr := types.HexToAddress("643d534e15a315173a3c18cd13c9f95c7484a9bc")
	balance := map[types.Address]*math.BigInt{}
	balance[addr] = math.NewBigInt(99999999)
	accounts := GetGenesisAccounts(cryptoType,genesisPath)
	//accounts := GetSampleAccounts(cryptoType)
	var buf bytes.Buffer
	for i := 0; i < len(accounts.Accounts); i++ {
		balance[accounts.Accounts[i].address] = math.NewBigInt(int64( accounts.Accounts[i].Balance))
		binary.Write(&buf,binary.BigEndian,accounts.Accounts[i].Address)
		binary.Write(&buf,binary.BigEndian,accounts.Accounts[i].Address)
	}
	h := sha256.New()
	h.Write(buf.Bytes())
	sum := h.Sum(nil)
	hash.MustSetBytes(sum, types.PaddingNone)
	seq.SetHash(hash)

	return seq.(*types.Sequencer), balance, accounts.MySecretKey
}


type Account struct {
	address    types.Address `json:"-"`
	Address    string `json:"address"`
	Balance    uint64 `json:"balance"`
}


type GenesisAccounts struct {
	MySecretKey   string `json:"my_secret_key"`
	Accounts []Account `json:"accounts"`
}


func GetMySecretKey(cryptoType crypto.CryptoType, genesisPath string) string  {
	accounts := GetGenesisAccounts(cryptoType,genesisPath)
	 return  accounts.MySecretKey

}

func GetGenesisAccounts(cryptoType crypto.CryptoType, genesisPath string) *GenesisAccounts{
	absPath, err := filepath.Abs(genesisPath)
	if err!=nil {
		panic(fmt.Sprintf("Error on parsing config file path: %s %v err", absPath,err))
	}
	data , err:= ioutil.ReadFile(absPath)
	if err!=nil {
		panic(err)
	}
	var accounts GenesisAccounts
	err = json.Unmarshal(data,&accounts)
	if err!=nil {
		panic(err)
	}
	for i:=0;i<len(accounts.Accounts);i++ {
         addr,err:= types.StringToAddress(accounts.Accounts[i].Address)
         if err!=nil {
         	panic(err)
		 }
		accounts.Accounts[i].address = addr
	}
	return  &accounts

}



func GetSampleAccounts(cryptoType crypto.CryptoType) []*account.SampleAccount {
	var accounts []*account.SampleAccount
	if cryptoType == crypto.CryptoTypeSecp256k1 {
		logrus.WithField("len", MaxAccountCount).Debug("Generating secp256k1 sample accounts")
		for i := 0; i < MaxAccountCount; i++ {
			acc := account.NewAccount(fmt.Sprintf("0x0170E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F%02X", i))
			acc.Id = i
			accounts = append(accounts, acc)
		}

	} else {
		logrus.WithField("len", math.MinInt(len(sampleEd25519PrivKeys), MaxAccountCount)).Debug("Generating ed25519 sample accounts")
		for i := 0; i < MaxAccountCount; i++ {
			if i >= len(sampleEd25519PrivKeys) {
				break
			}
			acc := account.NewAccount(sampleEd25519PrivKeys[i])
			acc.Id = i
			accounts = append(accounts, acc)
		}

	}
	return accounts
}

func newUnsignedSequencer(height uint64, accountNonce uint64) types.Txi {
	tx := types.Sequencer{
		TxBase: types.TxBase{
			AccountNonce: accountNonce,
			Type:         types.TxBaseTypeSequencer,
			Height:       height,
		},
	}
	return &tx
}
