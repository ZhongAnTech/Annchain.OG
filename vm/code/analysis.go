// Copyright 2014 The go-ethereum Authors
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

package code

import (
	"github.com/annchain/OG/vm/instruction"
	"github.com/annchain/OG/vm/common"
)

// codeBitmap collects data locations in code.
func CodeBitmap(code []byte) common.Bitvec {
	// The bitmap is 4 bytes longer than necessary, in case the code
	// ends with a PUSH32, the algorithm will push zeroes onto the
	// Bitvector outside the bounds of the actual code.
	bits := make(common.Bitvec, len(code)/8+1+4)
	for pc := uint64(0); pc < uint64(len(code)); {
		op := instruction.OpCode(code[pc])

		if op >= instruction.PUSH1 && op <= instruction.PUSH32 {
			numbits := op - instruction.PUSH1 + 1
			pc++
			for ; numbits >= 8; numbits -= 8 {
				bits.Set8(pc) // 8
				pc += 8
			}
			for ; numbits > 0; numbits-- {
				bits.Set(pc)
				pc++
			}
		} else {
			pc++
		}
	}
	return bits
}
