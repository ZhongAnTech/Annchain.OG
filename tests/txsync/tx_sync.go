package main

import (
	"net/url"
	"github.com/sirupsen/logrus"
	"fmt"
	"github.com/gorilla/websocket"
	"encoding/json"
	"github.com/annchain/OG/wserver"
	"time"
	"net/http"
)

type pm struct {
	message string
	port    int
}

const PORT_OFFSET_WS = 2
const PORT_OFFSET_RPC = 0
const PORT_START = 8000
const PORT_GAP = 100
const NODES = 10

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	agg := make(chan pm, 1000)
	count := 0

	for port := PORT_START; port <= PORT_START+NODES*PORT_GAP; port += PORT_GAP {
		err := startListen(agg, port)
		if err != nil {
			continue
		}
		count += 1
	}
	mapcount := make(map[string]map[int]struct{})

	deleted := 0

	ticker := time.NewTicker(time.Second * 10)

	for {
		select {
		case m := <-agg:
			uidata := &wserver.UIData{}
			err := json.Unmarshal([]byte(m.message), uidata)
			if err != nil {
				logrus.WithField("port", m.port).WithError(err).Error("unmarshal")
				return
			}
			key := uidata.Nodes[0].Data.Unit_s
			logrus.WithFields(logrus.Fields{
				"port": m.port,
				"len":  len(uidata.Nodes),
				"tx":   key,
			}).Debug("new tx")

			if _, ok := mapcount[key]; !ok {
				mapcount[key] = make(map[int]struct{})
			}
			mapcount[key][m.port] = struct{}{}
			if len(mapcount[key]) == count {
				logrus.WithField("key", key).Debug("fully announced")
				delete(mapcount, key)
				deleted += 1
				changed := false
				for k := range mapcount {
					// manual query to double confirm
					for missingPort := PORT_START; missingPort <= PORT_START+NODES*PORT_GAP; missingPort += PORT_GAP {
						if _, ok := mapcount[k][missingPort]; !ok {
							if manualQuery(missingPort, k) {
								mapcount[k][missingPort] = struct{}{}
								changed = true
								logrus.WithField("hash", k).WithField("port", missingPort).
									Debug("manually fetched by rpc. ws missing?")
							}
						}
					}
				}
				if !changed {
					for k, v := range mapcount {
						logrus.WithField("key", k).WithField("value", v).Warn("Still not fulfilled")
					}
				}

			}
		case <-ticker.C:
			logrus.WithField("v", float64(deleted)/10).Info("tps")
			deleted = 0
		}

	}
}
func manualQuery(port int, hash string) bool {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/transaction?hash=%s", port+PORT_OFFSET_RPC, hash))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return true

}
func startListen(agg chan pm, port int) error {
	wsPort := port + PORT_OFFSET_WS
	u := url.URL{Scheme: "ws", Host: fmt.Sprintf("127.0.0.1:%d", wsPort), Path: "/ws"}
	logrus.Infof("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logrus.WithError(err).Errorf("dial")
		return err
	}

	go func() {
		err := c.WriteMessage(websocket.TextMessage, []byte("{\"event\": \"new_unit\"}"))
		if err != nil {
			logrus.WithField("port", wsPort).Error("write")
			return
		}

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logrus.WithError(err).WithField("port", wsPort).Error("read error")
				return
			}
			agg <- pm{message: string(message), port: port}
		}
	}()
	return nil
}
