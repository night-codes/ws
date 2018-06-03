package ws

import (
	"errors"
	"fmt"
)

// Connections is slice of Connect instances
type Connections struct {
	connMap *connMap
}

func newConnections() Connections {
	return Connections{connMap: newConnMap()}
}

// Send message to open connect
func (cs Connections) Send(command string, message interface{}) error {
	errStr := ""
	for _, connect := range cs.connMap.Copy() {
		if err := connect.Send(command, message); err != nil {
			errStr += fmt.Sprintf("Connect %d: %v/n", connect.ID(), err)
		}
	}
	if errStr != "" {
		return errors.New(errStr)
	}
	return nil
}

// Add connection to list
func (cs Connections) Add(c *Connection) {
	cs.connMap.Set(c.ID(), c)
}

// Close all connections
func (cs Connections) Close() {
	for _, connect := range cs.connMap.Copy() {
		connect.Close()
	}
}

// Connection by ID (or empty closed if not found)
func (cs Connections) Connection(connID uint64) *Connection {
	connection, ok := cs.connMap.GetEx(connID)
	if !ok {
		connection = emptyConnection()
	}
	return connection
}

// Subscribers of commands ("command1,command2" etc.)
func (cs Connections) Subscribers(commands string) Connections {
	ret := newConnections()
	for _, connection := range cs.connMap.Copy() {
		if connection.IsSubscribed(commands) {
			ret.Add(connection)
		}
	}
	return ret
}
