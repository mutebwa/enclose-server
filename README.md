# Enclosure Game Server

![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)
[![Gorilla WebSocket](https://img.shields.io/badge/WebSocket-Gorilla-blueviolet)](https://github.com/gorilla/websocket)

The backend server for the Enclosure multiplayer game, built with Go and Gorilla WebSocket. This server handles real-time game logic, player matchmaking, and state synchronization.

## Features

- 🚀 Real-time multiplayer gameplay using WebSockets
- ⚡ Efficient grid-based game state management
- 🔒 Thread-safe concurrent operations
- 🧩 Automatic matchmaking system
- 🔄 Graceful shutdown and cleanup
- 📈 Scalable architecture for multiple game rooms
- 🛡️ Rate limiting and connection management

## Prerequisites

- Go 1.21+
- Git
- Make (optional)

## Getting Started

### Installation

1. Clone the repository:

```bash
git clone https://github.com/yourusername/enclosure-server.git
cd enclosure-server
```

2. Install dependencies:

```bash
go mod download
```

### Configuration

Configure the server using environment variables:

```bash
export PORT=8080
export MAX_GAMES=100
export GAME_TIMEOUT=30m
```

### Running the Server

```bash
go run cmd/server/main.go
```

Or with custom parameters:

```bash
PORT=8080 MAX_GAMES=50 go run cmd/server/main.go
```

### Testing the Connection

Use `wscat` to test WebSocket connectivity:

```bash
wscat -c ws://localhost:8080/ws
```

## API Documentation

### WebSocket Endpoint

- **URL**: `ws://localhost:8080/ws`
- **Protocol**: JSON over WebSocket

### Message Formats

#### From Client

```json
{
  "type": "move",
  "x": 5,
  "y": 5
}

{
  "type": "ready"
}

{
  "type": "reset"
}
```

#### From Server

```json
{
  "type": "gameState",
  "state": {
    "grid": [[...]],
    "scores": [0, 0],
    "currentTurn": 1,
    "gameOver": false,
    "message": "Player 1's Turn"
  }
}

{
  "type": "error",
  "message": "Invalid move"
}
```

## Game Rules

The server enforces:

- 64x64 grid gameplay
- Turn-based moves
- Enclosure detection
- Score calculation
- Win condition checks
- Player timeout handling

## Deployment

### Production Setup

1. Build the binary:

```bash
go build -o enclosure-server cmd/server/main.go
```

2. Run with systemd service:

```ini
[Unit]
Description=Enclosure Game Server

[Service]
ExecStart=/path/to/enclosure-server
Restart=always
Environment=PORT=8080 MAX_GAMES=100

[Install]
WantedBy=multi-user.target
```

### Docker

```Dockerfile
FROM golang:1.21-alpine

WORKDIR /app
COPY . .

RUN go build -o enclosure-server cmd/server/main.go

CMD ["./enclosure-server"]
```

## Architecture

```
server/
├── cmd/          # Main application entrypoint
├── internal/
│   ├── game/     # Core game logic
│   ├── hub/      # Connection management
│   └── handler/  # WebSocket handlers
└── pkg/          # Reusable utilities
```

## Monitoring

Access metrics via Prometheus endpoint (future implementation):

```bash
curl http://localhost:8080/metrics
```

## Contributing

1. Fork the repository
2. Create feature branch:

```bash
git checkout -b feature/new-feature
```

3. Commit changes following [Conventional Commits](https://www.conventionalcommits.org/)
4. Push and create Pull Request

## License

Probably MIT License

## Acknowledgements

- Gorilla WebSocket team
- Original game concept by Glody Mutebwa

```

This README includes:
1. Key technical details for developers
2. Clear setup instructions
3. API documentation
4. Deployment guidelines
5. Contribution guidelines
6. Architectural overview
7. Production considerations
