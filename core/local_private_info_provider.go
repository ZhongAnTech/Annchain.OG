package core

import (
	"encoding/json"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/arefactor/common/io"
	"github.com/spf13/viper"
	"io/ioutil"
)

type LocalPrivateInfoProvider struct {
}

func (l LocalPrivateInfoProvider) PrivateInfo() *account.PrivateInfo {
	// read key file
	bytes, err := ioutil.ReadFile(io.FixPrefixPath(viper.GetString("iddir"), "nodekey.json"))
	if err != nil {
		panic(err)
	}

	pi := &account.PrivateInfo{}
	err = json.Unmarshal(bytes, pi)
	if err != nil {
		panic(err)
	}
	return pi
}
