package performance

import (
	"github.com/latifrons/soccerdash"
)

type SoccerdashReporter struct {
	Reporter *soccerdash.Reporter
}

func (s SoccerdashReporter) Report(key string, value interface{}) {
	s.Reporter.Report(key, value, false)
}
