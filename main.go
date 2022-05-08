package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

const (
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		// Handle the upgrade request, and acquire the WebSocket connection.
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print(err.Error())
			return
		}
		defer c.Close()

		onClose := make(chan interface{})

		go func() {
			defer func() { close(onClose) }()
			for {
				_, _, e := c.ReadMessage()
				if e != nil {
					log.Print(err.Error())
					// An error means that connection was closed
					return
				}
			}
		}()

		go func() {
			ticker := time.Tick(time.Second * 50)

			c.SetReadLimit(2048)
			c.SetReadDeadline(time.Now().Add(pongWait))
			c.SetPongHandler(func(string) error { c.SetReadDeadline(time.Now().Add(pongWait)); return nil })

			for {
				<-ticker

			}
		}()

		<-onClose

	}).Methods("UPGRADE")

	log.Print("Server listening on port 8080")
	panic(http.ListenAndServe("0.0.0.0:8080", r))
}
