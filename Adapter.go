package ws

import (
	"encoding/json"
	"fmt"
	"strings"
)

type (
	// Adapter is slice of Connect instances
	Adapter struct {
		client     *Client
		command    string
		connection *Connection
		data       *[]byte
		requestID  int64
		sent       bool
		multiSend  bool
	}
	// NetContext is used network context, like *tokay.Context, *gin.Context, echo.Context etc.
	NetContext interface{}
)

// NewAdapter makes new *Adapter instance
func newAdapter(command string, connection *Connection, data *[]byte, requestID int64) *Adapter {
	return &Adapter{
		command:    command,
		connection: connection,
		data:       data,
		requestID:  requestID,
	}
}

// Data returns client message
func (a *Adapter) Data() []byte {
	return *a.data
}

// JSONData makes json.Unmarshal for client message
func (a *Adapter) JSONData(obj interface{}) error {
	return json.Unmarshal(*a.data, obj)
}

// StringData return string client message without json ""
func (a *Adapter) StringData() string {
	return strings.Trim(string(*a.data), "\"")
}

// Context returns copy of NetContext
func (a *Adapter) Context() NetContext {
	return a.connection.Context()
}

// RequestID returns adapter.requestID
func (a *Adapter) RequestID() int64 {
	return a.requestID
}

// Command returns request command
func (a *Adapter) Command() string {
	return a.command
}

// Send message to open connection
func (a *Adapter) Send(message interface{}) error {
	if a.sent && !a.multiSend{
		return fmt.Errorf("Adaper already sent")
	}
	a.sent = true
	if a.client != nil {
		return a.client.Send(a.command, message, a.requestID)
	}
	return a.connection.Send(a.command, message, a.requestID)
}

// Connection returns adapter connect instance
func (a *Adapter) Connection() *Connection {
	return a.connection
}

// Close all connects
func (a *Adapter) Close() {
	a.connection.Close()
}

// Subscribers of commands ("command1,command2" etc.)
func (a *Adapter) Subscribers(commands string) Connections {
	return a.connection.Subscribers(commands)
}

// User returns connection user
func (a *Adapter) User() *User {
	return a.connection.User()
}
