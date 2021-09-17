package connector

import (
	"fmt"
	"net/http"

	websocket "github.com/fasthttp-contrib/websocket"
	"github.com/night-codes/ws"
	"github.com/valyala/fasthttp"
)

var (
	// CheckOrigin wsUpgrader function
	CheckOrigin func(request interface{}) bool
)

// New makes new Channel with "github.com/valyala/fasthttp".RequestCtx
func New(bufferSizes ...int) (fasthttp.RequestHandler, *ws.Channel) {
	channel := ws.NewChannel()
	wsupgrader := getFastUpgrader(bufferSizes...)

	return func(ctx *fasthttp.RequestCtx) {
		copyCtx := &fasthttp.RequestCtx{}
		ctx.Request.CopyTo(&copyCtx.Request)
		ctx.Response.CopyTo(&copyCtx.Response)

		wsupgrader.Receiver = func(conn *websocket.Conn) {
			channel.Handler(conn, copyCtx)
		}
		if err := wsupgrader.Upgrade(ctx); err != nil {
			ctx.SetStatusCode(http.StatusBadRequest)
			fmt.Fprintf(ctx, "Failed to set websocket upgrade.")
		}
	}, channel
}

func getFastUpgrader(bufferSizes ...int) *websocket.Upgrader {
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
		socket.CheckOrigin = func(r *fasthttp.RequestCtx) bool {
			return CheckOrigin(r)
		}
	}
	return socket
}
