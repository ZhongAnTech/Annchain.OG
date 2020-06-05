package pool

import (
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/gcache"
	"github.com/sirupsen/logrus"
	"time"
)

type KnownTxiCacheConfig struct {
	MaxSize           int
	ExpirationSeconds int
}

// KnownTxiCache stores all known txs that may be fetched from the outside.
// If cache miss, it will try to load the tx from dag.
// but not txpool or txbuffer since the txs are probably in the cache after knowledge.
type KnownTxiCache struct {
	AdditionalHashLocators []LedgerHashLocator
	Config                 KnownTxiCacheConfig
	txiCache               gcache.Cache
}

func (k *KnownTxiCache) InitDefault() {
	// if there is a cache miss, gcache will try to load it from LedgerHashLocators (basically from disk)
	k.txiCache = gcache.New(k.Config.MaxSize).LRU().
		Expiration(time.Second * time.Duration(k.Config.ExpirationSeconds)).LoaderFunc(k.load).Build()
}

func (k *KnownTxiCache) Put(txi types.Txi) {
	if k.txiCache == nil {
		panic("not initialized")
	}

	err := k.txiCache.Set(txi.GetHash(), txi)
	if err != nil {
		logrus.WithError(err).Error("setting txi cache")
	}
}

func (k *KnownTxiCache) Get(hash types2.Hash) (txi types.Txi, err error) {
	if k.txiCache == nil {
		panic("not initialized")
	}

	v, err := k.txiCache.Get(hash)
	if err != nil {
		return
	}
	txi = v.(types.Txi)
	return
}

func (k *KnownTxiCache) load(hash interface{}) (txi interface{}, err error) {
	hasht := hash.(types2.Hash)
	for _, locator := range k.AdditionalHashLocators {
		txi = locator.GetTx(hasht)
		if txi == nil {
			err = gcache.KeyNotFoundError
		}
	}
	return
}
