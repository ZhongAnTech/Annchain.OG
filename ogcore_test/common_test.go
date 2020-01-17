package ogcore_test

import (
	"github.com/sirupsen/logrus"
)

func setupLog() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
		//TimestampFormat: "15:04:05.000000",
	})
}
