package sendArchives

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/client/httplib"
	"testing"
	"time"
)

type TxReq struct {
	Data []byte `json:"data"`
}

func TestArchives(t *testing.T) {
	Host := "http://192.168.45.149:11300"
	var i uint
	for {
		select {
		case <-time.After(time.Millisecond * 200):
			i++
			var data []byte
			if i > 255 {
				data = append(data, byte(i), byte(i), byte(i))
			} else {
				j := i / 255
				m := i % 255
				c := m / 255
				data = append(data, byte(c), byte(j), byte(m))
			}
			req := httplib.Post(Host + "/new_archive")

			txReq := TxReq{
				Data: data,
			}
			_, err := req.JSONBody(&txReq)
			if err != nil {
				panic(fmt.Errorf("encode tx errror %v", err))
			}
			d, _ := json.MarshalIndent(&txReq, "", "\t")
			fmt.Println(string(d))

			str, err := req.String()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(str)
		}
	}
}
