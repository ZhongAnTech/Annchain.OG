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
package sig_test

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
	"testing"
)

func TestSig(t *testing.T) {
	data, _ := hexutil.Decode("0x6060604052341561000f57600080fd5b336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555061117e8061005e6000396000f30060606040526004361061006d576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680634c725f4614610072578063893d20e8146102495780639d5af0f21461029e578063bea8fbc9146103c5578063de99aa451461044e575b600080fd5b341561007d57600080fd5b61012f600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509190803590602001908201803590602001908080601f01602080910402602001604051908101604052809392919081815260200183838082843782019150505050505091905050610635565b604051808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018060200180602001838103835285818151815260200191508051906020019080838360005b838110156101a557808201518184015260208101905061018a565b50505050905090810190601f1680156101d25780820380516001836020036101000a031916815260200191505b50838103825284818151815260200191508051906020019080838360005b8381101561020b5780820151818401526020810190506101f0565b50505050905090810190601f1680156102385780820380516001836020036101000a031916815260200191505b509550505050505060405180910390f35b341561025457600080fd5b61025c6108b0565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34156102a957600080fd5b6102de600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919080359060200190919050506108d9565b604051808060200180602001838103835285818151815260200191508051906020019080838360005b83811015610322578082015181840152602081019050610307565b50505050905090810190601f16801561034f5780820380516001836020036101000a031916815260200191505b50838103825284818151815260200191508051906020019080838360005b8381101561038857808201518184015260208101905061036d565b50505050905090810190601f1680156103b55780820380516001836020036101000a031916815260200191505b5094505050505060405180910390f35b34156103d057600080fd5b610405600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091908035906020019091905050610b36565b604051808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019250505060405180910390f35b341561045957600080fd5b610514600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001909190803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509190803590602001908201803590602001908080601f01602080910402602001604051908101604052809392919081815260200183838082843782019150505050505091905050610cd9565b604051808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018481526020018060200180602001838103835285818151815260200191508051906020019080838360005b83811015610590578082015181840152602081019050610575565b50505050905090810190601f1680156105bd5780820380516001836020036101000a031916815260200191505b50838103825284818151815260200191508051906020019080838360005b838110156105f65780820151818401526020810190506105db565b50505050905090810190601f1680156106235780820380516001836020036101000a031916815260200191505b50965050505050505060405180910390f35b600061063f610fa6565b610647610fa6565b61064f610fba565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156106aa57600080fd5b858160000181905250848160200181905250600160008873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020805480600101828161070d9190610fe0565b91600052602060002090600202016000839091909150600082015181600001908051906020019061073f929190611012565b50602082015181600101908051906020019061075c929190611012565b505050507f78dbaff0ef29afb97a47744dd2a02455409fc310530cdd20b2dbb6944265db0a878787604051808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018060200180602001838103835285818151815260200191508051906020019080838360005b838110156107fa5780820151818401526020810190506107df565b50505050905090810190601f1680156108275780820380516001836020036101000a031916815260200191505b50838103825284818151815260200191508051906020019080838360005b83811015610860578082015181840152602081019050610845565b50505050905090810190601f16801561088d5780820380516001836020036101000a031916815260200191505b509550505050505060405180910390a18686869350935093505093509350939050565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b6108e1610fa6565b6108e9610fa6565b600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020805490508310151561093957600080fd5b600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208381548110151561098557fe5b9060005260206000209060020201600001600160008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020848154811015156109e257fe5b9060005260206000209060020201600101818054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610a885780601f10610a5d57610100808354040283529160200191610a88565b820191906000526020600020905b815481529060010190602001808311610a6b57829003601f168201915b50505050509150808054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610b245780601f10610af957610100808354040283529160200191610b24565b820191906000526020600020905b815481529060010190602001808311610b0757829003601f168201915b50505050509050915091509250929050565b6000806000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610b9457600080fd5b600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208054905083101515610be457600080fd5b600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002083815481101515610c3057fe5b906000526020600020906002020160008082016000610c4f9190611092565b600182016000610c5f9190611092565b50507f9738979036ac493d6de9514633e66f68a70bdbca3100b46ce0cfe427ad581f378484604051808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019250505060405180910390a18383915091509250929050565b600080610ce4610fa6565b610cec610fa6565b610cf4610fba565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610d4f57600080fd5b600160008a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208054905088101515610d9f57600080fd5b86816000018190525085816020018190525080600160008b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002089815481101515610dfe57fe5b90600052602060002090600202016000820151816000019080519060200190610e28929190611012565b506020820151816001019080519060200190610e45929190611012565b509050507fa8ed599252d5b4b23e6be168f58e5f076c713668722fd77de15aaa99d7ae354689898989604051808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018481526020018060200180602001838103835285818151815260200191508051906020019080838360005b83811015610eea578082015181840152602081019050610ecf565b50505050905090810190601f168015610f175780820380516001836020036101000a031916815260200191505b50838103825284818151815260200191508051906020019080838360005b83811015610f50578082015181840152602081019050610f35565b50505050905090810190601f168015610f7d5780820380516001836020036101000a031916815260200191505b50965050505050505060405180910390a188888888945094509450945050945094509450949050565b602060405190810160405280600081525090565b6040805190810160405280610fcd6110da565b8152602001610fda6110da565b81525090565b81548183558181151161100d5760020281600202836000526020600020918201910161100c91906110ee565b5b505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061105357805160ff1916838001178555611081565b82800160010185558215611081579182015b82811115611080578251825591602001919060010190611065565b5b50905061108e919061112d565b5090565b50805460018160011615610100020316600290046000825580601f106110b857506110d7565b601f0160209004906000526020600020908101906110d6919061112d565b5b50565b602060405190810160405280600081525090565b61112a91905b80821115611126576000808201600061110d9190611092565b60018201600061111d9190611092565b506002016110f4565b5090565b90565b61114f91905b8082111561114b576000816000905550600101611133565b5090565b905600a165627a7a723058206d5361c53d71425ed79c6c32ebef1b2c5dde473b4f65a50f667efa7f339f066000290x3af39c21")
	fmt.Println(hexutil.Encode(data))
	pk, _ := crypto.PublicKeyFromString("0x010490e3c98826e0a530e22d34075e31cf478fead6297654dc5ce7e082fe1a29a3450ca669572bd257d2874c52158180b1cad654dbc091e92c40a983ee07361a17a7")
	//sk,_:= crypto.PrivateKeyFromString("0x01f2f9cfc3ca3d19654754045454487948448748734784234348079387342789342")
	tx := archive.Tx{
		TxBase: ogmessage.TxBase{
			Type:         archive.TxBaseTypeNormal,
			AccountNonce: 3,
			Hash:         common.HexToHash("0x49b51d4098087629f3489624951d3f81e3dfb87b8fcf3d0dae0474c7134908ab"),
			PublicKey:    pk.Bytes,
		},
		From:  common.HexToAddress("0x49fdaab0af739e16c9e1c9bf1715a6503edf4cab"),
		Value: math.NewBigInt(0),
		To:    common.Address{},
		Data:  data,
	}
	//fmt.Println(tx.SignatureTargets())
	fmt.Println(tx.Dump())
	sh := crypto.Sha256(tx.SignatureTargets())
	fmt.Println(sh)
	//fmt.Printf("%x ", tx.Data)
	//sig :=crypto.Signer.Sign(sk,tx.SignatureTargets())
	//fmt.Println(hexutil.Encode(sig.Bytes))
	//_ = sig
	//fmt.Println(sk.PublicKey().String())

}
