package communication

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/ogcore/message"
)

type OgPeer struct {
	Id             int
	PublicKey      crypto.PublicKey `json:"-"`
	Address        common.Address   `json:"address"`
	PublicKeyBytes hexutil.Bytes    `json:"public_key"`
}

func (o OgPeer) String() string {
	return fmt.Sprintf("Peer%d", o.Id)
}

type OgMessageEvent struct {
	Message message.OgMessage
	Peer    *OgPeer
}
