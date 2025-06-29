# Edge Insights IoT Simulator

A Go application that generates realistic IoT device logs and sends them to the Edge Insights WebSocket server.

## Features

- Simulates multiple IoT device types (sensors, cameras, controllers)
- Generates realistic log messages with different severity levels
- Sends logs via WebSocket to the Edge Insights server
- Runs continuously to provide live data for testing and demonstration

## Environment Variables

- `WEBSOCKET_URL`: URL of the WebSocket server (e.g., `wss://your-server.railway.app/ws`)

## Usage

```bash
go run main.go
```

## Deployment

This simulator is designed to be deployed as a background worker service on platforms like Render or Railway.
