package hotstuff_event

import (
	"fmt"
	"github.com/latifrons/logrus-logstash-hook"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
	"time"
)

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
	logger := logrus.New()
	logger.Hooks.Add(AddPeerLogHook{Id: strconv.Itoa(id)})
	// for socket debugging
	conn, err := net.DialTimeout("tcp", "127.0.0.1:1088", time.Millisecond*50)
	if err != nil {
		logrus.Warn("socket logger is not enabled")
	}
	hook := logrustash.New(conn, logrustash.DefaultFormatter(logrus.Fields{}))

	logger.Hooks.Add(hook)

	logger.SetLevel(logrus.TraceLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "15:04:05.000000",
	})

	return logger
}
