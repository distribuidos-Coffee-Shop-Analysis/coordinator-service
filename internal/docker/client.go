package docker

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	dockerSocket = "/var/run/docker.sock"
	dockerAPI    = "http://localhost"
	timeout      = 10 * time.Second
)

// Client wraps Docker socket connection for container management
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Docker client via Unix socket
func NewClient() (*Client, error) {
	// Create HTTP client with Unix socket transport
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.DialTimeout("unix", dockerSocket, timeout)
			},
		},
		Timeout: timeout,
	}

	// Verify connection by pinging Docker daemon
	resp, err := httpClient.Get(dockerAPI + "/v1.40/_ping")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon via socket %s: %w", dockerSocket, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker daemon returned status %d", resp.StatusCode)
	}

	log.Println("Successfully connected to Docker daemon via Unix socket")

	return &Client{httpClient: httpClient}, nil
}

// RestartContainer restarts a container by its name or ID
func (c *Client) RestartContainer(containerNameOrID string) error {
	log.Printf("Restarting container: %s", containerNameOrID)

	// POST request to restart endpoint
	// Docker API: POST /containers/{id}/restart
	url := fmt.Sprintf("%s/v1.40/containers/%s/restart", dockerAPI, containerNameOrID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create restart request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerNameOrID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Docker API returned status %d for container %s", resp.StatusCode, containerNameOrID)
	}

	log.Printf("Container %s restarted successfully", containerNameOrID)
	return nil
}

// Close closes the Docker client
func (c *Client) Close() error {
	if c.httpClient != nil {
		log.Println("Closing Docker client")
		c.httpClient.CloseIdleConnections()
	}
	return nil
}
