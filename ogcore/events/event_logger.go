package events

import (
	"github.com/annchain/OG/eventbus"
	"github.com/sirupsen/logrus"
)

type EventLogger struct {
}

func (e EventLogger) HandleEvent(ev eventbus.Event) {
	logrus.WithField("eventType", ev.GetEventType()).Debug("event received")
}
