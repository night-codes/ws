package connector

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/night-codes/ws"
)

var (
	// CheckOrigin wsUpgrader function
	CheckOrigin func(request interface{}) bool
)

// New makes new Channel with "github.com/gin-gonic/gin".RouterGroup
func New(path string, r *gin.RouterGroup, bufferSizes ...int) *ws.Channel {
	channel := ws.NewChannel()
	wsupgrader := getWsupgrader(bufferSizes...)
	r.GET(path, func(c *gin.Context) {
		cc := c.Copy()
		if conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil); err == nil {
			channel.Handler(conn, cc)
		} else {
			c.String(http.StatusBadRequest, "Failed to set websocket upgrade.")
		}
	})
	return channel
}

// getWsupgrader create new *websocket.Upgrader
func getWsupgrader(bufferSizes ...int) *websocket.Upgrader {
	if len(bufferSizes) == 0 {
		bufferSizes = append(bufferSizes, 4096, 4096)
	} else if len(bufferSizes) == 1 {
		bufferSizes = append(bufferSizes, bufferSizes[0])
	}
	socket := &websocket.Upgrader{
		ReadBufferSize:  bufferSizes[0],
		WriteBufferSize: bufferSizes[1],
	}
	if CheckOrigin != nil {
		socket.CheckOrigin = func(r *http.Request) bool {
			return CheckOrigin(r)
		}
	}
	return socket
}
