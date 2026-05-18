package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	//startPushUpdates(ws_hub)

	http.HandleFunc("/greet", greet_http_path)
	// TODO: Add a rest API to turn kichen lights/light on/off via MQTT publish home/kitchen/lights/(...)/set
	http.HandleFunc("/set_lamp_state", func(w http.ResponseWriter, r *http.Request) {
		post_set_lamp_state(w, r, mqttClient)
	})
	http.HandleFunc("/ws", ws_hub.ws_serve)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func greet_http_path(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World")
}

func post_set_lamp_state(w http.ResponseWriter, r *http.Request, mqtt *MQTTClient) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(405)
		return
	}
	data := struct {
		Room  string `json:"room"`
		Lamp  string `json:"lamp"`
		State string `json:"state"` //0/1 | on/off
	}{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid request body", "details": err.Error()})
		return
	}

	if data.State != "0" && data.State != "1" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		log.Printf("invalid state: %d", data.State)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid state", "details": "state must be 0 or 1"})
		return
	}

	// basic sanitization: disallow characters that would allow publishing to other topics
	if strings.ContainsAny(data.Room, "/+#") || strings.ContainsAny(data.Lamp, "/+#") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid characters in room or lamp"})
		return
	}

	// if Lamp is empty, publish a message targeting all lights in the room
	var topic string
	if data.Lamp == "" {
		topic = fmt.Sprintf("home/%s/lights/+/set", data.Room)
	} else {
		topic = fmt.Sprintf("home/%s/lights/%s/set", data.Room, data.Lamp)
	}

	mqtt.publish(topic, data.State)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "topic": topic, "published": data.State})
}
