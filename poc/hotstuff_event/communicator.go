package hotstuff_event

import "github.com/sirupsen/logrus"

type Hub struct {
	Channels map[int]chan *Msg
}

func (h *Hub) GetChannel(id int) chan *Msg {
	return h.Channels[id]
}

func (h *Hub) Send(msg *Msg, id int) {
	logrus.WithField("msg", msg).WithField("to", id).Info("sending")
	defer logrus.WithField("msg", msg).WithField("to", id).Info("sent")
	h.Channels[id] <- msg
}

func (h *Hub) SendToAllButMe(msg *Msg, myId int) {
	for id := range h.Channels {
		if id != myId {
			h.Send(msg, id)
		}
	}
}
