package og

import (
	"fmt"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/og/types"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"testing"
	"time"
)

func TestAccountGenerator_Generate(t *testing.T) {
	randomReader := rand.New(rand.NewSource(time.Now().UnixNano()))
	g := &LedgerAccountGenerator{
		AddressConverter: &OgAddressConverter{},
	}

	for method := range []int{0, 1, 2, 3} {
		gotAccount, err := g.Generate(types.CryptoType(method), randomReader)
		utilfuncs.PanicIfError(err, "generate")
		{
			bytes, err := gotAccount.PrivateKey.Bytes()
			utilfuncs.PanicIfError(err, "PrivateKey")
			log.Info(hexutil.ToHex(bytes))
			log.Info(len(bytes))
		}
		{
			bytes, err := gotAccount.PrivateKey.Raw()
			utilfuncs.PanicIfError(err, "PrivateKey")
			log.Info(hexutil.ToHex(bytes))
			log.Info(len(bytes))
		}

		{
			bytes, err := gotAccount.PublicKey.Bytes()
			utilfuncs.PanicIfError(err, "PublicKey")
			log.Info(hexutil.ToHex(bytes))
			log.Info(len(bytes))
		}
		{
			bytes, err := gotAccount.PublicKey.Raw()
			utilfuncs.PanicIfError(err, "PublicKey")
			log.Info(hexutil.ToHex(bytes))
			log.Info(len(bytes))
		}

		{
			bytes := gotAccount.Address.Bytes()
			log.Info(hexutil.ToHex(bytes))
			log.Info(len(bytes))
		}
		log.Info()
		l := &LedgerAccountHolder{
			CryptoType: types.CryptoType(method),
			account:    gotAccount,
		}
		err = l.Save(fmt.Sprintf("D:\\tmp\\test\\dump_%d.json", method), gotAccount)
		utilfuncs.PanicIfError(err, "save account")
	}

}
