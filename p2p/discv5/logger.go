package discv5

import (
	"github.com/sirupsen/logrus"
)

var log   *logrus.Logger

func SetLogger ( logger *logrus.Logger) {
	log = logger
}
