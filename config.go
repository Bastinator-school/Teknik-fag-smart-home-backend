package main

import (
	"fmt"
)

type config struct {
	MQTT MQTT_options
}

type MQTT_options struct {
	broker_url  string
	broker_user string
	broker_pass string
	Client_id   string //name the server will use when registering to MQTT
}

func load_config() config {
	fmt.Println("loading config")

	return config{}
}

/*func set_MQTT_options() MQTT_options {
	os.ReadFile("config.ini")

}*/
