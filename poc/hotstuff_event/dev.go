package hotstuff_event

import (
	"fmt"
	"github.com/latifrons/logrus-logstash-hook"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
	"time"
)

var vpeers []string

// All stucts that need call InitDefault() before being used
type NeedInitDefault interface {
	InitDefault()
}

type NodeLogger struct {
	Logger *logrus.Logger
}

type AddPeerLogHook struct {
	Id string
}

func (a AddPeerLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (a AddPeerLogHook) Fire(e *logrus.Entry) error {
	e.Message = fmt.Sprintf("[%s] ", a.Id) + e.Message
	return nil
}

func SetupOrderedLog(id int) *logrus.Logger {
	//logger := logrus.New()
	logger := logrus.StandardLogger()
	logger.Hooks.Add(AddPeerLogHook{Id: strconv.Itoa(id)})
	// for socket debugging
	conn, err := net.DialTimeout("tcp", "127.0.0.1:1088", time.Millisecond*50)
	if err != nil {
		logrus.Warn("socket logger is not enabled")
	} else {
		hook := logrustash.New(conn, logrustash.DefaultFormatter(logrus.Fields{}))

		logger.Hooks.Add(hook)
	}

	logger.SetLevel(logrus.WarnLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "15:04:05.000000",
	})

	return logger
}

func SetPeers(peers []string) {
	vpeers = peers
}

func Indexof(peerId string) int {
	for i, v := range vpeers {
		if v == peerId {
			return i
		}
	}
	return -1
}

func PrettyId(peerId string) string {
	return fmt.Sprintf("#%d", Indexof(peerId))
}

func PrettyIds(peerId []string) string {
	s := make([]string, len(peerId))
	for i, v := range peerId {
		s[i] = PrettyId(v)
	}
	return strings.Join(s, ",")
}
