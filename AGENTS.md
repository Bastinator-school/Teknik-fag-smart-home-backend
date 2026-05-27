# AGENTS.md - Smart Home Backend Development Guide

## Architecture Overview

This is a **Go-based event broker** connecting MQTT IoT devices to WebSocket-connected frontend clients. The system has three main flow paths:

1. **Device → Backend**: MQTT messages from smart home devices flow to the `MQTTClient`
2. **Backend → Frontend**: MQTT messages are automatically broadcast to all WebSocket clients via the `ws_hub`
3. **Frontend → Device**: HTTP POST requests to `/set_lamp_state` are converted to MQTT publish operations

### Component Relationships
```
HTTP Server (port 8080)
├── /set_lamp_state (POST) ──→ MQTTClient.publish()
├── /ws (WebSocket) ─────────→ ws_hub (manages clients)
└── /greet (GET) ──────────→ simple response

MQTT Broker (localhost:1883)
↓ (subscribed topics)
MQTTClient ──→ msgChan (buffered ch.)
              ↓
         broadcast_to_websockets() ──→ hub.broadcast
                                          ↓
                                    All connected ws_clients
```

## Critical Component Patterns

### WebSocket Hub Pattern (websockets.go)
- **Single goroutine model**: Call `go hub.Run()` once at startup
- **Channel-based operations**: Register/unregister clients via channels, never map directly
- **Non-blocking broadcasts**: Uses `select` with `default` to drop slow clients rather than blocking
  - This prevents one slow client from blocking message delivery to others
- **Message structure**: All messages use `message_in{Type, Payload}` and `message_out{Type, Payload}`
- **Handler extensibility**: Add new message types in `handleClientMessage()` switch statement

### MQTT Client Pattern (MQTT.go)
- **Constructor approach**: `NewMQTTClient()` creates client with hardcoded credentials and subscriptions
- **Topic subscription at connect-time**: Configured in `SetOnConnectHandler` callback (line 27-38)
- **Message buffering**: `msgChan` has 64-slot buffer to handle burst messages
- **Mandatory broadcast loop**: Call `go mqttClient.broadcast_to_websockets()` to drain the message channel
  - If this goroutine isn't running, messages queue up and can cause memory issues
- **Topic structure**: `home/{room}/{device_type}/{device_name}/{attribute}` (e.g., `home/kitchen/lights/ceiling/state`)

### CORS & Security (main.go)
- **CORS wildcard enabled** on `/set_lamp_state` for development (InsecureSkipVerify)
- **Input sanitization**: Room and lamp names must NOT contain `/+#` (MQTT wildcard chars)
  - This prevents topic injection attacks
- **State validation**: Lamp state must be exactly `"0"` or `"1"` (string values, not boolean)

## Build & Deployment

**Build command:**
```bash
go build -o smarthome-server main.go config.go MQTT.go websockets.go DB.go
```

**Hard dependencies at runtime:**
- MQTT broker listening on `tcp://localhost:1883` with credentials: user=`smarthome`, pass=`smarthome`
- Client credentials in `MQTTClient` hardcoded (line 23-25) – change before production

**Configuration:**
- `config.ini` exists but is **not currently loaded** (see commented code in config.go line 24-27)
- All MQTT subscriptions hardcoded in `NewMQTTClient()` – must edit source to change topics

## Workflow: Adding a New Feature

1. **New WebSocket message type** → Add case in `handleClientMessage()` switch, return `message_out`
2. **New MQTT subscription** → Edit topics map in `SetOnConnectHandler()` callback
3. **New HTTP endpoint** → Add `http.HandleFunc()` in `http_server()`, ensure CORS headers match `/set_lamp_state`
4. **Broadcasting to clients** → Send to `hub.broadcast` channel as `message_out{Type, Payload}`

## Key Files

- **main.go** (102 lines): HTTP routing, entry point, CORS setup
- **websockets.go** (167 lines): Hub and client management, message protocol
- **MQTT.go** (89 lines): MQTT connection, subscriptions, message forwarding
- **config.go** (28 lines): Config structures (not yet integrated)
- **go.mod**: Dependencies (coder/websocket, eclipse/paho.mqtt.golang)

## Known Limitations & TODOs

- Credentials hardcoded: `smarthome`/`smarthome` (line MQTT.go:24-25)
- Config loading incomplete (see commented sections in config.go:24-27)
- Database integration placeholder (DB.go empty)
- No authentication/authorization layer
- No persistence of client state between restarts
- InsecureSkipVerify on WebSocket (development-only setting)

## Testing Endpoints

```bash
# HTTP GET - simple test
curl http://localhost:8080/greet

# HTTP POST - control lights (must send JSON)
curl -X POST http://localhost:8080/set_lamp_state \
  -H "Content-Type: application/json" \
  -d '{"room":"kitchen","lamp":"ceiling","state":"1"}'

# WebSocket - connect and send ping
wscat -c ws://localhost:8080/ws
# Then send: {"type":"ping"}
# Should receive: {"type":"pong","payload":"pong"}
```

