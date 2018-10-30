package og

import (
	"github.com/annchain/OG/mylog"
	"github.com/sirupsen/logrus"
)

var msgLog   *logrus.Logger

func InitLoggers( logger *logrus.Logger,logdir string ) {
	msgLog =  mylog.InitLogger(logger,logdir, "msg.log",)
	logrus.Debug("message logger initialized.")
}
