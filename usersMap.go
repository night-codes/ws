package ws

import (
	"sync"
)

// *User map
type usersMap struct {
	sync.RWMutex
	connects map[interface{}]*User
}

func newUsersMap() *usersMap {
	return &usersMap{connects: make(map[interface{}]*User)}
}

func (m *usersMap) Set(key interface{}, val *User) {
	m.Lock()
	m.connects[key] = val
	m.Unlock()
}

func (m *usersMap) Delete(key interface{}) {
	m.Lock()
	delete(m.connects, key)
	m.Unlock()
}

func (m *usersMap) Get(key interface{}) *User {
	m.RLock()
	v, _ := m.connects[key]
	m.RUnlock()

	return v
}

func (m *usersMap) Len() int {
	m.RLock()
	n := len(m.connects)
	m.RUnlock()

	return n
}

func (m *usersMap) GetEx(key interface{}) (*User, bool) {
	m.RLock()
	v, exists := m.connects[key]
	m.RUnlock()
	return v, exists
}
