package tendermint

import "time"

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
	currentRequest *WaiterRequest
	requestChannel chan *WaiterRequest
	contextChannel chan WaiterContext
	quit           chan bool
}

func NewWaiter() *Waiter {
	return &Waiter{
		requestChannel: make(chan *WaiterRequest, 100),
		contextChannel: make(chan WaiterContext, 100),
		quit:           make(chan bool),
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
			if w.currentRequest == nil || !r.Equal(w.currentRequest.Context){
				timer.Stop()
			}
		case <-timer.C:
			// timeout, trigger callback
			if w.currentRequest != nil {
				w.currentRequest.TimeoutCallback(w.currentRequest.Context)
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
