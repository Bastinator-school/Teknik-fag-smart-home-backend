# Smart Home Backend

Go-based event broker that bridges MQTT devices and WebSocket clients.

## Configuration

The server uses defaults and overrides them from `./config.ini` if present. Empty values in `config.ini` do not override defaults.

Supported keys:

- `broker_url` (default `tcp://localhost:1883`)
- `broker_user` (default `smarthome`)
- `broker_pass` (default `smarthome`)
- `client_id` (default `smarthome-server`)

Example `config.ini`:

```
broker_url=tcp://localhost:1883
broker_user=smarthome
broker_pass=smarthome
client_id=smarthome-server
```

## Run

```bash
go build -o smarthome-server main.go config.go MQTT.go websockets.go DB.go
./smarthome-server
```

## Endpoints

- `GET /greet`
- `POST /set_lamp_state` (JSON body with `room`, `lamp`, `state`)
- `GET /ws` (WebSocket)
