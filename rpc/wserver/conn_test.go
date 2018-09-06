package wserver

import (
	"testing"
	"github.com/gorilla/websocket"
	"fmt"
)

func TestEvent2Conns(t *testing.T) {
	e2c := NewEvent2Cons()
	for i := 0; i < 10; i++ {
		con := NewConn(&websocket.Conn{})
		e2c.Add(EVENT_NEW_UNIT, con)
	}
	conns, err := e2c.Get(EVENT_NEW_UNIT)
	firstLen := len(conns)
	if err != nil {
		t.Errorf("Get EVENT_NEW_UNIT error: %s\n", err)
	} else {
		show(conns)
	}

	e2c.Remove(EVENT_NEW_UNIT, conns[0])
	conns, _ = e2c.Get(EVENT_NEW_UNIT)
	secondLen := len(conns)
	if firstLen == secondLen {
		t.Errorf("Remove error\n")
	} else {
		fmt.Printf("Remove successfully\n")
	}
	show(conns)

	c, err := e2c.GetWithID(EVENT_NEW_UNIT, conns[0].GetID())
	if err != nil {
		t.Errorf("%s\n", err)
	} else {
		if c.GetID() == conns[0].GetID() {
			fmt.Println("Got conn with ID OK")
		}
	}

}

func show(conns []*Conn) {
	for i, c := range conns {
		fmt.Printf("%d-th conn ID: %s\n", i, c.GetID())
	}
}
