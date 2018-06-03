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

func (m *usersMap) Copy() (c map[interface{}]*User) {
	c = make(map[interface{}]*User, len(m.connects))

	m.RLock()
	for k := range m.connects {
		c[k] = m.connects[k]
	}
	m.RUnlock()

	return c
}

func (m *usersMap) Set(key interface{}, val *User) {
	m.Lock()
	m.connects[key] = val
	m.Unlock()
}

func (m *usersMap) Clear() {
	m.Lock()
	m.connects = make(map[interface{}]*User)
	m.Unlock()
}

func (m *usersMap) Replace(newMap map[interface{}]*User) {
	m.Lock()
	m.connects = newMap
	m.Unlock()
}

func (m *usersMap) Delete(key interface{}) {
	m.Lock()
	delete(m.connects, key)
	m.Unlock()
}

func (m *usersMap) Get(key interface{}) *User {
	m.RLock()
	v := m.connects[key]
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
