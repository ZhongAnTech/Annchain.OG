package syncer

import (
	"github.com/annchain/OG/mylog"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func InitLoggers(logger *logrus.Logger, logdir string) {
	log = mylog.InitLogger(logger, logdir, "og_syncer.log")
	logrus.Debug("syncer logger initialized.")
}
