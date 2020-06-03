package performance

import "github.com/latifrons/soccerdash"

type SoccerdashReporter struct {
	Id         string
	IpPort     string
	BufferSize int
	reporter   *soccerdash.Reporter
}

func (s *SoccerdashReporter) InitDefault() {
	s.reporter = &soccerdash.Reporter{
		Id:            s.Id,
		TargetAddress: s.IpPort,
		BufferSize:    s.BufferSize,
	}
}

func (s SoccerdashReporter) Report(key string, value interface{}) {
	s.reporter.Report(key, value, false)
}
