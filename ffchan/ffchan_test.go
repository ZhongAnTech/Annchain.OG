package ffchan

import (
	"testing"
	"github.com/sirupsen/logrus"
	"time"
)

func TestChan(t *testing.T) {
	c := make(chan bool)

	go func() {
		time.Sleep(time.Second * 5)
		<-c
		time.Sleep(time.Second * 5)
		<-c
	}()

	logrus.Info("Sending 1st")
	<-NewTimeoutSender(c, true, "test1", 1000).C
	logrus.Info("Sending 2st")
	<-NewTimeoutSender(c, true, "test2", 1000).C

}
