package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type config struct {
	MQTT MQTT_options
	DB   DB_options
}

type MQTT_options struct {
	broker_url  string
	broker_user string
	broker_pass string
	client_id   string // name the server will use when registering to MQTT
}

type DB_options struct {
	driver                 string
	dsn                    string
	maxOpenConns           int
	maxIdleConns           int
	connMaxLifetimeSeconds int
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
		DB: DB_options{
			driver:                 "sqlite",
			dsn:                    "smarthome.db",
			maxOpenConns:           10,
			maxIdleConns:           2,
			connMaxLifetimeSeconds: 3600,
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
		case "db_driver":
			cfg.DB.driver = value
		case "db_dsn":
			cfg.DB.dsn = value
		case "db_max_open_conns":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.DB.maxOpenConns = v
			}
		case "db_max_idle_conns":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.DB.maxIdleConns = v
			}
		case "db_conn_max_lifetime_secs":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.DB.connMaxLifetimeSeconds = v
			}
		}
	}

	return cfg
}
