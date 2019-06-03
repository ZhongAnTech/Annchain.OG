package p2p

import (
	"github.com/annchain/OG/mylog"
	"github.com/annchain/OG/p2p/discover"
	"github.com/annchain/OG/p2p/discv5"
	"github.com/annchain/OG/p2p/onode"
	"github.com/sirupsen/logrus"
)

var log = logrus.StandardLogger()

func InitLoggers(logger *logrus.Logger, logdir string) {
	log = mylog.InitLogger(logger, logdir, "og_p2p")
	logrus.Debug("p2p logger initialized.")
	discover.SetLogger(log)
	discv5.SetLogger(log)
	onode.SetLogger(log)

}
