package onode

import (
	"github.com/sirupsen/logrus"
)

var log = logrus.StandardLogger()

func SetLogger(logger *logrus.Logger) {
	log = logger
}
