package main

import (
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	client  mqtt.Client
	msgChan chan mqtt.Message
	hub     *ws_hub
}

func NewMQTTClient(brokerURL string, hub *ws_hub) *MQTTClient {
	m := &MQTTClient{
		msgChan: make(chan mqtt.Message, 64),
		hub:     hub,
	}

	options := mqtt.NewClientOptions()
	options.AddBroker(brokerURL)
	options.SetClientID("smarthome-server")
	options.SetUsername("smarthome")
	options.SetPassword("smarthome")

	options.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("MQTT connected")
		topics := map[string]byte{
			"home/kitchen/lights/+/state":              1, //this subscribes to /home/kitchen/(any light)/state
			"home/kitchen/lights/+/set":                1, //TODO: Remove this its debug
			"home/devices/arduino/state":               2, //this subscribes to the arduino state so if the arduino unexpectedly disconnects from MQTT we can warn the user
			"home/kitchen/sensors/temperature/+/value": 1, //this subscribes to /home/kitchen/sensors/temperature/(any sensor)/value
			"home/kitchen/sensors/humidity/+/value":    1, //this subscribes to /home/kitchen/sensors/temperature/(any sensor)/value
		}
		m.subscribe(topics)
	})

	options.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})

	m.client = mqtt.NewClient(options)
	return m
}

func (m *MQTTClient) publish(topic string, data string) {
	token := m.client.Publish(topic, 0, false, data)
	// Wait for the publish to complete and check for errors
	token.Wait()
	if err := token.Error(); err != nil {
		log.Printf("MQTT publish error to %s: %v", topic, err)
		return
	}
	log.Printf("MQTT published message: %v to topic: %s", data, topic)
}

func (m *MQTTClient) Connect() error {
	token := m.client.Connect()
	token.Wait()
	return token.Error()
}

func (m *MQTTClient) subscribe(topic map[string]byte) {
	token := m.client.SubscribeMultiple(topic, func(_ mqtt.Client, msg mqtt.Message) {
		m.msgChan <- msg
		log.Printf("MQTT received message: %s", string(msg.Payload()))
	})
	token.Wait()
	if err := token.Error(); err != nil {
		log.Printf("Subscribe error: %v", err)
	}
}

// broadcast_to_websockets drains the message channel and broadcasts to WebSocket clients.
// Call this in its own goroutine.
func (m *MQTTClient) broadcast_to_websockets() {
	for msg := range m.msgChan {
		// Log and forward structured payload to websocket clients
		payloadStr := string(msg.Payload())
		log.Printf("MQTT received message - topic: %s payload: %s", msg.Topic(), payloadStr)
		m.hub.broadcast <- message_out{
			Type:    "server_push",
			Payload: map[string]string{"topic": msg.Topic(), "payload": payloadStr},
		}
	}
}
