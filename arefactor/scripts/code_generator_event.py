t = """
type NewLocalHeightUpdatedEvent struct{
	Height int64
}

// event generator

// member
newLocalHeightUpdatedEventSubscribers []og_interface.NewLocalHeightUpdatedEventSubscriber


func (d *IntArrayLedger) AddSubscriberNewLocalHeightUpdatedEvent(sub og_interface.NewLocalHeightUpdatedEventSubscriber) {
	d.newLocalHeightUpdatedEventSubscribers = append(d.newLocalHeightUpdatedEventSubscribers, sub)
}

func (n *IntArrayLedger) notifyNewLocalHeightUpdatedEvent(event og_interface.NewLocalHeightUpdatedEvent) {
	for _, subscriber := range n.newLocalHeightUpdatedEventSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewLocalHeightUpdatedEventChannel(), event, "notifyNewLocalHeightUpdatedEvent").C
		//subscriber.NewLocalHeightUpdatedEventChannel() <- event
	}
}

// subscriber interface
type NewLocalHeightUpdatedEventSubscriber interface{
	NewLocalHeightUpdatedChannel() chan *NewLocalHeightUpdatedEvent
}


// member of subscriber

newLocalHeightUpdatedEventChan chan *og_interface.NewLocalHeightUpdatedEvent

// init default
b.newLocalHeightUpdatedEventChan = make(chan *og_interface.NewLocalHeightUpdatedEvent)

func (b *IntLedgerSyncer) NewLocalHeightUpdatedChannel() chan *og_interface.NewLocalHeightUpdatedEvent {
	return b.newLocalHeightUpdatedEventChan
}


"""


if __name__ == '__main__':
    target = "NewHeightDetected"
    target_camel = str.lower(target[0]) + target[1:]
    t = t.replace("NewLocalHeightUpdated", target)
    t = t.replace("newLocalHeightUpdated", target_camel)
    print(t)