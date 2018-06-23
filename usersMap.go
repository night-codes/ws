package ws

import (
	"sync"
)

// *User map
type usersMap struct {
	sync.RWMutex
	users map[interface{}]*User
}

func newUsersMap() *usersMap {
	return &usersMap{users: make(map[interface{}]*User)}
}

func (m *usersMap) Set(key interface{}, val *User) {
	m.Lock()
	m.users[key] = val
	m.Unlock()
}

func (m *usersMap) Delete(key interface{}) {
	m.Lock()
	delete(m.users, key)
	m.Unlock()
}

func (m *usersMap) Get(key interface{}) *User {
	m.RLock()
	v, _ := m.users[key]
	m.RUnlock()

	return v
}

func (m *usersMap) Len() int {
	m.RLock()
	n := len(m.users)
	m.RUnlock()

	return n
}

func (m *usersMap) GetEx(key interface{}) (*User, bool) {
	m.RLock()
	v, exists := m.users[key]
	m.RUnlock()
	return v, exists
}

func (m *usersMap) Copy() (c map[interface{}]*User) {
	c = make(map[interface{}]*User, len(m.users))

	m.RLock()
	for k := range m.users {
		c[k] = m.users[k]
	}
	m.RUnlock()

	return c
}
