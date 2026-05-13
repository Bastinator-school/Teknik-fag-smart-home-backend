package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {

	fmt.Println("starting server")
	http_server()

	fmt.Println("server started")
}

func http_server() {
	ws_hub := NewHub()
	go ws_hub.Run()
	mqttClient := NewMQTTClient("tcp://localhost:1883", ws_hub)
	if err := mqttClient.Connect(); err != nil {
		log.Fatal(err)
	}
	go mqttClient.broadcast_to_websockets()
	startPushUpdates(ws_hub)

	http.HandleFunc("/greet", greet_http_path)

	http.HandleFunc("/ws", ws_hub.ws_serve)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func greet_http_path(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World")
}
