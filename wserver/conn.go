package wserver

import (
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Conn wraps websocket.Conn with Conn. It defines to listen and read
// data from Conn.
type Conn struct {
	Conn *websocket.Conn

	AfterReadFunc   func(messageType int, r io.Reader)
	BeforeCloseFunc func()

	once   sync.Once
	id     string
	stopCh chan struct{}
}

// Write write p to the websocket connection. The error returned will always
// be nil if success.
func (c *Conn) Write(p []byte) (n int, err error) {
	select {
	case <-c.stopCh:
		return 0, errors.New("Conn is closed, can't be written")
	default:
		err = c.Conn.WriteMessage(websocket.TextMessage, p)
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}
}

// GetID returns the id generated using UUID algorithm.
func (c *Conn) GetID() string {
	c.once.Do(func() {
		u := uuid.New()
		c.id = u.String()
	})

	return c.id
}

// Listen listens for receive data from websocket connection. It blocks
// until websocket connection is closed.
func (c *Conn) Listen() {
	c.Conn.SetCloseHandler(func(code int, text string) error {
		if c.BeforeCloseFunc != nil {
			c.BeforeCloseFunc()
		}

		if err := c.Close(); err != nil {
			log.Println(err)
		}

		message := websocket.FormatCloseMessage(code, "")
		c.Conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		return nil
	})

	// Keeps reading from Conn util get error.
ReadLoop:
	for {
		select {
		case <-c.stopCh:
			break ReadLoop
		default:
			messageType, r, err := c.Conn.NextReader()
			if err != nil {
				// TODO: handle read error maybe
				break ReadLoop
			}

			if c.AfterReadFunc != nil {
				c.AfterReadFunc(messageType, r)
			}
		}
	}
}

// Close close the connection.
func (c *Conn) Close() error {
	select {
	case <-c.stopCh:
		return errors.New("Conn already been closed")
	default:
		c.Conn.Close()
		close(c.stopCh)
		return nil
	}
}

// NewConn wraps conn.
func NewConn(conn *websocket.Conn) *Conn {
	return &Conn{
		Conn:   conn,
		stopCh: make(chan struct{}),
	}
}

const (
	EVENT_NEW_UNIT = "new_unit"
)

// event2Cons contains a map of map
// key: event type
// value: another map whose key: Conn's ID ,value: Conn
type event2Cons struct {
	conns map[string]map[string]*Conn
	mu    sync.RWMutex
}

func (e *event2Cons) getFromMap(key string) (v map[string]*Conn, ok bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	v, ok = e.conns[key]
	return
}

func (e *event2Cons) setMap(key string, v map[string]*Conn) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.conns[key] = v
}

func NewEvent2Cons() *event2Cons {
	return &event2Cons{
		conns: make(map[string]map[string]*Conn),
	}
}
func (e *event2Cons) Add(eventType string, conn *Conn) error {
	conns, ok := e.getFromMap(eventType)
	if !ok {
		conns = make(map[string]*Conn)
		e.setMap(eventType, conns)
	}
	thisID := conn.GetID()
	if _, ok := e.getFromMap(thisID); !ok {
		//not exist,add it
		v, _ := e.getFromMap(eventType)
		v[thisID] = conn
		e.setMap(eventType, v)
	} else {
		return fmt.Errorf("Conn with ID: %s already exist!", thisID)
	}
	return nil
}

func (e *event2Cons) Remove(eventType string, conn *Conn) error {
	conns, ok := e.getFromMap(eventType)
	if !ok {
		return fmt.Errorf("No Connection with eventType: %s\n", eventType)
	}
	thisID := conn.GetID()
	if _, ok := e.getFromMap(thisID); !ok {
		return fmt.Errorf("No connection with ID: %s\n", thisID)
	} else {
		delete(conns, thisID)
	}
	return nil
}

func (e *event2Cons) Get(eventType string) ([]*Conn, error) {
	conns, ok := e.getFromMap(eventType)
	if !ok {
		return nil, fmt.Errorf("No Connection with eventType: %s\n", eventType)
	}
	var ret []*Conn
	for _, c := range conns {
		ret = append(ret, c)
	}
	return ret, nil
}

func (e *event2Cons) GetWithID(eventType string, ID string) (*Conn, error) {
	conns, err := e.Get(eventType)
	if err != nil {
		return nil, err
	}
	for _, c := range conns {
		if c.GetID() == ID {
			return c, nil
		}
	}
	return nil, fmt.Errorf("No Connection with eventType: %s ID: %s\n", eventType, ID)
}
