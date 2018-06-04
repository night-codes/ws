package ws

import (
	"github.com/night-codes/tokay"
)

type (
	// Map is alias for map[string]interface{}
	Map map[string]interface{}
)

var (
	dev = true
)

// New makes new Channel
func New(path string, r *tokay.RouterGroup) (cnannel *Channel) {
	cnannel = &Channel{
		connMap: newConnMap(),
		users:   newUsersMap(),
		subscrs: newSubscrMap(),
		readers: newReaderMap(),
		closeCh: make(chan bool),
	}

	r.GET(path, func(c *tokay.Context) {
		c.Websocket(func() {
			cnannel.handler(c)
		})
	})
	cnannel.Read("subscribe", func(a *Adapter) {
		command := a.StringData()
		a.Connection().Subscribe(command)
	})
	return
}
