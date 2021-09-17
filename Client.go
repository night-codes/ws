package ws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/night-codes/events"

	"github.com/gorilla/websocket"
	"github.com/night-codes/conv"
)

type (
	// Client ws instance
	Client struct {
		url           string
		chBreak       chan bool
		dialer        *websocket.Dialer
		conn          *websocket.Conn
		send          chan *sndMsg
		requestID     int64
		subLock       sync.RWMutex
		subscriptions map[string]bool
		readers       *readersMap
		requests      *requestsMap
		timeout       time.Duration
		connected     bool
		debug         bool
		Reconnect     *events.Event
	}

	sndMsg struct {
		requestID int64
		command   string
		data      []byte
	}
)

// NewClient makes new WC Client
func NewClient(url string, debug ...bool) *Client {
	if len(debug) == 0 {
		debug = append(debug, false)
	}
	ws := &Client{
		url:       url,
		requestID: 0,
		dialer: &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: time.Second,
		},
		send:          make(chan *sndMsg, 100000),
		subLock:       sync.RWMutex{},
		subscriptions: map[string]bool{},
		readers:       newReaderMap(),
		requests:      newRequestsMap(),
		timeout:       time.Second * 30,
		debug:         debug[0],
		Reconnect:     events.New(),
		chBreak:       make(chan bool, 2),
	}

	go ws.connect()
	return ws
}

// Request information from server
func (c *Client) Request(command string, message interface{}, timeout ...time.Duration) ([]byte, error) {
	requestID := atomic.AddInt64(&c.requestID, 1)
	resultCh := make(chan []byte)
	timeoutD := c.timeout

	if len(timeout) > 0 {
		timeoutD = timeout[0]
	}
	c.requests.Set(requestID, func(a *Adapter) {
		resultCh <- a.Data()
	})
	if err := c.Send(command, message, requestID); err != nil {
		c.requests.Delete(requestID)
		return []byte{}, err
	}

	select {
	case result := <-resultCh:
		return result, nil
	case <-time.After(timeoutD):
		c.requests.Delete(requestID)
		return []byte{}, fmt.Errorf("\"%s\" request timeout", command)
	}
}

// Send message to server
func (c *Client) Send(command string, message interface{}, requestID ...int64) (err error) {
	go func() {
		var reqID int64
		if len(requestID) > 0 {
			reqID = requestID[0]
		}

		bytesMessage, ok := message.([]byte)
		if !ok {
			bytesMessage, err = json.Marshal(message)
			if err != nil {
				return
			}
		}

		c.send <- &sndMsg{
			requestID: reqID,
			command:   command,
			data:      bytesMessage,
		}
	}()
	return nil
}

// Read is client message (request) handler
func (c *Client) Read(command string, fn func(*Adapter)) {
	c.readers.Set(command, fn)
}

// ChangeURL for client connection
func (c *Client) ChangeURL(url string) {
	c.url = url
	c.chBreak <- true
}

// Close client connection
func (c *Client) Close() {
	c.url = ""
	c.chBreak <- true
}

func (c *Client) connect() {
	for {
		close := func() bool {
			if c.url == "" {
				return true
			}
			var err error
			c.conn, _, err = c.dialer.Dial(c.url, http.Header{
				"ws-client": []string{"true"},
			})
			if err != nil {
				return false
			}

			if c.debug {
				fmt.Printf("ws.Client: + Connected to %s\n", c.url)
			}
			go c.Reconnect.Emit(true)
			c.connected = true
			closed := make(chan bool)
			go func() {
				for {
					select {
					case msg := <-c.send:
						c.conn.WriteMessage(websocket.TextMessage, append([]byte(conv.String(msg.requestID)+":"+msg.command+":"), msg.data...))
					case <-closed:
						return
					}
				}
			}()

			cmds := []string{}
			c.subLock.RLock()
			for command := range c.subscriptions {
				cmds = append(cmds, command)
			}
			c.subLock.RUnlock()

			for _, command := range cmds {
				c.Send("subscribe", command)
			}

		cycle:
			for {
				chMessage := make(chan []byte)
				go func() {
					_, message, err := c.conn.ReadMessage()
					if err != nil {
						c.chBreak <- true
					}
					chMessage <- message
				}()

				select {
				case message := <-chMessage:
					result := bytes.SplitN(message, []byte(":"), 3)
					if len(result) == 3 {
						requestID := conv.Int64(result[0])
						command := string(result[1])
						data := result[2]

						if requestID > 0 { // answer to the request from client
							if fn, ex := c.requests.GetEx(requestID); ex {
								fn(newAdapter(command, nil, &data, requestID))
								c.requests.Delete(requestID)
							}
						} else if fns, exists := c.readers.GetEx(command); exists {
							adapter := newAdapter(command, nil, &data, requestID)
							adapter.client = c
							for _, fn := range fns {
								fn(adapter)
							}
						}
					}
				case <-c.chBreak:
					if c.url == "" {
						return true
					}
					break cycle
				}
			}

			if c.debug {
				fmt.Printf("ws.Client: - Connection closed: %s\n", c.url)
			}
			c.connected = false
			c.conn.Close()
			closed <- true
			return false
		}()

		if close {
			return
		}

		time.Sleep(time.Second / 20)
	}
}

// Subscribe connection to command
func (c *Client) Subscribe(command string) {
	if c.connected {
		c.Send("subscribe", command)
	}
	c.subLock.Lock()
	c.subscriptions[command] = true
	c.subLock.Unlock()
}

// UnSubscribe from command
func (c *Client) UnSubscribe(command string) {
	c.subLock.Lock()
	delete(c.subscriptions, command)
	c.subLock.Unlock()
	if c.connected {
		c.ChangeURL(c.url)
	}
}

/* func (c *Client) Wait(commands []string, fn func()) {
	//...
} */
