package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {

	fmt.Println("starting server")
	http_server()
	fmt.Println("server stopped")
}

func http_server() {
	ws_hub := NewHub()
	go ws_hub.Run()
	cfg := load_config()

	// Initialize DB (driver-agnostic). If driver isn't registered at runtime,
	// sql.Open will fail when trying to Ping.
	db, err := NewSQLDB(cfg.DB.driver, cfg.DB.dsn, cfg.DB.maxOpenConns, cfg.DB.maxIdleConns, time.Duration(cfg.DB.connMaxLifetimeSeconds)*time.Second)
	if err != nil {
		log.Printf("DB initialize error: %v", err)
	} else {
		// keep DB open for the lifetime of the server
		defer db.Close()
	}

	mqttClient := NewMQTTClient(cfg.MQTT, ws_hub)
	if err := mqttClient.Connect(); err != nil {
		log.Fatal(err)
	}
	go mqttClient.broadcast_to_websockets()
	//startPushUpdates(ws_hub)

	http.HandleFunc("/greet", greet_http_path)

	http.HandleFunc("/set_lamp_state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		// If you need cookies/auth, also add:
		// w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			// Preflight request - no body, return OK
			w.WriteHeader(http.StatusOK)
			return
		}
		post_set_lamp_state(w, r, mqttClient)
	})
	http.HandleFunc("/ws", ws_hub.ws_serve)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func greet_http_path(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World")
}

func post_set_lamp_state(w http.ResponseWriter, r *http.Request, mqtt *MQTTClient) {
	fmt.Println("Got request to set lamp state")
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(405)
		return
	}
	data := struct {
		Room  string `json:"room"`
		Lamp  string `json:"lamp"`
		State string `json:"state"` //   0/1 | on/off
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
		log.Printf("invalid state: %s", data.State)
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
