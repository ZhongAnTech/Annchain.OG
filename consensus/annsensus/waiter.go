package annsensus

import (
	"github.com/annchain/OG/ffchan"
	"github.com/sirupsen/logrus"
	"time"
)

type WaiterContext interface {
	Equal(WaiterContext) bool
	IsAfter(WaiterContext) bool
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
		requestChannel:      make(chan *WaiterRequest, 10),
		contextChannel:      make(chan WaiterContext, 10),
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
		case request := <-w.requestChannel:
			// could be an updated request
			// if it is really updated request,
			if w.currentRequest != nil && !request.Context.IsAfter(w.currentRequest.Context) {
				// this request is before current waiting request, ignore.
				continue
			}
			logrus.Trace("request is newer and we will reset")
			w.currentRequest = request
			if !timer.Stop() {
				// drain the timer but do not use the method in document
				// timer may already be consumed so use a select
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(request.WaitTime)
		case latestContext := <-w.contextChannel:
			if w.currentRequest == nil || latestContext.IsAfter(w.currentRequest.Context) {
				// a new state is updated, cancel all pending timeouts
				if w.currentRequest != nil {
					logrus.WithField("new", latestContext.(*TendermintContext).StepType).
						WithField("old", w.currentRequest.Context.(*TendermintContext).StepType).
						Debug("new state updated")
				} else {
					logrus.WithField("new", latestContext.(*TendermintContext).StepType).
						WithField("old", nil).
						Debug("new state updated")
				}
				if !timer.Stop() {
					// drain the timer but do not use the method in document
					// timer may already be consumed so use a select
					select {
					case <-timer.C:
					default:
					}
				}
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
	//ffchan.NewTimeoutSenderShort(w.requestChannel, req, "waiterrequest")
}

func (w *Waiter) UpdateContext(context WaiterContext) {
	w.contextChannel <- context
	//ffchan.NewTimeoutSenderShort(w.contextChannel, context, "waitercontext")
}
