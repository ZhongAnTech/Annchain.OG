package runtime

import "github.com/annchain/OG/vm/types"

// Interpreter is used to run Ethereum based vmtypes.Contracts and will utilise the
// passed environment to query external sources for state information.
// The Interpreter will run the byte code VM based on the passed
// configuration.
type Interpreter interface {
	// Run loops and evaluates the vmtypes.Contract's code with the given input data and returns
	// the return byte-slice and an error if one occurred.
	Run(Contract *types.Contract, input []byte, static bool) ([]byte, error)
	// CanRun tells if the vmtypes.Contract, passed as an argument, can be
	// run by the current interpreter. This is meant so that the
	// caller can do something like:
	//
	// ```golang
	// for _, interpreter := range interpreters {
	//   if interpreter.CanRun(vmtypes.Contract.code) {
	//     interpreter.Run(vmtypes.Contract.code, input)
	//   }
	// }
	// ```
	CanRun([]byte) bool
}
