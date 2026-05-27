package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

type message_in struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type message_out struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type ws_client struct {
	send chan message_out
}

type ws_hub struct {
	clients_map map[*ws_client]struct{}
	register    chan *ws_client
	unregister  chan *ws_client
	broadcast   chan message_out
}

func NewHub() *ws_hub {
	return &ws_hub{
		clients_map: make(map[*ws_client]struct{}),
		register:    make(chan *ws_client),
		unregister:  make(chan *ws_client),
		broadcast:   make(chan message_out, 64),
	}
}

// in this case run() is a part of hub object so its run like you would run a function within a method in c/c++
// Run must be called in its own goroutine (go hub.Run()).
func (hub *ws_hub) Run() {
	for {
		select {
		case client := <-hub.register:
			hub.clients_map[client] = struct{}{}
			log.Printf("client connected  (total: %d)", len(hub.clients_map))

		case client := <-hub.unregister:
			if _, ok := hub.clients_map[client]; ok {
				delete(hub.clients_map, client)
				close(client.send)
				log.Printf("client disconnected (total: %d)", len(hub.clients_map))
			}

		case msg := <-hub.broadcast:
			for client := range hub.clients_map {
				// Non-blocking send; drop slow clients rather than blocking the hub.
				select {
				case client.send <- msg:
				default:
					log.Println("slow client – dropping message")
				}
			}
		}
	}
}

func (hub *ws_hub) ws_serve(writer http.ResponseWriter, request *http.Request) {
	conn, err := websocket.Accept(writer, request, &websocket.AcceptOptions{
		// Adjust origin policy for your needs:
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}
	client := &ws_client{send: make(chan message_out, 32)}
	hub.register <- client //add client to clients_map via the hub.register method

	ctx, cancel := context.WithCancel(request.Context())
	defer cancel()

	go func() {
		for msg := range client.send {
			data, err := json.Marshal(msg)
			if err != nil {
				log.Println("marshal:", err)
				continue
			}
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				log.Println("write:", err)
				cancel() // signal reader to stop
				return
			}
		}
	}()

	defer func() {
		hub.unregister <- client //remove client from clients_map via the hub.unregister method
		conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			return // client gone
		}

		var msg message_in
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Println("bad message:", err)
			continue
		}

		response, err := handleClientMessage(msg)
		if err != nil {
			client.send <- message_out{Type: "error", Payload: err.Error()}
			continue
		}
		if response != nil {
			client.send <- *response // reply only to the requesting client
		}
	}

}

func handleClientMessage(msg message_in) (*message_out, error) {
	switch msg.Type {

	case "ping":
		return &message_out{Type: "pong", Payload: "pong"}, nil

	case "get_time":
		return &message_out{
			Type:    "time",
			Payload: time.Now().UTC().Format(time.RFC3339),
		}, nil

	// Add more request types here:
	// case "subscribe": ...
	// case "query":     ...

	default:
		log.Printf("unknown message type: %q", msg.Type)
		return nil, nil // silently ignore unknown types; or return an error
	}
}

/*func startPushUpdates(hub *ws_hub) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for t := range ticker.C {
			hub.broadcast <- message_out{
				Type:    "server_push",
				Payload: map[string]string{"message": "data changed", "at": t.UTC().Format(time.RFC3339)},
			}
		}
	}()
}
*/
