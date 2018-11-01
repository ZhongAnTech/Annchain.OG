package ffchan

import (
	"github.com/sirupsen/logrus"
	"testing"
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
	<-NewTimeoutSender(c, true).C
	logrus.Info("Sending 2st")
	<-NewTimeoutSender(c, true).C

}
