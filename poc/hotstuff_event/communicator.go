package hotstuff_event

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

type Hub struct {
	Channels map[int]chan *Msg
}

func (h *Hub) GetChannel(id int) chan *Msg {
	return h.Channels[id]
}

func (h *Hub) Send(msg *Msg, id int, pos string) {
	logrus.WithField("message", msg).WithField("to", id).Trace(fmt.Sprintf("[%d] sending [%s] to [%d]", msg.SenderId, pos, id))
	//defer logrus.WithField("msg", msg).WithField("to", id).Info("sent")
	h.Channels[id] <- msg
}

func (h *Hub) SendToAllButMe(msg *Msg, myId int, pos string) {
	for id := range h.Channels {
		if id != myId {
			h.Send(msg, id, pos)
		}
	}
}
func (h *Hub) SendToAllIncludingMe(msg *Msg, myId int, pos string) {
	for id := range h.Channels {
		h.Send(msg, id, pos)
	}
}
