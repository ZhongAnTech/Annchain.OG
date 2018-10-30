package downloader

import (
	"github.com/annchain/OG/mylog"
	"github.com/sirupsen/logrus"
)

var log   *logrus.Logger

func InitLoggers( logger *logrus.Logger, logdir string ) {
	log =  mylog.InitLogger(logger,logdir, "downloader.log",)
	logrus.Debug("message logger initialized.")
}
