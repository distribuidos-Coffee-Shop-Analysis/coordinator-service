package main

import (
	"fmt"

	"github.com/distribuidos-Coffee-Shop-Analysis/coordinator-service/internal/monitor"
)

// getMonitoredNodes generates the complete list of nodes to monitor dynamically
// Includes workers AND other coordinators (excluding self)
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

	// ========================================
	// WORKERS
	// ========================================

	// Connection Node
	targets = append(targets, monitor.CheckTarget{
		Name:          "Connection Node",
		Host:          "connection-node",
		Port:          healthPort,
		ContainerName: "connection-node",
	})

	// Filter Nodes - Year (1-4)
	for i := 1; i <= 4; i++ {
		containerName := fmt.Sprintf("filter-node-year-%d", i)
		targets = append(targets, monitor.CheckTarget{
			Name:          fmt.Sprintf("Filter Year %d", i),
			Host:          containerName,
			Port:          healthPort,
			ContainerName: containerName,
		})
	}

	// Filter Nodes - Hour (1-4)
	for i := 1; i <= 4; i++ {
		containerName := fmt.Sprintf("filter-node-hour-%d", i)
		targets = append(targets, monitor.CheckTarget{
			Name:          fmt.Sprintf("Filter Hour %d", i),
			Host:          containerName,
			Port:          healthPort,
			ContainerName: containerName,
		})
	}

	// Filter Nodes - Amount (1-4)
	for i := 1; i <= 4; i++ {
		containerName := fmt.Sprintf("filter-node-amount-%d", i)
		targets = append(targets, monitor.CheckTarget{
			Name:          fmt.Sprintf("Filter Amount %d", i),
			Host:          containerName,
			Port:          healthPort,
			ContainerName: containerName,
		})
	}

	// Group By Nodes - Q2 (1-4)
	for i := 1; i <= 4; i++ {
		containerName := fmt.Sprintf("group-by-node-q2-%d", i)
		targets = append(targets, monitor.CheckTarget{
			Name:          fmt.Sprintf("Group By Q2 %d", i),
			Host:          containerName,
			Port:          healthPort,
			ContainerName: containerName,
		})
	}

	// Group By Nodes - Q3 (1-4)
	for i := 1; i <= 4; i++ {
		containerName := fmt.Sprintf("group-by-node-q3-%d", i)
		targets = append(targets, monitor.CheckTarget{
			Name:          fmt.Sprintf("Group By Q3 %d", i),
			Host:          containerName,
			Port:          healthPort,
			ContainerName: containerName,
		})
	}

	// Group By Nodes - Q4 (1-4)
	for i := 1; i <= 4; i++ {
		containerName := fmt.Sprintf("group-by-node-q4-%d", i)
		targets = append(targets, monitor.CheckTarget{
			Name:          fmt.Sprintf("Group By Q4 %d", i),
			Host:          containerName,
			Port:          healthPort,
			ContainerName: containerName,
		})
	}

	// Aggregate Nodes
	aggregates := []string{"q2", "q3", "q4"}
	for _, query := range aggregates {
		containerName := fmt.Sprintf("aggregate-node-%s-1", query)
		targets = append(targets, monitor.CheckTarget{
			Name:          fmt.Sprintf("Aggregate %s", query),
			Host:          containerName,
			Port:          healthPort,
			ContainerName: containerName,
		})
	}

	// Joiner Nodes - Q2 (1)
	targets = append(targets, monitor.CheckTarget{
		Name:          "Joiner Q2",
		Host:          "joiner-node-q2-1",
		Port:          healthPort,
		ContainerName: "joiner-node-q2-1",
	})

	// Joiner Nodes - Q3 (1)
	targets = append(targets, monitor.CheckTarget{
		Name:          "Joiner Q3",
		Host:          "joiner-node-q3-1",
		Port:          healthPort,
		ContainerName: "joiner-node-q3-1",
	})

	// Joiner Nodes - Q4 Users (1-5)
	for i := 1; i <= 5; i++ {
		containerName := fmt.Sprintf("joiner-node-q4-users-%d", i)
		targets = append(targets, monitor.CheckTarget{
			Name:          fmt.Sprintf("Joiner Q4 Users %d", i),
			Host:          containerName,
			Port:          healthPort,
			ContainerName: containerName,
		})
	}

	// Joiner Nodes - Q4 Stores (1)
	targets = append(targets, monitor.CheckTarget{
		Name:          "Joiner Q4 Stores",
		Host:          "joiner-node-q4-stores-1",
		Port:          healthPort,
		ContainerName: "joiner-node-q4-stores-1",
	})

	return targets
}
