package ws

import "sync"

type (
	requestsMap struct {
		sync.RWMutex
		fns map[int64]readFunc
	}
)

func newRequestsMap() *requestsMap {
	return &requestsMap{fns: make(map[int64]readFunc)}
}

func (m *requestsMap) Set(key int64, val readFunc) {
	m.Lock()
	defer m.Unlock()

	m.fns[key] = val
}
func (m *requestsMap) Delete(key int64) {
	m.Lock()
	delete(m.fns, key)
	m.Unlock()
}

func (m *requestsMap) Get(key int64) readFunc {
	m.RLock()
	v, _ := m.fns[key]
	m.RUnlock()

	return v
}

func (m *requestsMap) Len() int {
	m.RLock()
	n := len(m.fns)
	m.RUnlock()

	return n
}

func (m *requestsMap) GetEx(key int64) (readFunc, bool) {
	m.RLock()
	v, exists := m.fns[key]
	m.RUnlock()
	return v, exists
}
