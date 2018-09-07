package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/client/httplib"
	"github.com/spf13/cobra"
)

var (
	InfoCmd = &cobra.Command{
		Use:   "info",
		Short: "get chorus info",
		Run:   status,
	}
	netInfoCmd = &cobra.Command{
		Use:   "net",
		Short: "net_info",
		Run:   netInfo,
	}
)

func status(cmd *cobra.Command, args []string) {
	requesrUrl := Host + "/status"
	req := httplib.Get(requesrUrl)
	data, err := req.Bytes()
	if err != nil {
		fmt.Println(err)
		return
	}
	var out bytes.Buffer
	json.Indent(&out, data, "", "\t")
	fmt.Println(out.String())
}

func netInfo(cmd *cobra.Command, args []string) {

}
