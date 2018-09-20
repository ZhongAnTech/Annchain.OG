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
const NODES = 30
const REPORT_INTERVAL_SECONDS = 5

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	logrus.SetLevel(logrus.InfoLevel)
	agg := make(chan pm, 1000)
	count := 0

	maxPort := PORT_START

	for port := PORT_START; port <= PORT_START+NODES*PORT_GAP; port += PORT_GAP {
		err := startListen(agg, port)
		if err != nil {
			continue
		}
		if port > maxPort {
			maxPort = port
		}
		count += 1
	}
	mapcount := make(map[string]map[int]struct{})

	deleted := 0

	ticker := time.NewTicker(time.Second * REPORT_INTERVAL_SECONDS)

	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = time.Second * 5

	lastCheck := time.Now()

	for {
		select {
		case <-ticker.C:
			logrus.Warn("Ticker")
			toDelete := []string{}
			for k := range mapcount {
				changed := false
				// manual query to double confirm
				for missingPort := PORT_START; missingPort <= maxPort; missingPort += PORT_GAP {
					if _, ok := mapcount[k][missingPort]; !ok {
						if manualQuery(missingPort, k) {
							mapcount[k][missingPort] = struct{}{}
							changed = true
							logrus.WithField("hash", k).WithField("port", missingPort).
								Debug("manually fetched by rpc. ws missing?")
						}
					}
				}
				if !changed {
					if len(mapcount[k]) == count {
						logrus.WithField("hash", k).Debug("Should be fulfilled but not.")
						toDelete = append(toDelete, k)
					} else {
						logrus.WithField("key", k).WithField("value", mapcount[k]).Warn("Still not fulfilled")
					}
				}
			}
			for _, k := range toDelete {
				delete(mapcount, k)
				deleted += 1
			}
			logrus.WithField("count", deleted).
				WithField("v", float64(deleted)/(time.Since(lastCheck).Seconds())).
				Info("tps")
			lastCheck = time.Now()
			deleted = 0
		case m := <-agg:
			uidata := &wserver.UIData{}
			err := json.Unmarshal([]byte(m.message), uidata)
			if err != nil {
				logrus.WithField("port", m.port).WithError(err).Error("unmarshal")
				return
			}
			key := uidata.Nodes[0].Data.Unit
			//logrus.WithFields(logrus.Fields{
			//	"port": m.port,
			//	"len":  len(uidata.Nodes),
			//	"tx":   key,
			//}).Debug("new tx")

			if _, ok := mapcount[key]; !ok {
				mapcount[key] = make(map[int]struct{})
			}
			mapcount[key][m.port] = struct{}{}
			if len(mapcount[key]) == count {
				logrus.WithField("key", key).Debug("fully announced")
				delete(mapcount, key)
				deleted += 1
			}

		}
	}
}
func manualQuery(port int, hash string) bool {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/transaction?hash=%s", port+PORT_OFFSET_RPC, hash))

	if err != nil {
		logrus.WithField("port", port).WithError(err).Warn("Request query error")
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
