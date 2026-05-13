package main

import (
	"fmt"
	"log"
	"time"

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
	options.SetKeepAlive(30 * time.Second)
	options.SetAutoReconnect(true)

	options.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("MQTT connected")
		m.subscribe("test/topic")
	})

	options.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})

	m.client = mqtt.NewClient(options)
	return m
}

func (m *MQTTClient) Connect() error {
	token := m.client.Connect()
	token.Wait()
	return token.Error()
}

func (m *MQTTClient) subscribe(topic string) {
	token := m.client.Subscribe(topic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		m.msgChan <- msg
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
		log.Println(fmt.Sprintf(`{"type":"payload":"%s"}`, msg.Topic(), msg.Payload()))
		m.hub.broadcast <- message_out{
			Type:    "server_push",
			Payload: map[string]string{"message": fmt.Sprintf(`{"type":"payload":"%s"}`, msg.Topic(), msg.Payload())},
		}
	}
}
