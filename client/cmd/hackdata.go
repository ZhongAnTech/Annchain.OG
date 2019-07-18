package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	hackdataCmd = &cobra.Command{
		Use:   "hackdata",
		Short: "data generator for hackathon register.",
		Run:   hackData,
	}
)

func hackData(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("please enter the phone number of your register team")
		return
	}

	phoneStr := args[0]
	if len(phoneStr) > 20 {
		fmt.Println("too long for a phone number.")
		return
	}

	phoneLenInRLP := fmt.Sprintf("%x", len(phoneStr))
	phoneLenInRLP = BaseRlpHex[:64-len(phoneLenInRLP)] + phoneLenInRLP

	phoneInRLP := fmt.Sprintf("%x", []byte(phoneStr))
	phoneInRLP = phoneInRLP + BaseRlpHex[len(phoneInRLP):]

	dataFormat := "e978c36f" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		phoneLenInRLP +
		phoneInRLP

	fmt.Println(dataFormat)
}

var BaseRlpHex = "0000000000000000000000000000000000000000000000000000000000000000"
