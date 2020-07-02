package og

import (
	"fmt"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/og_interface"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"testing"
	"time"
)

func TestAccountGenerator_Generate(t *testing.T) {
	randomReader := rand.New(rand.NewSource(time.Now().UnixNano()))
	g := &LocalLedgerAccountProvider{
		PrivateGenerator: &CachedPrivateGenerator{
			Reader: randomReader,
		},
		AddressConverter: &OgAddressConverter{},
		BackFilePath:     "",
		CryptoType:       og_interface.CryptoTypeSecp256k1,
		account:          nil,
	}

	for method := range []int{0, 1, 2, 3} {
		gotAccount, err := g.Generate()
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
		l := &LocalLedgerAccountProvider{
			PrivateGenerator: &CachedPrivateGenerator{},
			AddressConverter: &OgAddressConverter{},
			BackFilePath:     fmt.Sprintf("D:\\tmp\\test\\dump_%d.json", method),
			CryptoType:       og_interface.CryptoTypeSecp256k1,
			account:          gotAccount,
		}
		err = l.Save()
		utilfuncs.PanicIfError(err, "save account")
	}

}
