package dummy

import (
	"github.com/annchain/OG/arefactor/consensus_interface"
	"io/ioutil"
)

type DummyConsensusAccount struct {
	id string
}

func (d DummyConsensusAccount) Id() string {
	return d.id
}

type DummyConsensusAccountProvider struct {
	BackFilePath string
	account      consensus_interface.ConsensusAccount
}

func (d *DummyConsensusAccountProvider) ProvideAccount() (consensus_interface.ConsensusAccount, error) {
	if d.account == nil {
		account, err := d.Load()
		if err != nil {
			d.Generate()
		} else {
			d.account = account
		}
	}
	return d.account, nil
}

func (d *DummyConsensusAccountProvider) Generate() (account consensus_interface.ConsensusAccount, err error) {
	d.account =
		&DummyConsensusAccount{
			id: "dummy1",
		}
	return d.account, nil
}

func (d *DummyConsensusAccountProvider) Load() (account consensus_interface.ConsensusAccount, err error) {
	byteContent, err := ioutil.ReadFile(d.BackFilePath)
	if err != nil {
		return
	}
	d.account =
		&DummyConsensusAccount{
			id: string(byteContent),
		}
	return d.account, nil
}

func (d *DummyConsensusAccountProvider) Save() (err error) {
	err = ioutil.WriteFile(d.BackFilePath, []byte(d.account.Id()), 0600)
	return
}
