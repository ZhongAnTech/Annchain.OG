package core

import (
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
)

const MaxAccountCount = 20

func DefaultGenesis() (*types.Sequencer, map[types.Address]*math.BigInt) {

	//crypto.SignerSecp256k1{},
	seq := newUnsignedSequencer(0, []types.Hash{}, 0)
	seq.GetBase().Signature = common.FromHex("3044022012302bd7c951fcbfef2646d996fa42709a3cc35dfcaf480fa4f0f8782645585d0220424d7102da89f447b28c53aae388acf0ba57008c8048f5e34dc11765b1cab7f6")
	seq.GetBase().PublicKey = common.FromHex("b3e1b8306e1bab15ed51a4c24b086550677ba99cd62835965316a36419e8f59ce6a232892182da7401a329066e8fe2af607287139e637d314bf0d61cb9d1c7ee")
	hash := seq.CalcTxHash()
	seq.SetHash(hash)

	addr := types.HexToAddress("643d534e15a315173a3c18cd13c9f95c7484a9bc")
	balance := map[types.Address]*math.BigInt{}
	balance[addr] = math.NewBigInt(99999999)

	accounts := GetSampleAccounts()
	for i := 0; i < MaxAccountCount; i++ {
		balance[accounts[i].Address] = math.NewBigInt(8888888)
	}
	return seq.(*types.Sequencer), balance
}

func GetSampleAccounts() []account.SampleAccount {
	var accounts []account.SampleAccount
	for i := 0; i < MaxAccountCount; i++ {
		accounts = append(accounts, account.NewAccount(fmt.Sprintf("0x0170E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F%02d", i)))
	}
	return accounts
}

func newUnsignedSequencer(id uint64, contractHashOrder []types.Hash, accountNonce uint64) types.Txi {
	tx := types.Sequencer{
		Id:                id,
		ContractHashOrder: contractHashOrder,
		TxBase: types.TxBase{
			AccountNonce: accountNonce,
			Type:         types.TxBaseTypeSequencer,
		},
	}
	return &tx
}
