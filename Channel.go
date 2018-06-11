package ws

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/night-codes/tokay"
	tokayWebsocket "github.com/night-codes/tokay-websocket"
	"github.com/valyala/fasthttp"
	"gopkg.in/night-codes/types.v1"
)

// Channel is websocket route
type (
	Channel struct {
		connMap  *connMap
		users    *usersMap
		subscrs  *subscrMap
		readers  *readersMap
		requests *requestsMap
		closeCh  chan bool
		closed   bool
	}

	messageStruct struct {
		Message []byte
		Err     error
	}

	connIface interface {
		SetReadLimit(limit int64)
		ReadMessage() (messageType int, p []byte, err error)
		WriteMessage(messageType int, data []byte) error
		Close() error
	}
)

var nextConnID uint64

func newChannel() *Channel {
	return &Channel{
		connMap:  newConnMap(),
		users:    newUsersMap(),
		subscrs:  newSubscrMap(),
		readers:  newReaderMap(),
		requests: newRequestsMap(),
		closeCh:  make(chan bool),
	}
}

// tokay websocket handler
func (channel *Channel) handlerTokay(c *tokay.Context) {
	if channel.closed {
		c.String(400, "Channel is closed.")
		return
	}

	conn := c.WSConn
	connection := newConnection(atomic.AddUint64(&nextConnID, 1), channel, conn, c.Copy())
	channel.readLoop(conn, connection)
	connection.Close()
}

// fasthttp websocket handler
func (channel *Channel) handlerFasthttp(ctx *fasthttp.RequestCtx, conn *tokayWebsocket.Conn) {
	if channel.closed {
		ctx.SetStatusCode(http.StatusBadRequest)
		fmt.Fprintf(ctx, "Channel is closed.")
		return
	}

	copyCtx := &fasthttp.RequestCtx{}
	ctx.Request.CopyTo(&copyCtx.Request)
	ctx.Response.CopyTo(&copyCtx.Response)

	connection := newConnection(atomic.AddUint64(&nextConnID, 1), channel, conn, copyCtx)
	channel.readLoop(conn, connection)
	connection.Close()
}

// gin websocket handler
func (channel *Channel) handlerGin(c *gin.Context, conn *websocket.Conn) {
	if channel.closed {
		c.String(400, "Channel is closed.")
		return
	}

	connection := newConnection(atomic.AddUint64(&nextConnID, 1), channel, conn, c.Copy())
	channel.readLoop(conn, connection)
	connection.Close()
}

// net websocket handler
func (channel *Channel) handlerNetHTTP(w http.ResponseWriter, r *http.Request, conn *websocket.Conn) {
	if channel.closed {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Channel is closed.\n")
		return
	}

	connection := newConnection(atomic.AddUint64(&nextConnID, 1), channel, conn, r)
	channel.readLoop(conn, connection)
	connection.Close()
}

func (channel *Channel) readLoop(conn connIface, connection *Connection) {
	for {
		resultCh := make(chan *messageStruct)

		go func() {
			_, message, err := conn.ReadMessage()
			resultCh <- &messageStruct{message, err}
		}()

		select {
		case result := <-resultCh:
			if result.Err != nil {
				return
			}
			go func(message []byte) {
				var result = bytes.SplitN(message, []byte(":"), 3)
				if len(result) == 3 {
					requestID := types.Int64(result[0])
					command := string(result[1])
					data := result[2]
					if requestID < 0 { // answer to the request from server
						if fns, ex := channel.requests.GetEx(requestID); ex && len(fns) > 0 {
							fns[0](newAdapter(command, connection, &data, requestID))
							channel.requests.Delete(requestID)
						}
					} else if fns, exists := channel.readers.GetEx(command); exists {
						adapter := newAdapter(command, connection, &data, requestID)
						for _, fn := range fns {
							fn(adapter)
						}
					}
				}
			}(result.Message)
		case channel.closed = <-channel.closeCh:
			return
		}
	}
}

// Read is client message (request) handler
func (channel *Channel) Read(command string, fn func(*Adapter)) {
	channel.readers.Set(command, fn)
}

// Close ws instance connections
func (channel *Channel) Close() {
	channel.closeCh <- true
	cs := channel.connMap.Copy()
	for _, v := range cs {
		v.Close()
	}
}

// User by id
func (channel *Channel) User(userID interface{}) *User {
	user, ok := channel.users.GetEx(userID)
	if !ok {
		user = newUser(userID)
	}
	return user
}

// Connection by ID (or empty closed if not found)
func (channel *Channel) Connection(connID uint64) *Connection {
	connection, ok := channel.connMap.GetEx(connID)
	if !ok {
		connection = emptyConnection()
	}
	return connection
}

// Send message to open connections
func (channel *Channel) Send(command string, message interface{}) error {
	errStr := ""
	for _, connection := range channel.connMap.Copy() {
		if err := connection.Send(command, message); err != nil {
			errStr += fmt.Sprintf("Connect %d: %v/n", connection.ID(), err)
		}
	}
	if errStr != "" {
		return errors.New(errStr)
	}
	return nil
}

// Subscribers of commands ("command1,command2" etc.)
func (channel *Channel) Subscribers(commands string) Connections {
	ret := newConnections()
	for _, v := range strings.Split(commands, ",") {
		if v, ok := channel.subscrs.GetEx(strings.TrimSpace(v)); ok {
			for _, connection := range v.Copy() {
				ret.Add(connection)
			}
		}
	}
	return ret
}

func (channel *Channel) subscribeReader() {
	channel.Read("subscribe", func(a *Adapter) {
		command := a.StringData()
		a.Connection().Subscribe(command)
	})
}
