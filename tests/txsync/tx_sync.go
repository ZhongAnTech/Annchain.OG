package main

import (
	"net/url"
	"github.com/sirupsen/logrus"
	"fmt"
	"github.com/gorilla/websocket"
	"encoding/json"
	"github.com/annchain/OG/wserver"
	"time"
)

type pm struct {
	message string
	port    int
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	agg := make(chan pm, 1000)
	count := 0

	for port := 8002; port <= 8002 + 10 * 100; port += 100 {
		err := startListen(agg, port)
		if err != nil {
			continue
		}
		count += 1
	}
	mapcount := make(map[string][]int)

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
				mapcount[key] = []int{}
			}
			mapcount[key] = append(mapcount[key], m.port)
			if len(mapcount[key]) == count {
				logrus.WithField("key", key).Debug("fully announced")
				delete(mapcount, key)
				deleted += 1
				for k, v := range mapcount {
					logrus.WithField("key", k).WithField("value", v).Warn("Still not fulfilled")
				}
			}
		case <-ticker.C:
			logrus.WithField("v", float64(deleted)/10).Info("tps")
			deleted = 0
		}

	}
}
func startListen(agg chan pm, port int) error {
	u := url.URL{Scheme: "ws", Host: fmt.Sprintf("127.0.0.1:%d", port), Path: "/ws"}
	logrus.Infof("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logrus.WithError(err).Errorf("dial")
		return err
	}

	go func() {
		err := c.WriteMessage(websocket.TextMessage, []byte("{\"event\": \"new_unit\"}"))
		if err != nil {
			logrus.WithField("port", port).Error("write")
			return
		}

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logrus.WithError(err).WithField("port", port).Error("read error")
				return
			}
			agg <- pm{message: string(message), port: port}
		}
	}()
	return nil
}
