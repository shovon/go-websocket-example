package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// So, as it turns out, in order to maintain a more robust connection between
// two hosts in a stable connection, ideally, both hosts need to be sending
// pings to eachother. Otherwise, either hosts may mistake the other host as
// having silently closed the connection.
//
// This example demoes on how to implement a WebSocket connection that will
// be relatively stable.
//
// This example also demoes using some additional safeguards that WebSocket has.
// For example, you may want to limit the amount of bytes being provided by the
// client to the server. Additionally, set a read deadline. That is, the wait
// time until the other host sends at least any message.

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	writeWait = 60 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

// A 64KiB read limit from the other host
const readLimit = 1024 * 64

// So the idea is this:
//
// A read deadline is set every time we receive a pong. However, a read deadline
// will also be set when the program first starts up.

var mut sync.Mutex

func writeMessage(c *websocket.Conn, messageType int, data []byte) error {
	mut.Lock()
	defer mut.Unlock()
	return c.WriteMessage(messageType, data)
}

func randInt(max int) int {
	return int(rand.Float32() * float32(max))
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Print("Got a new connection")
		// Handle the upgrade request, and acquire the WebSocket connection.
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print(err.Error())
			return
		}
		defer c.Close()

		c.SetReadLimit(readLimit)

		// Setting things up for pinging
		c.SetReadDeadline(time.Now().Add(pongWait))
		c.SetPongHandler(func(string) error {
			c.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		onClose := make(chan interface{})
		messages := make(chan []byte)

		go func() {
			defer func() { onClose <- nil }()
			for {
				_, message, e := c.ReadMessage()
				if e != nil {
					return
				}
				messages <- message
			}
		}()

		go func() {
			ticker := time.Tick(time.Second * 50)
			for {
				select {
				case <-ticker:
					c.SetWriteDeadline(time.Now().Add(writeWait))
					if err := writeMessage(c, websocket.PingMessage, nil); err != nil {
						onClose <- nil
					}
				case msg := <-messages:
					fmt.Println(string(msg))
					c.SetWriteDeadline(time.Now().Add(writeWait))
					go func() {
						<-time.Tick(time.Second * time.Duration(randInt(10)))
						if err := writeMessage(c, websocket.TextMessage, []byte(fmt.Sprintf("Got message: %s", string(msg)))); err != nil {
							onClose <- nil
						}
					}()
				case <-onClose:
					return
				}
			}
		}()

		<-onClose

	})

	log.Print("Server listening on port 8080")
	panic(http.ListenAndServe("0.0.0.0:8080", r))
}
