package message

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
)

type GeneralMessageType byte

type GeneralMessage interface {
	GetType() GeneralMessageType
	GetBytes() []byte
	String() string
}

type GeneralPeer struct {
	Id             int
	PublicKey      crypto.PublicKey `json:"-"`
	Address        common.Address   `json:"address"`
	PublicKeyBytes hexutil.Bytes    `json:"public_key"`
}

type GeneralMessageEvent struct {
	Message GeneralMessage
	Sender  GeneralPeer
}
