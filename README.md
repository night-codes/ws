# ws
Simple to use golang websocket client/server library + vanilla js client library.

## Example
```go
    package main

import (
	"log"
	"net/http"
	"time"

	"github.com/night-codes/ws"
	// "github.com/night-codes/ws/fasthttp/connector"
	// "github.com/night-codes/ws/tokay/connector"
	// "github.com/night-codes/ws/gin/connector"
	"github.com/night-codes/ws/http/connector"
)

func main() {
	go server()
	time.Sleep(time.Second)
	go client()
	time.Sleep(time.Second * 5)
}

// === SERVER ===
func server() {
	handler, mainWS := connector.New()

	// Listen command from the client and send answer
	mainWS.Read("test", func(a *ws.Adapter) {
		data := a.Data()
		log.Println("1)", string(data))
		a.Send("pong")
	})

	// Listen command from the client without answer
	mainWS.Read("command", func(a *ws.Adapter) {
		data := a.Data()
		log.Println("3)", string(data))
	})

	// Send command to each client and wait for answer
	go func() {
		time.Sleep(time.Second * 2)
		for _, conn := range mainWS.GetConnects() {
			if result, err := conn.Request("client test", "client ping", time.Second); err == nil {
				log.Println("5)", string(result))
			}
		}
	}()

	// Listen and serve:

	// net/http:
	http.Handle("/myws", handler) // [net/http]
	log.Fatal(http.ListenAndServe(":8080", nil))

	// fasthttp:
	/* fasthttp.ListenAndServe(":8080", func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/myws":
			handler(ctx)
		default:
			ctx.Error("Unsupported path", fasthttp.StatusNotFound)
		}
	}) */

	// tokay
	/* app := tokay.New()
	app.GET("/myws", handler)
	panic(app.Run(":8080", "Application started at %s")) */

	// gin
	/* app := gin.New()
	app.GET("/myws", handler)
	panic(app.Run(":8080")) */
}

// === CLIENT ===
func client() {
	conn := ws.NewClient("ws://localhost:8080/myws")

	// Send command to the server and wait for answer
	if result, err := conn.Request("test", "ping", time.Second); err == nil {
		log.Println("2)", string(result))
	}

	// Send command to the server without answer waiting
	conn.Send("command", "Server command 1")
	conn.Send("command", "Server command 2")
	conn.Send("command", "Server command 3")

	// Listen command from the server
	conn.Read("client test", func(a *ws.Adapter) {
		data := a.Data()
		log.Println("4)", string(data))
		a.Send("client pong")
	})
}

```

## MIT License

Copyright (c) 2018 Oleksiy Chechel

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
