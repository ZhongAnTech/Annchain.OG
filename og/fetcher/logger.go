package fetcher

import (
	"github.com/annchain/OG/mylog"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func InitLoggers(logger *logrus.Logger, logdir string) {
	log = mylog.InitLogger(logger, logdir, "og_fetcher.log")
	logrus.Debug("fetcher logger initialized.")
}
