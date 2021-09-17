package connector

import (
	"net/http"

	"github.com/night-codes/tokay"
	tokayWebsocket "github.com/night-codes/tokay-websocket"
	"github.com/night-codes/ws"
	"github.com/valyala/fasthttp"
)

var (
	// CheckOrigin wsUpgrader function
	CheckOrigin func(request interface{}) bool
)

// New makes new Channel with "github.com/night-codes/tokay"
func New(bufferSizes ...int) (tokay.Handler, *ws.Channel) {
	channel := ws.NewChannel()
	wsupgrader := getFastUpgrader(bufferSizes...)

	return func(c *tokay.Context) {
		cc := c.Copy()
		wsupgrader.Receiver = func(conn *tokayWebsocket.Conn) {
			cc.WSConn = conn
			channel.Handler(conn, cc)
		}
		if err := wsupgrader.Upgrade(c.RequestCtx); err != nil {
			c.String(http.StatusBadRequest, "Failed to set websocket upgrade.")
		}
	}, channel
}

func getFastUpgrader(bufferSizes ...int) *tokayWebsocket.Upgrader {
	if len(bufferSizes) == 0 {
		bufferSizes = append(bufferSizes, 4096, 4096)
	} else if len(bufferSizes) == 1 {
		bufferSizes = append(bufferSizes, bufferSizes[0])
	}
	socket := &tokayWebsocket.Upgrader{
		ReadBufferSize:  bufferSizes[0],
		WriteBufferSize: bufferSizes[1],
	}
	if CheckOrigin != nil {
		socket.CheckOrigin = func(r *fasthttp.RequestCtx) bool {
			return CheckOrigin(r)
		}
	}
	return socket
}
