package og

import (
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/common/crypto"
)

func DefaultGenesis() (*types.Sequencer, map[types.Address]*math.BigInt) {
	txCreator := TxCreator{
		Signer: &crypto.SignerSecp256k1{},
	}
	seq := txCreator.NewUnsignedSequencer(0, []types.Hash{}, 0)
	seq.GetBase().Signature = common.FromHex("3044022012302bd7c951fcbfef2646d996fa42709a3cc35dfcaf480fa4f0f8782645585d0220424d7102da89f447b28c53aae388acf0ba57008c8048f5e34dc11765b1cab7f6")
	seq.GetBase().PublicKey = common.FromHex("b3e1b8306e1bab15ed51a4c24b086550677ba99cd62835965316a36419e8f59ce6a232892182da7401a329066e8fe2af607287139e637d314bf0d61cb9d1c7ee")

	addr := types.HexToAddress("643d534e15a315173a3c18cd13c9f95c7484a9bc")
	balance := map[types.Address]*math.BigInt{}
	balance[addr] = math.NewBigInt(99999999)

	return seq.(*types.Sequencer), balance
}
