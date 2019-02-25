package tendermint

import (
	"time"
	"github.com/annchain/OG/ffchan"
)

type WaiterContext interface {
	Equal(WaiterContext) bool
	Newer(WaiterContext) bool
}

type WaiterRequest struct {
	WaitTime        time.Duration
	Context         WaiterContext
	TimeoutCallback func(WaiterContext)
}

// Waiter provides a way to wait for some context to be changed in a certain time.
// If the context is not changed, callback function will be triggered.
type Waiter struct {
	currentRequest      *WaiterRequest
	requestChannel      chan *WaiterRequest
	contextChannel      chan WaiterContext
	callbackEventChanel chan *WaiterRequest
	quit                chan bool
}

func NewWaiter(callbackEventChannel chan *WaiterRequest) *Waiter {
	return &Waiter{
		requestChannel:      make(chan *WaiterRequest, 100),
		contextChannel:      make(chan WaiterContext, 100),
		quit:                make(chan bool),
		callbackEventChanel: callbackEventChannel,
	}
}

func (w *Waiter) StartEventLoop() {
	timer := time.NewTimer(time.Duration(10))
	for {
		select {
		case <-w.quit:
			break
		case r := <-w.requestChannel:
			// could be an updated request
			// if it is really updated request,
			if w.currentRequest != nil && !r.Context.Newer(w.currentRequest.Context) {
				continue
			}
			w.currentRequest = r
			timer.Reset(r.WaitTime)
		case r := <-w.contextChannel:
			if w.currentRequest == nil || !r.Equal(w.currentRequest.Context) {
				timer.Stop()
			}
		case <-timer.C:
			// timeout, trigger callback
			if w.currentRequest != nil {
				ffchan.NewTimeoutSenderShort(w.callbackEventChanel, w.currentRequest, "waiterCallback")
				//w.currentRequest.TimeoutCallback(w.currentRequest.Context)
			}

		}
	}
}

func (w *Waiter) UpdateRequest(req *WaiterRequest) {
	w.requestChannel <- req
}

func (w *Waiter) UpdateContext(context WaiterContext) {
	w.contextChannel <- context
}
