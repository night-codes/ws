package ws

import "sync"

type (
	readFunc   func(*Adapter)
	readersMap struct {
		sync.RWMutex
		fns map[string][]readFunc
	}
)

func newReaderMap() *readersMap {
	return &readersMap{fns: make(map[string][]readFunc)}
}

func (m *readersMap) Copy() (c map[string][]readFunc) {
	c = make(map[string][]readFunc, len(m.fns))

	m.RLock()
	for k := range m.fns {
		c[k] = m.fns[k]
	}
	m.RUnlock()

	return c
}

func (m *readersMap) Set(key string, val readFunc) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.fns[key]; ok {
		m.fns[key] = append(m.fns[key], val)
		return
	}
	m.fns[key] = []readFunc{val}
}

func (m *readersMap) Clear() {
	m.Lock()
	m.fns = make(map[string][]readFunc)
	m.Unlock()
}

func (m *readersMap) Replace(newMap map[string][]readFunc) {
	m.Lock()
	m.fns = newMap
	m.Unlock()
}

func (m *readersMap) Delete(key string) {
	m.Lock()
	delete(m.fns, key)
	m.Unlock()
}

func (m *readersMap) Get(key string) []readFunc {
	m.RLock()
	v := m.fns[key]
	m.RUnlock()

	return v
}

func (m *readersMap) Len() int {
	m.RLock()
	n := len(m.fns)
	m.RUnlock()

	return n
}

func (m *readersMap) GetEx(key string) ([]readFunc, bool) {
	m.RLock()
	v, exists := m.fns[key]
	m.RUnlock()
	return v, exists
}
