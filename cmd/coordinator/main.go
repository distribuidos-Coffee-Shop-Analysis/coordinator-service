package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/distribuidos-Coffee-Shop-Analysis/coordinator-service/internal/docker"
	"github.com/distribuidos-Coffee-Shop-Analysis/coordinator-service/internal/monitor"
)

const (
	checkInterval = 5 * time.Second
	healthPort    = "12346"
)

func main() {
	log.Println("Starting Coordinator Service...")

	// Initialize Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}
	defer dockerClient.Close()

	// Initialize health checker
	healthChecker := monitor.NewHealthChecker()

	// Define targets to monitor
	// Add all your containers here that need health monitoring
	targets := []monitor.CheckTarget{
		{
			Name:          "Connection Node",
			Host:          "connection-node",
			Port:          healthPort,
			ContainerName: "connection-node",
		},
		{
			Name:          "Filter Year 1",
			Host:          "filter-node-year-1",
			Port:          healthPort,
			ContainerName: "filter-node-year-1",
		},
		{
			Name:          "Filter Year 2",
			Host:          "filter-node-year-2",
			Port:          healthPort,
			ContainerName: "filter-node-year-2",
		},
		{
			Name:          "Joiner Q2",
			Host:          "joiner-node-q2-1",
			Port:          healthPort,
			ContainerName: "joiner-node-q2-1",
		},
		// Add more nodes as needed from your docker-compose
	}

	log.Printf("Monitoring %d targets with interval: %v", len(targets), checkInterval)
	for _, target := range targets {
		log.Printf("  - %s", target.String())
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create ticker for periodic health checks
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	log.Println("Health monitoring started")

	// Main monitoring loop
	for {
		select {
		case <-ticker.C:
			// Check health of all targets
			for _, target := range targets {
				if !healthChecker.IsAlive(target.Host, target.Port) {
					log.Printf("ERROR: %s is not responding to health checks", target.Name)
					log.Printf("Attempting to restart container: %s", target.ContainerName)
					
					if err := dockerClient.RestartContainer(target.ContainerName); err != nil {
						log.Printf("ERROR: Failed to restart container %s: %v", target.ContainerName, err)
					} else {
						log.Printf("SUCCESS: Container %s restarted", target.ContainerName)
					}
				} else {
					log.Printf("OK: %s is healthy", target.Name)
				}
			}

		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down...", sig)
			return
		}
	}
}

