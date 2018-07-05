package ws

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/night-codes/tokay"
	tokayWebsocket "github.com/night-codes/tokay-websocket"
	"github.com/valyala/fasthttp"
)

type (
	// Map is alias for map[string]interface{}
	Map map[string]interface{}
)

var (
	// CheckOrigin wsUpgrader function
	CheckOrigin func(request interface{}) bool
)

// New makes new Channel with "net/http".Request
func New(bufferSizes ...int) (http.HandlerFunc, *Channel) {
	channel := newChannel()
	wsupgrader := getWsupgrader(bufferSizes...)
	return func(w http.ResponseWriter, r *http.Request) {
		if conn, err := wsupgrader.Upgrade(w, r, nil); err == nil {
			channel.handler(conn, r)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "Failed to set websocket upgrade.\n")
		}
	}, channel
}

// NewFasthttp makes new Channel with "github.com/valyala/fasthttp".RequestCtx
func NewFasthttp(bufferSizes ...int) (fasthttp.RequestHandler, *Channel) {
	channel := newChannel()
	wsupgrader := getFastUpgrader(bufferSizes...)

	return func(ctx *fasthttp.RequestCtx) {
		copyCtx := &fasthttp.RequestCtx{}
		ctx.Request.CopyTo(&copyCtx.Request)
		ctx.Response.CopyTo(&copyCtx.Response)

		wsupgrader.Receiver = func(conn *tokayWebsocket.Conn) {
			channel.handler(conn, copyCtx)
		}
		if err := wsupgrader.Upgrade(ctx); err != nil {
			ctx.SetStatusCode(http.StatusBadRequest)
			fmt.Fprintf(ctx, "Failed to set websocket upgrade.")
		}
	}, channel
}

// NewTokay makes new Channel with "github.com/night-codes/tokay".RouterGroup
func NewTokay(path string, r *tokay.RouterGroup, bufferSizes ...int) *Channel {
	channel := newChannel()
	wsupgrader := getFastUpgrader(bufferSizes...)

	r.GET(path, func(c *tokay.Context) {
		cc := c.Copy()
		wsupgrader.Receiver = func(conn *tokayWebsocket.Conn) {
			cc.WSConn = conn
			channel.handler(conn, cc)
		}
		if err := wsupgrader.Upgrade(c.RequestCtx); err != nil {
			c.String(http.StatusBadRequest, "Failed to set websocket upgrade.")
		}
	})
	return channel
}

// NewGin makes new Channel with "github.com/gin-gonic/gin".RouterGroup
func NewGin(path string, r *gin.RouterGroup, bufferSizes ...int) *Channel {
	channel := newChannel()
	wsupgrader := getWsupgrader(bufferSizes...)
	r.GET(path, func(c *gin.Context) {
		cc := c.Copy()
		if conn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil); err == nil {
			channel.handler(conn, cc)
		} else {
			c.String(http.StatusBadRequest, "Failed to set websocket upgrade.")
		}
	})
	return channel
}

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
