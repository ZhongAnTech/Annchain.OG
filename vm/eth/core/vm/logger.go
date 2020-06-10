// Copyright 2015 The go-ethereum Authors
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
	"encoding/hex"
	"fmt"
	math2 "github.com/annchain/OG/arefactor/common/math"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/vm/eth/common/hexutil"
	"github.com/annchain/OG/vm/instruction"
	vmtypes "github.com/annchain/OG/vm/types"
	"io"
	"math/big"
	"sort"
	"strings"
	"time"
)

// Storage represents a contract's storage.
type Storage map[types.Hash]types.Hash

// Copy duplicates the current storage.
func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}

	return cpy
}

// LogConfig are the configuration options for structured logger the OVM
type LogConfig struct {
	DisableMemory  bool // disable memory capture
	DisableStack   bool // disable stack capture
	DisableStorage bool // disable storage capture
	Debug          bool // print output during capture end
	Limit          int  // maximum length of output, but zero means unlimited
}

//go:generate gencodec -type StructLog -field-override structLogMarshaling -out gen_structlog.go

// StructLog is emitted to the OVM each cycle and lists information about the current internal state
// prior to the execution of the statement.
type StructLog struct {
	Pc            uint64                    `json:"pc"`
	Op            instruction.OpCode        `json:"op"`
	Gas           uint64                    `json:"gas"`
	GasCost       uint64                    `json:"gasCost"`
	Memory        []byte                    `json:"memory"`
	MemorySize    int                       `json:"memSize"`
	Stack         []*big.Int                `json:"stack"`
	Storage       map[types.Hash]types.Hash `json:"-"`
	Depth         int                       `json:"depth"`
	RefundCounter uint64                    `json:"refund"`
	Err           error                     `json:"-"`
}

// overrides for gencodec
type structLogMarshaling struct {
	Stack       []*math2.HexOrDecimal256
	Gas         math2.HexOrDecimal64
	GasCost     math2.HexOrDecimal64
	Memory      hexutil.Bytes
	OpName      string `json:"opName"` // adds call to OpName() in MarshalJSON
	ErrorString string `json:"error"`  // adds call to ErrorString() in MarshalJSON
}

// OpName formats the operand name in a human-readable format.
func (s *StructLog) OpName() string {
	return s.Op.String()
}

// ErrorString formats the log's error as a string.
func (s *StructLog) ErrorString() string {
	if s.Err != nil {
		return s.Err.Error()
	}
	return ""
}

// StructLogger is an OVM state logger and implements Tracer.
//
// StructLogger can capture state based on the given Log configuration and also keeps
// a track record of modified storage which is used in reporting snapshots of the
// contract their storage.
type StructLogger struct {
	cfg LogConfig

	Logs          []StructLog
	changedValues map[common.Address]Storage
	output        []byte
	err           error
}

// NewStructLogger returns a new logger
func NewStructLogger(cfg *LogConfig) *StructLogger {
	logger := &StructLogger{
		changedValues: make(map[common.Address]Storage),
	}
	if cfg != nil {
		logger.cfg = *cfg
	}
	return logger
}

// CaptureStart implements the Tracer interface to initialize the tracing operation.
func (l *StructLogger) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) error {
	return nil
}

// CaptureState Logs a new structured log message and pushes it out to the environment
//
// CaptureState also tracks SSTORE ops to track dirty values.
func (l *StructLogger) CaptureState(ctx *vmtypes.Context, pc uint64, op instruction.OpCode, gas, cost uint64, memory *Memory, stack *Stack, contract *vmtypes.Contract, depth int, err error) error {
	// check if already accumulated the specified number of Logs
	if l.cfg.Limit != 0 && l.cfg.Limit <= len(l.Logs) {
		return vmtypes.ErrTraceLimitReached
	}

	// initialise new changed values storage container for this contract
	// if not present.
	if l.changedValues[contract.Address()] == nil {
		l.changedValues[contract.Address()] = make(Storage)
	}

	// capture SSTORE opcodes and determine the changed value and store
	// it in the local storage container.
	if op == instruction.SSTORE && stack.len() >= 2 {
		var (
			value   = types.BigToHash(stack.data[stack.len()-2])
			address = types.BigToHash(stack.data[stack.len()-1])
		)
		l.changedValues[contract.Address()][address] = value
	}
	// Copy a snapstot of the current memory state to a new buffer
	var mem []byte
	if !l.cfg.DisableMemory {
		mem = make([]byte, len(memory.Data()))
		copy(mem, memory.Data())
	}
	// Copy a snapshot of the current stack state to a new buffer
	var stck []*big.Int
	if !l.cfg.DisableStack {
		stck = make([]*big.Int, len(stack.Data()))
		for i, item := range stack.Data() {
			stck[i] = new(big.Int).Set(item)
		}
	}
	// Copy a snapshot of the current storage to a new container
	var storage Storage
	if !l.cfg.DisableStorage {
		storage = l.changedValues[contract.Address()].Copy()
	}
	// create a new snaptshot of the OVM.
	log := StructLog{pc, op, gas, cost, mem, memory.Len(), stck, storage, depth, ctx.StateDB.GetRefund(), err}

	l.Logs = append(l.Logs, log)
	return nil
}

// CaptureFault implements the Tracer interface to trace an execution fault
// while running an opcode.
func (l *StructLogger) CaptureFault(ctx *vmtypes.Context, pc uint64, op instruction.OpCode, gas, cost uint64, memory *Memory, stack *Stack, contract *vmtypes.Contract, depth int, err error) error {
	return nil
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (l *StructLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error {
	l.output = output
	l.err = err
	if l.cfg.Debug {
		fmt.Printf("0x%x\n", output)
		if err != nil {
			fmt.Printf(" error: %v\n", err)
		}
	}
	return nil
}

// StructLogs returns the captured log entries.
func (l *StructLogger) StructLogs() []StructLog { return l.Logs }

// Error returns the VM error captured by the trace.
func (l *StructLogger) Error() error { return l.err }

// Output returns the VM return value captured by the trace.
func (l *StructLogger) Output() []byte { return l.output }

func (l *StructLogger) Write(writer io.Writer) {
	WriteTrace(writer, l.Logs)
}

// WriteTrace writes a formatted trace to the given writer
func WriteTrace(writer io.Writer, logs []StructLog) {
	omitZero := true
	for _, log := range logs {
		fmt.Fprintf(writer, "%-16spc=%08d gas=%v cost=%v", log.Op, log.Pc, log.Gas, log.GasCost)
		if log.Err != nil {
			fmt.Fprintf(writer, " ERROR: %v", log.Err)
		}
		fmt.Fprintln(writer)

		if len(log.Stack) > 0 {
			fmt.Fprintln(writer, "Stack:")
			for i := len(log.Stack) - 1; i >= 0; i-- {
				s := fmt.Sprintf("%08d  %x", len(log.Stack)-i-1, math2.PaddedBigBytes(log.Stack[i], 32))
				if omitZero {
					s = strings.Replace(s, "00", "__", -1)
				}
				fmt.Println(s)
			}
		}
		if len(log.Memory) > 0 {
			fmt.Fprintln(writer, "Memory:")
			fmt.Fprint(writer, hex.Dump(log.Memory))
		}
		if len(log.Storage) > 0 {
			fmt.Fprintln(writer, "Storage:")
			var keys types.Hashes
			for h := range log.Storage {
				keys = append(keys, h)
			}
			sort.Sort(keys)

			for _, key := range keys {
				item := log.Storage[key]
				s := fmt.Sprintf("%s: %s", key.Hex(), item.Hex())
				if omitZero {
					s = strings.Replace(s, "00", "__", -1)
				}
				fmt.Println(s)

			}
		}
		fmt.Fprintln(writer)
	}
}

// WriteLogs writes vm Logs in a readable format to the given writer
func WriteLogs(writer io.Writer, logs []*vmtypes.Log) {
	for _, log := range logs {
		fmt.Fprintf(writer, "LOG%d: %x seq=%d txi=%x\n", len(log.Topics), log.Address, log.SequenceID, log.TxIndex)

		for i, topic := range log.Topics {
			fmt.Fprintf(writer, "%08d  %x\n", i, topic)
		}

		fmt.Fprint(writer, hex.Dump(log.Data))
		fmt.Fprintln(writer)
	}
}
