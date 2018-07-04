package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/gin-gonic/gin"
	"github.com/night-codes/tokay"
	"github.com/night-codes/tokay-websocket"
	"gopkg.in/night-codes/types.v1"
)

// Connection instance
// [Copying Connection by value is forbidden. Use pointer to Connection instead.]
type Connection struct {
	id              uint64
	user            *User
	conn            connIface
	closed          bool
	subscribes      map[string]bool
	subscribesMutex sync.RWMutex
	writeMutex      sync.RWMutex
	wsClient        bool
	channel         *Channel
	context         NetContext
	requestID       int64
	timeout         time.Duration
	origin          string
}

// NewConnection creates new *Connection instance
func newConnection(connID uint64, channel *Channel, conn connIface, context NetContext) *Connection {
	c := &Connection{
		id:         connID,
		channel:    channel,
		conn:       conn,
		context:    context,
		subscribes: make(map[string]bool),
		timeout:    time.Second * 30,
	}
	channel.connMap.Set(connID, c)

	switch cc := context.(type) {
	case *tokay.Context:
		c.origin = cc.GetHeader("Origin")
		c.wsClient = len(cc.GetHeader("ws-client")) > 0
		c.setUser(cc.Get("userID"))
	case *gin.Context:
		c.origin = cc.Request.Header.Get("Origin")
		c.wsClient = len(cc.Request.Header.Get("ws-client")) > 0
		user, _ := cc.Get("userID")
		c.setUser(user)
	case *fasthttp.RequestCtx:
		c.origin = string(cc.Request.Header.Peek("Origin"))
		c.wsClient = len(string(cc.Request.Header.Peek("ws-client"))) > 0
		c.setUser(cc.UserValue("userID"))
	case *http.Request:
		c.origin = cc.Header.Get("Origin")
		c.wsClient = len(cc.Header.Get("ws-client")) > 0
		c.setUser(cc.Context().Value("userID"))
		// Example:
		// import "net/http"
		// import "context"
		// ...
		// request.WithContext(context.WithValue(request.Context(), "UserID", 12345))
	}
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

// Origin of connection
func (c *Connection) Origin() string {
	return c.origin
}

// ID of Connection
func (c *Connection) ID() uint64 {
	return c.id
}

// Request information from client
func (c *Connection) Request(command string, message interface{}, timeout ...time.Duration) ([]byte, error) {
	requestID := atomic.AddInt64(&c.requestID, -1)
	resultCh := make(chan []byte)
	timeoutD := c.timeout

	if len(timeout) > 0 {
		timeoutD = timeout[0]
	}
	c.channel.requests.Set(requestID, func(a *Adapter) {
		resultCh <- a.Data()
	})

	if err := c.Send(command, message, 0, requestID); err != nil {
		c.channel.requests.Delete(requestID)
		return []byte{}, err
	}

	select {
	case result := <-resultCh:
		return result, nil
	case <-time.Tick(timeoutD):
		c.channel.requests.Delete(requestID)
		return []byte{}, fmt.Errorf("\"%s\" request timeout", command)
	}
}

// Context returns copy of NetContext
func (c *Connection) Context() NetContext {
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
		if c.user.connMap.Len() == 0 {
			c.channel.users.Delete(c.user.ID())
		}
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
func (c *Connection) Send(command string, message interface{}, requestID ...int64) error {
	if c.closed {
		return fmt.Errorf("Connection %d already clossed", c.ID())
	}

	if message != nil {
		var reqID int64
		var srvReqID int64
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
				strRequestID = types.String(srvReqID)
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

		wstype := websocket.TextMessage
		if c.channel.UseBinary {
			wstype = websocket.BinaryMessage
		}

		c.writeMutex.Lock()
		c.conn.WriteMessage(wstype, *msg)
		c.writeMutex.Unlock()
	}
	return nil
}
