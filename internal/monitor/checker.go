package monitor

import (
	"fmt"
	"log"
	"net"
	"time"
)

const (
	pingMessage = "PING"
	pongMessage = "PONG"
	dialTimeout = 2 * time.Second
	readTimeout = 2 * time.Second
)

// HealthChecker verifies the health of TCP endpoints
type HealthChecker struct{}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{}
}

// IsAlive checks if a host is responding to health checks
// Protocol: Connect -> Send "PING" -> Expect "PONG"
func (hc *HealthChecker) IsAlive(host string, port string) bool {
	address := net.JoinHostPort(host, port)
	
	// Connect with timeout
	conn, err := net.DialTimeout("tcp", address, dialTimeout)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", address, err)
		return false
	}
	defer conn.Close()

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		log.Printf("Failed to set read deadline for %s: %v", address, err)
		return false
	}

	// Send PING
	_, err = conn.Write([]byte(pingMessage))
	if err != nil {
		log.Printf("Failed to send PING to %s: %v", address, err)
		return false
	}

	// Read response
	buffer := make([]byte, len(pongMessage))
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Failed to read response from %s: %v", address, err)
		return false
	}

	response := string(buffer[:n])
	if response != pongMessage {
		log.Printf("Unexpected response from %s: got '%s', expected '%s'", address, response, pongMessage)
		return false
	}

	return true
}

// CheckTarget represents a target to monitor
type CheckTarget struct {
	Name          string
	Host          string
	Port          string
	ContainerName string
}

// String returns a string representation of the target
func (t *CheckTarget) String() string {
	return fmt.Sprintf("%s (%s:%s -> container: %s)", t.Name, t.Host, t.Port, t.ContainerName)
}

