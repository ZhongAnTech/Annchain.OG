// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	math2 "github.com/annchain/OG/arefactor/common/math"
	"math/big"

	"github.com/annchain/OG/vm/eth/common"
)

func memorySha3(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), stack.Back(1))
}

func memoryCallDataCopy(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), stack.Back(2))
}

func memoryReturnDataCopy(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), stack.Back(2))
}

func memoryCodeCopy(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), stack.Back(2))
}

func memoryExtCodeCopy(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(1), stack.Back(3))
}

func memoryMLoad(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), big.NewInt(32))
}

func memoryMStore8(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), big.NewInt(1))
}

func memoryMStore(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), big.NewInt(32))
}

func memoryCreate(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(1), stack.Back(2))
}

func memoryCreate2(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(1), stack.Back(2))
}

func memoryCall(stack *Stack) *big.Int {
	x := common.CalcMemSize(stack.Back(5), stack.Back(6))
	y := common.CalcMemSize(stack.Back(3), stack.Back(4))

	return math2.BigMax(x, y)
}

func memoryDelegateCall(stack *Stack) *big.Int {
	x := common.CalcMemSize(stack.Back(4), stack.Back(5))
	y := common.CalcMemSize(stack.Back(2), stack.Back(3))

	return math2.BigMax(x, y)
}

func memoryStaticCall(stack *Stack) *big.Int {
	x := common.CalcMemSize(stack.Back(4), stack.Back(5))
	y := common.CalcMemSize(stack.Back(2), stack.Back(3))

	return math2.BigMax(x, y)
}

func memoryReturn(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), stack.Back(1))
}

func memoryRevert(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), stack.Back(1))
}

func memoryLog(stack *Stack) *big.Int {
	mSize, mStart := stack.Back(1), stack.Back(0)
	return common.CalcMemSize(mStart, mSize)
}
