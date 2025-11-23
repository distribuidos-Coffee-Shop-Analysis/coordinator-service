package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/distribuidos-Coffee-Shop-Analysis/coordinator-service/internal/docker"
	"github.com/distribuidos-Coffee-Shop-Analysis/coordinator-service/internal/election"
	"github.com/distribuidos-Coffee-Shop-Analysis/coordinator-service/internal/monitor"
)

const (
	checkInterval = 5 * time.Second
	healthPort    = "12346"
)

func main() {
	log.Println("Starting Coordinator Service...")

	// Read environment variables for election
	myID, err := strconv.Atoi(getEnv("MY_ID", "1"))
	if err != nil {
		log.Fatalf("Invalid MY_ID: %v", err)
	}

	totalReplicas, err := strconv.Atoi(getEnv("TOTAL_REPLICAS", "3"))
	if err != nil {
		log.Fatalf("Invalid TOTAL_REPLICAS: %v", err)
	}

	// Initialize Bully election with heartbeats
	elector := election.NewCoordinator(myID, totalReplicas)
	elector.Start()

	// Initialize Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}
	defer dockerClient.Close()

	// Initialize health checker
	healthChecker := monitor.NewHealthChecker()

	// Get all monitored nodes dynamically
	targets := getMonitoredNodes()

	log.Printf("Configured to monitor %d targets with interval: %v", len(targets), checkInterval)
	log.Printf("Waiting for leader election...")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create ticker for periodic health checks
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Main monitoring loop
	for {
		select {
		case <-ticker.C:
			if !elector.IsLeader() {
				log.Printf("Not leader (Leader ID=%d), skipping health checks", elector.GetLeaderID())
				continue
			}

			log.Printf("I am the leader, performing health checks...")

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

		case isLeader := <-elector.LeaderChan():
			if isLeader {
				log.Printf("*** BECAME LEADER - Starting active monitoring ***")
			} else {
				log.Printf("*** LOST LEADERSHIP - Entering standby mode ***")
			}

		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down...", sig)
			return
		}
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
