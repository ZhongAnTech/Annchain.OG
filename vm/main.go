package main

import (
	"github.com/annchain/OG/vm/eth/core/vm/runtime"
	"fmt"
	"github.com/annchain/OG/common"
)

func ExampleExecute() {
	ret, state, err := runtime.Execute(common.Hex2Bytes("6060604052600a8060106000396000f360606040526008565b00"), nil, nil)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(ret)
	fmt.Println(state.Dump())
	// Output:
	// [96 96 96 64 82 96 8 86 91 0]
}


// loads a solidity file and run it
func main() {
	ExampleExecute()
}