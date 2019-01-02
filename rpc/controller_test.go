package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	ROOT             = "http://localhost:8000"
	PATH_NEW_ACCOUNT = "/new_account"
	PATH_NEW_TX      = "/new_transaction"
	PATH_NONCE       = "/query_nonce"
)

func TestNewAccount(t *testing.T) {
	pri, pub, addr, err := newAccount("secp256k1")
	if err != nil {
		t.Error(err.Error())
	} else {
		fmt.Println("prikey: ", pri)
		fmt.Println("pubkey: ", pub)
		fmt.Println("address:", addr)
	}
}

type account struct {
	Privkey string `json:"privkey"`
	Pubkey  string `json:"pubkey"`
}

func newAccount(algorithm string) (string, string, string, error) {
	body := map[string]string{"algorithm": algorithm}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return "", "", "", err
	}
	resp, err := http.Post(ROOT+PATH_NEW_ACCOUNT, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	var a account
	var signer crypto.Signer
	io.Copy(&buf, resp.Body)
	err = json.Unmarshal(buf.Bytes(), &a)
	if err != nil {
		return "", "", "", err
	}
	switch algorithm {
	case "secp256k1":
		signer = &crypto.SignerSecp256k1{}
	case "ed25519":
		signer = &crypto.SignerEd25519{}
	}
	pubkey, err := crypto.PublicKeyFromString(a.Pubkey)
	if err != nil {
		return "", "", "", err
	}
	addr := signer.Address(pubkey)
	return a.Privkey, a.Pubkey, addr.String(), nil
}
func TestQueryNonce(t *testing.T) {
	_, _, addr, err := newAccount("secp256k1")
	if err != nil {
		t.Error(err.Error())
		return
	}

	n, err := nonce(addr)
	if err != nil {
		t.Error(err.Error())
		return
	}
	t.Log(n)
	//nonce("0xcfad46e0bcd2229f6cdd6fdc36365b738127b7a6")
}

type nonceResp struct {
	Nonce int `json:"nonce"`
}

func nonce(addr string) (int, error) {
	resp, err := http.Get(ROOT + PATH_NONCE + "?address=" + addr)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	var n nonceResp
	io.Copy(&buf, resp.Body)
	err = json.Unmarshal(buf.Bytes(), &n)
	if err != nil {
		return 0, err
	}
	return n.Nonce, nil
}

func TestSendTx(t *testing.T) {
	sendTx("secp256k1")
}

func sendTx(algorithm string) {
	priv1, pub1, addr1, err := newAccount(algorithm)
	if err != nil {
		return
	}

	_, _, addr2, err := newAccount(algorithm)
	if err != nil {
		return
	}

	fromPriv, err := crypto.PrivateKeyFromString(priv1)
	if err != nil {
		return
	}
	fromPub, err := crypto.PublicKeyFromString(pub1)
	if err != nil {
		return
	}
	fromAddr, err := types.StringToAddress(addr1)
	if err != nil {
		return
	}
	toAddr, err := types.StringToAddress(addr2)

	var signer crypto.Signer
	switch algorithm {
	case "secp256k1":
		signer = &crypto.SignerSecp256k1{}
	case "ed25519":
		signer = &crypto.SignerEd25519{}
	}

	for nonce := 0; nonce < 10; nonce++ {
		tx := types.Tx{
			TxBase: types.TxBase{
				AccountNonce: uint64(nonce),
			},
			From:  fromAddr,
			To:    toAddr,
			Value: math.NewBigInt(0),
		}

		signature := signer.Sign(fromPriv, tx.SignatureTargets())

		newTxData := map[string]string{
			"nonce":     fmt.Sprintf("%d", tx.TxBase.AccountNonce),
			"from":      tx.From.String(),
			"to":        tx.To.String(),
			"value":     fmt.Sprintf("%d", tx.Value.GetInt64()),
			"signature": hexutil.Encode(signature.Bytes),
			"pubkey":    fromPub.PublicKeyToString(),
		}

		jsonData, err := json.Marshal(newTxData)
		if err != nil {
			return
		}
		resp, err := http.Post(ROOT+PATH_NEW_TX, "appliaction/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return
		}
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)
		time.Sleep(time.Millisecond * 100)
	}

}
