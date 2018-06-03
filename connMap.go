package ws

import (
	"sync"
)

// *Connections map
type connMap struct {
	sync.RWMutex
	connects map[uint64]*Connection
}

func newConnMap() *connMap {
	return &connMap{connects: make(map[uint64]*Connection)}
}

func (m *connMap) Copy() (c map[uint64]*Connection) {
	c = make(map[uint64]*Connection, len(m.connects))

	m.RLock()
	for k := range m.connects {
		c[k] = m.connects[k]
	}
	m.RUnlock()

	return c
}

func (m *connMap) Set(key uint64, val *Connection) {
	m.Lock()
	m.connects[key] = val
	m.Unlock()
}

func (m *connMap) Clear() {
	m.Lock()
	m.connects = make(map[uint64]*Connection)
	m.Unlock()
}

func (m *connMap) Replace(newMap map[uint64]*Connection) {
	m.Lock()
	m.connects = newMap
	m.Unlock()
}

func (m *connMap) Delete(key uint64) {
	m.Lock()
	delete(m.connects, key)
	m.Unlock()
}

func (m *connMap) Get(key uint64) *Connection {
	m.RLock()
	v := m.connects[key]
	m.RUnlock()

	return v
}

func (m *connMap) Len() int {
	m.RLock()
	n := len(m.connects)
	m.RUnlock()

	return n
}

func (m *connMap) GetEx(key uint64) (*Connection, bool) {
	m.RLock()
	v, exists := m.connects[key]
	m.RUnlock()
	return v, exists
}
