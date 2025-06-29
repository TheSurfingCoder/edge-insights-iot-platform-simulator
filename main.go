package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// SimulatorConfig holds configuration for the log simulator
type SimulatorConfig struct {
	WebSocketURL string        // URL of the WebSocket server to connect to
	DeviceCount  int           // Number of IoT devices to simulate
	LogInterval  time.Duration // How often to send logs
	Duration     time.Duration // How long to run the simulation
}

// Device represents a simulated IoT device
type Device struct {
	ID       string
	Type     string
	Location string
}

// LogMessage represents an IoT device log entry
type LogMessage struct {
	Time     time.Time `json:"time"`
	DeviceID string    `json:"device_id"`
	LogType  string    `json:"log_type"`
	Message  string    `json:"message"`
}

// LogResponse represents the response after processing a log
type LogResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// LogSimulator manages the simulation of IoT devices
type LogSimulator struct {
	config  SimulatorConfig
	devices []Device
	conn    *websocket.Conn
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get WebSocket URL from environment variable
	wsURL := os.Getenv("WEBSOCKET_URL")
	if wsURL == "" {
		wsURL = "ws://localhost:8080/ws" // fallback
	}

	config := SimulatorConfig{
		WebSocketURL: wsURL,
		DeviceCount:  5,
		LogInterval:  2 * time.Second,
		Duration:     0, // Run indefinitely
	}

	simulator := NewLogSimulator(config)

	log.Println("Starting IoT Log Simulator...")
	log.Printf("Connecting to: %s", config.WebSocketURL)
	log.Printf("Simulating %d devices", config.DeviceCount)
	log.Printf("Log interval: %v", config.LogInterval)

	// Run indefinitely
	for {
		if err := simulator.Run(); err != nil {
			log.Printf("Simulator failed: %v", err)
			log.Println("Restarting in 30 seconds...")
			time.Sleep(30 * time.Second)
		}
	}
}

func NewLogSimulator(config SimulatorConfig) *LogSimulator {
	return &LogSimulator{
		config:  config,
		devices: generateDevices(config.DeviceCount),
	}
}

func generateDevices(count int) []Device {
	deviceTypes := []string{"temperature_sensor", "humidity_sensor", "motion_detector", "camera", "controller"}
	locations := []string{"warehouse_a", "warehouse_b", "office_floor_1", "parking_lot", "server_room"}

	devices := make([]Device, count)
	for i := 0; i < count; i++ {
		devices[i] = Device{
			ID:       fmt.Sprintf("device_%03d", i+1),
			Type:     deviceTypes[i%len(deviceTypes)],
			Location: locations[i%len(locations)],
		}
	}
	return devices
}

func (s *LogSimulator) Run() error {
	conn, _, err := websocket.DefaultDialer.Dial(s.config.WebSocketURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()
	s.conn = conn

	log.Println("Connected to WebSocket server")

	ticker := time.NewTicker(s.config.LogInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.sendRandomLog(); err != nil {
			log.Printf("Error sending log: %v", err)
		}
	}

	return nil
}

func (s *LogSimulator) sendRandomLog() error {
	device := s.devices[rand.Intn(len(s.devices))]
	logMessage := s.generateLogMessage(device)

	jsonData, err := json.Marshal(logMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	if err := s.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		return fmt.Errorf("failed to send log: %w", err)
	}

	var response LogResponse
	if err := s.conn.ReadJSON(&response); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if response.Success {
		log.Printf("✅ Log sent successfully: %s", logMessage.Message)
	} else {
		log.Printf("❌ Failed to send log: %s", response.Error)
	}

	return nil
}

func (s *LogSimulator) generateLogMessage(device Device) LogMessage {
	logTypes := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	logType := logTypes[rand.Intn(len(logTypes))]

	var message string

	switch device.Type {
	case "temperature_sensor":
		temp := 15 + rand.Float64()*25
		if logType == "ERROR" {
			message = fmt.Sprintf("Temperature sensor malfunction at %s", device.Location)
		} else {
			message = fmt.Sprintf("Temperature reading: %.1f°C at %s", temp, device.Location)
		}
	case "humidity_sensor":
		humidity := 30 + rand.Float64()*50
		if logType == "ERROR" {
			message = fmt.Sprintf("Humidity sensor calibration error at %s", device.Location)
		} else {
			message = fmt.Sprintf("Humidity reading: %.1f%% at %s", humidity, device.Location)
		}
	case "motion_detector":
		if logType == "INFO" {
			message = fmt.Sprintf("Motion detected at %s", device.Location)
		} else if logType == "ERROR" {
			message = fmt.Sprintf("Motion sensor offline at %s", device.Location)
		} else {
			message = fmt.Sprintf("Motion sensor status check at %s", device.Location)
		}
	case "camera":
		if logType == "INFO" {
			message = fmt.Sprintf("Camera recording started at %s", device.Location)
		} else if logType == "ERROR" {
			message = fmt.Sprintf("Camera storage full at %s", device.Location)
		} else {
			message = fmt.Sprintf("Camera maintenance check at %s", device.Location)
		}
	case "controller":
		if logType == "INFO" {
			message = fmt.Sprintf("System check completed at %s", device.Location)
		} else if logType == "ERROR" {
			message = fmt.Sprintf("Controller communication timeout at %s", device.Location)
		} else {
			message = fmt.Sprintf("Controller status update at %s", device.Location)
		}
	default:
		message = fmt.Sprintf("Device activity at %s", device.Location)
	}

	return LogMessage{
		Time:     time.Now(),
		DeviceID: device.ID,
		LogType:  logType,
		Message:  message,
	}
}
