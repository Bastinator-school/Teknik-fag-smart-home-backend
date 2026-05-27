package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type config struct {
	MQTT MQTT_options
}

type MQTT_options struct {
	broker_url  string
	broker_user string
	broker_pass string
	client_id   string // name the server will use when registering to MQTT
}

func load_config() config {
	fmt.Println("loading config")

	cfg := config{
		MQTT: MQTT_options{
			broker_url:  "tcp://localhost:1883",
			broker_user: "smarthome",
			broker_pass: "smarthome",
			client_id:   "smarthome-server",
		},
	}

	file, err := os.Open("./config.ini")
	if err != nil {
		return cfg
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if value == "" {
			continue
		}
		switch key {
		case "broker_url":
			cfg.MQTT.broker_url = value
		case "broker_user":
			cfg.MQTT.broker_user = value
		case "broker_pass":
			cfg.MQTT.broker_pass = value
		case "client_id":
			cfg.MQTT.client_id = value
		}
	}

	return cfg
}
