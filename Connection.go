package ws

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"gopkg.in/night-codes/types.v1"

	"github.com/night-codes/tokay"
	"github.com/night-codes/tokay-websocket"
)

// Connection instance
// [Copying Connection by value is forbidden. Use pointer to Connection instead.]
type Connection struct {
	id              uint64
	user            *User
	conn            *websocket.Conn
	closed          bool
	subscribes      map[string]bool
	subscribesMutex sync.RWMutex
	writeMutex      sync.RWMutex
	wsClient        bool
	channel         *Channel
	context         *tokay.Context
	requestID       uint64
}

// NewConnection creates new *Connection instance
func newConnection(connID uint64, channel *Channel, conn *websocket.Conn, context *tokay.Context) *Connection {
	c := &Connection{
		id:         connID,
		channel:    channel,
		conn:       conn,
		wsClient:   len(context.GetHeader("ws-client")) > 0,
		context:    context,
		subscribes: make(map[string]bool),
	}
	channel.connMap.Set(connID, c)
	c.setUser(context.Get("UserID"))
	return c
}

// NewConnection creates new *Connection instance
func emptyConnection() *Connection {
	return &Connection{
		closed: true,
	}
}

// User returns connection user
func (c *Connection) User() *User {
	return c.user
}

// ID of Connection
func (c *Connection) ID() uint64 {
	return c.id
}

// Request send message to open connect and wait for answer
func (c *Connection) Request(command string, message interface{}) ([]byte, error) {
	c.Send(command, message, 0, atomic.AddUint64(&c.requestID, 1))
	return []byte{}, nil
}

// Context returns copy of *tokay.Context
func (c *Connection) Context() *tokay.Context {
	return c.context
}

// Subscribers returns Connects Subscribers of commands ("command1,command2" etc.)
func (c *Connection) Subscribers(commands string) Connections {
	conns := newConnections()
	if !c.closed && c.IsSubscribed(commands) {
		conns.Add(c)
	}
	return conns
}

// IsSubscribed returns true if Connect subscribed for one of commands ("command1,command2" etc.)
func (c *Connection) IsSubscribed(commands string) bool {
	if !c.closed {
		c.subscribesMutex.RLock()
		defer c.subscribesMutex.RUnlock()

		for _, v := range strings.Split(commands, ",") {
			if _, ok := c.subscribes[strings.TrimSpace(v)]; ok {
				return true
			}
		}
	}
	return false
}

// Close connect
func (c *Connection) Close() {
	if !c.closed {
		c.closed = true
		c.conn.Close()

		c.writeMutex.Lock()
		c.conn.WriteMessage(websocket.CloseMessage, nil)
		c.writeMutex.Unlock()

		c.user.connMap.Delete(c.ID())
		c.channel.connMap.Delete(c.ID())
	}
}

func (c *Connection) setUser(userID interface{}) {
	if !c.closed {
		user, ok := c.channel.users.GetEx(userID)
		if !ok {
			user = newUser(userID)
			c.channel.users.Set(userID, user)
		}

		user.connMap.Set(c.ID(), c)
		c.user = user
	}
}

// Subscribe connection to command
func (c *Connection) Subscribe(command string) {
	if !c.closed {
		subscribes, ok := c.channel.subscrs.GetEx(command)
		if !ok {
			subscribes = newConnMap()
			c.channel.subscrs.Set(command, subscribes)
		}

		subscribes.Set(c.ID(), c)
		c.subscribesMutex.Lock()
		defer c.subscribesMutex.Unlock()
		c.subscribes[strings.TrimSpace(command)] = true
	}
}

// Send message to open connect
func (c *Connection) Send(command string, message interface{}, requestID ...uint64) error {
	if c.closed {
		return fmt.Errorf("Connection %d already clossed", c.ID())
	}

	if message != nil {
		var reqID uint64
		var srvReqID uint64
		if len(requestID) > 0 {
			reqID = requestID[0]
			if len(requestID) == 2 {
				srvReqID = requestID[1]
			}
		}

		msg := &[]byte{}
		var err error

		if c.wsClient {
			if m, ok := message.([]byte); ok {
				msg = &m
			} else if m, ok := message.(*[]byte); ok {
				msg = m
			} else {
				var m []byte
				m, err = json.Marshal(message)
				msg = &m
			}

			strRequestID := types.String(reqID)
			if srvReqID != 0 {
				strRequestID = "-" + types.String(srvReqID)
			}
			*msg = append([]byte(strRequestID+":"+command+":"), *msg...)
		} else {
			var m []byte
			m, err = json.Marshal(Map{
				"command":      command,
				"requestID":    reqID,
				"srvRequestID": srvReqID,
				"data":         message,
			})
			msg = &m
		}

		if err != nil {
			return fmt.Errorf("WS: Connection.Send: json.Marshal: %v", err)
		}

		wstype := websocket.BinaryMessage
		if dev {
			wstype = websocket.TextMessage
		}

		c.writeMutex.Lock()
		c.conn.WriteMessage(wstype, *msg)
		c.writeMutex.Unlock()
	}
	return nil
}
