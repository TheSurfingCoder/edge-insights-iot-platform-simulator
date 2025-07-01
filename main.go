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
	Time       time.Time `json:"time"`
	DeviceID   string    `json:"device_id"`
	DeviceType string    `json:"device_type"`
	Location   string    `json:"location"`
	RawValue   *float64  `json:"raw_value,omitempty"` // Pointer so it can be nil
	Unit       string    `json:"unit,omitempty"`
	LogType    string    `json:"log_type"`
	Message    string    `json:"message"`
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
		LogInterval:  10 * time.Second,
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

	// Debug logging - check what's being sent
	log.Printf("Sending log - Device: %s, Message: '%s', Message length: %d",
		device.ID, logMessage.Message, len(logMessage.Message))

	jsonData, err := json.Marshal(logMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	// Debug logging - check the JSON
	log.Printf("JSON data: %s", string(jsonData))

	if err := s.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		return fmt.Errorf("failed to send log: %w", err)
	}

	// Try to read response with a short timeout
	s.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

	// Read the first message (should be success response)
	var response LogResponse
	if err := s.conn.ReadJSON(&response); err != nil {
		log.Printf("⚠️  No response (continuing): %v", err)
		return nil
	}

	if response.Success {
		log.Printf("✅ Log sent successfully: %s", logMessage.LogType)
	} else {
		log.Printf("❌ Server error: %s", response.Error)
	}

	// Try to read the second message (broadcast) but don't fail if we can't
	s.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var broadcastMsg map[string]interface{}
	if err := s.conn.ReadJSON(&broadcastMsg); err == nil {
		// Successfully read broadcast message, ignore it
		log.Printf(" Broadcast received (ignored)")
	}

	return nil
}

func (s *LogSimulator) generateLogMessage(device Device) LogMessage {
	var rawValue float64
	var unit string
	var message string
	var logType string

	switch device.Type {
	case "temperature_sensor":
		rawValue, unit, message, logType = s.generateTemperatureMessage(device)
	case "humidity_sensor":
		rawValue, unit, message, logType = s.generateHumidityMessage(device)
	case "motion_detector":
		rawValue, unit, message, logType = s.generateMotionMessage(device)
	case "camera":
		rawValue, unit, message, logType = s.generateCameraMessage(device)
	case "controller":
		rawValue, unit, message, logType = s.generateControllerMessage(device)
	default:
		rawValue = 0
		unit = "unknown"
		message = fmt.Sprintf("Device %s operating normally in %s", device.ID, device.Location)
		logType = "INFO"
	}

	return LogMessage{
		Time:       time.Now(),
		DeviceID:   device.ID,
		DeviceType: device.Type,
		Location:   device.Location,
		RawValue:   &rawValue,
		Unit:       unit,
		LogType:    logType,
		Message:    message,
	}
}

func (s *LogSimulator) generateTemperatureMessage(device Device) (float64, string, string, string) {
	// Increased probability for unusual events for better testing
	randVal := rand.Float64()

	if randVal < 0.05 { // 5% chance for critical events (was 0.5%)
		// Critical temperature spike
		rawValue := 40 + rand.Float64()*10 // 40-50°C
		message := fmt.Sprintf("CRITICAL: Temperature spike to %.1f°C - HVAC system failure detected in %s. Immediate attention required.", rawValue, device.Location)
		return rawValue, "celsius", message, "CRITICAL"
	} else if randVal < 0.12 { // 7% chance for warnings (was 0.5%)
		// Temperature warning
		rawValue := 32 + rand.Float64()*8 // 32-40°C
		message := fmt.Sprintf("Temperature alert: %.1f°C detected - Above normal threshold in %s. Monitoring closely.", rawValue, device.Location)
		return rawValue, "celsius", message, "WARNING"
	} else if randVal < 0.18 { // 6% chance for maintenance (was 0.5%)
		// Maintenance event
		rawValue := 18 + rand.Float64()*4      // 18-22°C
		adjustment := 0.1 + rand.Float64()*0.4 // 0.1-0.5°C adjustment
		message := fmt.Sprintf("Temperature sensor maintenance: Calibration completed with +%.1f°C adjustment in %s. Sensor accuracy improved.", adjustment, device.Location)
		return rawValue, "celsius", message, "INFO"
	} else {
		// Normal operation (82% of the time, was 98.5%)
		rawValue := 18 + rand.Float64()*12 // 18-30°C
		message := fmt.Sprintf("Temperature reading: %.1f°C - Normal operating range maintained in %s", rawValue, device.Location)
		return rawValue, "celsius", message, "INFO"
	}
}

func (s *LogSimulator) generateHumidityMessage(device Device) (float64, string, string, string) {
	randVal := rand.Float64()

	if randVal < 0.05 { // 5% chance for critical events (was 0.5%)
		// Critical humidity issue
		rawValue := 85 + rand.Float64()*15 // 85-100%
		message := fmt.Sprintf("CRITICAL: Humidity spike to %.1f%% - Possible water leak or ventilation failure in %s", rawValue, device.Location)
		return rawValue, "percent", message, "CRITICAL"
	} else if randVal < 0.12 { // 7% chance for warnings (was 0.5%)
		// Humidity warning
		rawValue := 75 + rand.Float64()*10 // 75-85%
		message := fmt.Sprintf("Humidity warning: %.1f%% detected - Elevated moisture levels in %s. Check ventilation system.", rawValue, device.Location)
		return rawValue, "percent", message, "WARNING"
	} else if randVal < 0.18 { // 6% chance for environmental events (was 0.5%)
		// Environmental event
		rawValue := 60 + rand.Float64()*15 // 60-75%
		message := fmt.Sprintf("Environmental event: Humidity increased to %.1f%% - Weather conditions affecting %s", rawValue, device.Location)
		return rawValue, "percent", message, "INFO"
	} else {
		// Normal operation (82% of the time, was 98.5%)
		rawValue := 35 + rand.Float64()*25 // 35-60%
		message := fmt.Sprintf("Humidity level: %.1f%% - Optimal conditions maintained in %s", rawValue, device.Location)
		return rawValue, "percent", message, "INFO"
	}
}

func (s *LogSimulator) generateMotionMessage(device Device) (float64, string, string, string) {
	randVal := rand.Float64()

	if randVal < 0.05 { // 5% chance for security alerts (was 0.5%)
		// Security alert
		rawValue := 1.0
		message := fmt.Sprintf("Security alert: Unauthorized motion detected during off-hours in %s. Security team notified.", device.Location)
		return rawValue, "boolean", message, "SECURITY"
	} else if randVal < 0.12 { // 7% chance for unusual patterns (was 0.5%)
		// Unusual motion pattern
		rawValue := 1.0
		message := fmt.Sprintf("Unusual motion pattern: Multiple rapid triggers detected in %s. Investigating activity.", device.Location)
		return rawValue, "boolean", message, "WARNING"
	} else if randVal < 0.18 { // 6% chance for maintenance (was 0.5%)
		// Maintenance event
		rawValue := 0.0
		message := fmt.Sprintf("Motion sensor maintenance: Sensitivity adjustment completed in %s. False alarm rate reduced.", device.Location)
		return rawValue, "boolean", message, "INFO"
	} else {
		// Normal operation (82% of the time, was 98.5%)
		rawValue := float64(rand.Intn(2)) // 0 or 1
		var message string
		if rawValue == 1 {
			message = fmt.Sprintf("Motion detected: Standard activity in %s", device.Location)
		} else {
			message = fmt.Sprintf("Motion sensor: No activity detected in %s", device.Location)
		}
		return rawValue, "boolean", message, "INFO"
	}
}

func (s *LogSimulator) generateCameraMessage(device Device) (float64, string, string, string) {
	randVal := rand.Float64()

	if randVal < 0.05 { // 5% chance for security events (was 0.5%)
		// Security event
		rawValue := 1.0
		message := fmt.Sprintf("Security camera: Unauthorized access attempt detected in %s. Recording incident for review.", device.Location)
		return rawValue, "boolean", message, "SECURITY"
	} else if randVal < 0.12 { // 7% chance for malfunctions (was 0.5%)
		// Camera malfunction
		rawValue := 0.0
		message := fmt.Sprintf("Camera malfunction: Connection lost in %s. Attempting automatic reconnection.", device.Location)
		return rawValue, "boolean", message, "ERROR"
	} else if randVal < 0.18 { // 6% chance for maintenance (was 0.5%)
		// Maintenance event
		rawValue := 1.0
		message := fmt.Sprintf("Camera maintenance: Lens cleaning and focus adjustment completed in %s", device.Location)
		return rawValue, "boolean", message, "INFO"
	} else {
		// Normal operation (82% of the time, was 98.5%)
		rawValue := 1.0
		message := fmt.Sprintf("Camera feed: Normal surveillance activity in %s", device.Location)
		return rawValue, "boolean", message, "INFO"
	}
}

func (s *LogSimulator) generateControllerMessage(device Device) (float64, string, string, string) {
	randVal := rand.Float64()

	if randVal < 0.05 { // 5% chance for system failures (was 0.5%)
		// System failure
		rawValue := 0.0
		message := fmt.Sprintf("System controller failure: Backup system activated in %s. Primary system diagnostics running.", device.Location)
		return rawValue, "boolean", message, "CRITICAL"
	} else if randVal < 0.12 { // 7% chance for alerts (was 0.5%)
		// System alert
		rawValue := 1.0
		message := fmt.Sprintf("Controller alert: Power consumption spike detected in %s. Monitoring system performance.", device.Location)
		return rawValue, "boolean", message, "WARNING"
	} else if randVal < 0.18 { // 6% chance for maintenance (was 0.5%)
		// Maintenance event
		rawValue := 1.0
		message := fmt.Sprintf("Controller maintenance: Scheduled reboot completed in %s. All systems operational.", device.Location)
		return rawValue, "boolean", message, "INFO"
	} else {
		// Normal operation (82% of the time, was 98.5%)
		rawValue := 1.0
		message := fmt.Sprintf("System controller: All systems operating normally in %s", device.Location)
		return rawValue, "boolean", message, "INFO"
	}
}
