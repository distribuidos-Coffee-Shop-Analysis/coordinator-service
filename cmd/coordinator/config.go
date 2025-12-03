package main

import (
	"fmt"
	"log"
	"os"

	"github.com/distribuidos-Coffee-Shop-Analysis/coordinator-service/internal/monitor"
	"gopkg.in/yaml.v3"
)

// DockerCompose represents the structure of docker-compose.yml
type DockerCompose struct {
	Services map[string]Service `yaml:"services"`
}

// Service represents a service in docker-compose.yml
type Service struct {
	ContainerName string `yaml:"container_name"`
}

// loadWorkersFromCompose reads the docker-compose.yml and extracts worker services
func loadWorkersFromCompose(composePath string) ([]monitor.CheckTarget, error) {
	// Read the compose file
	data, err := os.ReadFile(composePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	// Parse YAML
	var compose DockerCompose
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Extract all services as targets
	targets := []monitor.CheckTarget{}
	for _, service := range compose.Services {
		if service.ContainerName == "" {
			continue // Skip services without explicit container_name
		}

		targets = append(targets, monitor.CheckTarget{
			Name:          service.ContainerName,
			Host:          service.ContainerName,
			Port:          healthPort,
			ContainerName: service.ContainerName,
		})
	}

	log.Printf("Loaded %d worker nodes from compose file: %s", len(targets), composePath)
	return targets, nil
}

// getMonitoredNodes generates the complete list of nodes to monitor dynamically
// Includes workers (from docker-compose.yml) AND other coordinators (excluding self)
func getMonitoredNodes(myID, totalReplicas int) []monitor.CheckTarget {
	targets := []monitor.CheckTarget{}

	// ========================================
	// COORDINATORS (Cross-Monitoring)
	// ========================================
	for i := 1; i <= totalReplicas; i++ {
		// CRITICAL: Never monitor myself
		if i == myID {
			continue
		}

		containerName := fmt.Sprintf("coordinator-%d", i)
		targets = append(targets, monitor.CheckTarget{
			Name:          fmt.Sprintf("Coordinator %d", i),
			Host:          containerName,
			Port:          healthPort,
			ContainerName: containerName,
		})
	}

	composePath := getEnv("COMPOSE_PATH", "/app/nodes-compose.yml")

	workerTargets, err := loadWorkersFromCompose(composePath)
	if err != nil {
		log.Printf("WARNING: Failed to load workers from compose file: %v", err)
		log.Printf("Continuing with only coordinator monitoring...")
	} else {
		targets = append(targets, workerTargets...)
	}

	return targets
}
