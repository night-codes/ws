package connector

import (
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/night-codes/ws"
)

var (
	// CheckOrigin wsUpgrader function
	CheckOrigin func(request interface{}) bool
)

// New makes new Channel with "net/http".Request
func New(bufferSizes ...int) (http.HandlerFunc, *ws.Channel) {
	channel := ws.NewChannel()
	wsupgrader := getWsupgrader(bufferSizes...)
	return func(w http.ResponseWriter, r *http.Request) {
		if conn, err := wsupgrader.Upgrade(w, r, nil); err == nil {
			channel.Handler(conn, r)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "Failed to set websocket upgrade.\n")
		}
	}, channel
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
