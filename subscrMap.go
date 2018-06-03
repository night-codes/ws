package ws

import (
	"sync"
)

type (
	// *connMap map
	subscrMap struct {
		sync.RWMutex
		subscr map[string]*connMap
	}
)

func newSubscrMap() *subscrMap {
	return &subscrMap{subscr: make(map[string]*connMap)}
}

func (m *subscrMap) Copy() (c map[string]*connMap) {
	c = make(map[string]*connMap, len(m.subscr))

	m.RLock()
	for k := range m.subscr {
		c[k] = m.subscr[k]
	}
	m.RUnlock()

	return c
}

func (m *subscrMap) Set(key string, val *connMap) {
	m.Lock()
	m.subscr[key] = val
	m.Unlock()
}

func (m *subscrMap) Clear() {
	m.Lock()
	m.subscr = make(map[string]*connMap)
	m.Unlock()
}

func (m *subscrMap) Replace(newMap map[string]*connMap) {
	m.Lock()
	m.subscr = newMap
	m.Unlock()
}

func (m *subscrMap) Delete(key string) {
	m.Lock()
	delete(m.subscr, key)
	m.Unlock()
}

func (m *subscrMap) Get(key string) *connMap {
	m.RLock()
	v := m.subscr[key]
	m.RUnlock()

	return v
}

func (m *subscrMap) Len() int {
	m.RLock()
	n := len(m.subscr)
	m.RUnlock()

	return n
}

func (m *subscrMap) GetEx(key string) (*connMap, bool) {
	m.RLock()
	v, exists := m.subscr[key]
	m.RUnlock()
	return v, exists
}
