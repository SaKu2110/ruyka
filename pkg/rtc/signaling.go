package rtc

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

type EventType string

const (
	EventTypeOffer     EventType = "offer"
	EventTypeAnswer    EventType = "answer"
	EventTypeCandidate EventType = "candidate"
)

type Message interface{}

type SignalConnection interface {
	ReadMessage(Message) error
	WriteMessage(Message) error
}

type signalConnection struct {
	mux  sync.Mutex
	conn *websocket.Conn
}

func NewSignalConnection(c *websocket.Conn) SignalConnection {
	return &signalConnection{
		mux:  sync.Mutex{},
		conn: c,
	}
}

func (c *signalConnection) ReadMessage(msg Message) error {
	typ, raw, err := c.conn.ReadMessage()
	if err != nil {
		return err
	}

	switch typ {
	case websocket.TextMessage:
		return json.Unmarshal(raw, msg)
	default:
		return errors.New("invalid message type")
	}
}

func (c *signalConnection) WriteMessage(msg Message) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}
