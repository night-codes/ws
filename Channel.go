package ws

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/night-codes/tokay"
	"gopkg.in/night-codes/types.v1"
)

// Channel is websocket route
type Channel struct {
	connMap *connMap
	users   *usersMap
	subscrs *subscrMap
	readers *readersMap
	closeCh chan bool
	closed  bool
}

type messageStruct struct {
	MessageType int
	Message     []byte
	Err         error
}

var nextConnID uint64

// websocket handler
func (channel *Channel) handler(c *tokay.Context) {
	if channel.closed {
		c.String(400, "Channel is closed. Wait.")
		return
	}

	conn := c.WSConn
	connection := newConnection(atomic.AddUint64(&nextConnID, 1), channel, conn, c.Copy())

ReadLoop:
	for {
		resultCh := make(chan *messageStruct)

		go func() {
			messageType, message, err := conn.ReadMessage()
			resultCh <- &messageStruct{messageType, message, err}
		}()

		select {
		case result := <-resultCh:
			if result.Err != nil {
				break ReadLoop
			}
			go func(message []byte) {
				var result = bytes.SplitN(message, []byte(":"), 3)
				if len(result) == 3 {
					requestID := types.Uint64(result[0])
					command := string(result[1])
					data := result[2]
					if fn, exists := channel.readers.GetEx(command); exists {
						fn(newAdapter(command, connection, &data, requestID))
					}
				}
				return
			}(result.Message)
		case channel.closed = <-channel.closeCh:
			break ReadLoop
		}
	}

	connection.Close()
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

// UserByID returns User by id
func (channel *Channel) UserByID(userID interface{}) *User {
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
