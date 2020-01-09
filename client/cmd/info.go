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
