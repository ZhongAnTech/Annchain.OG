package performance

import (
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
)

type SoccerdashReporter struct {
	Id         string
	IpPort     string
	BufferSize int
	Logger     *logrus.Logger
	reporter   *soccerdash.Reporter
}

func (s *SoccerdashReporter) InitDefault() {
	s.reporter = &soccerdash.Reporter{
		Id:            s.Id,
		TargetAddress: s.IpPort,
		BufferSize:    s.BufferSize,
		Logger:        s.Logger,
	}
}

func (s SoccerdashReporter) Report(key string, value interface{}) {
	s.reporter.Report(key, value, false)
}
