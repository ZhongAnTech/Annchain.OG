package ogcore_test

import "github.com/sirupsen/logrus"

func setupLog() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
}
