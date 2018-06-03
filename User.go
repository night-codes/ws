package ws

import (
	"errors"
	"fmt"
)

// User instance
type User struct {
	connMap *connMap
	id      interface{}
}

// newUser creates new *User instance
func newUser(userID interface{}) *User {
	return &User{connMap: newConnMap(), id: userID}
}

// ID in users list
func (u *User) ID() interface{} {
	return u.id
}

// Send message to open user's connections
func (u *User) Send(command string, message interface{}) error {
	errStr := ""
	for _, connection := range u.connMap.Copy() {
		if err := connection.Send(command, message); err != nil {
			errStr += fmt.Sprintf("Connect %d: %v/n", connection.ID(), err)
		}
	}
	if errStr != "" {
		return errors.New(errStr)
	}
	return nil
}

// Connection by ID (or empty closed if not found)
func (u *User) Connection(connID uint64) *Connection {
	connection, ok := u.connMap.GetEx(connID)
	if !ok {
		connection = emptyConnection()
	}
	return connection
}

// Subscribers  of commands ("command1,command2" etc.)
func (u *User) Subscribers(commands string, message interface{}) Connections {
	ret := newConnections()
	for _, connection := range u.connMap.Copy() {
		if connection.IsSubscribed(commands) {
			ret.Add(connection)
		}
	}
	return ret
}

// Close User's connections
func (u *User) Close() {
	cs := u.connMap.Copy()
	for _, v := range cs {
		v.Close()
	}
}
