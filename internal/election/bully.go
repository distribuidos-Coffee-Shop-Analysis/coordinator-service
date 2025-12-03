package election

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const (
	electionPort       = "12340"
	timeout            = 2 * time.Second
	heartbeatInterval  = 2 * time.Second
	electionTimeout    = 6 * time.Second
	
	// Protocol messages
	msgElection = "ELECTION"
	msgOK       = "OK"
	msgLeader   = "LEADER"
)

// Coordinator represents a coordinator node in the election
type Coordinator struct {
	myID              int
	totalReplicas     int
	isLeader          bool
	leaderID          int
	mu                sync.RWMutex
	leaderChan        chan bool
	lastHeartbeat     time.Time
	heartbeatMu       sync.RWMutex
	stopHeartbeat     chan bool
}

// NewCoordinator creates a new coordinator for Bully election
func NewCoordinator(myID, totalReplicas int) *Coordinator {
	return &Coordinator{
		myID:          myID,
		totalReplicas: totalReplicas,
		isLeader:      false,
		leaderID:      -1,
		leaderChan:    make(chan bool, 10),
		lastHeartbeat: time.Now(),
		stopHeartbeat: make(chan bool, 1),
	}
}

// Start begins the election process and TCP server
func (c *Coordinator) Start() {
	log.Printf("Starting Bully election: MY_ID=%d, TOTAL_REPLICAS=%d", c.myID, c.totalReplicas)
	
	// Start TCP server to receive election messages
	go c.startServer()
	
	// Start election timeout monitor
	go c.monitorElectionTimeout()
	
	// Wait a bit for all coordinators to start their servers
	time.Sleep(2 * time.Second)
	
	// Start initial election
	go c.startElection()
}

// startServer starts TCP server to receive election messages
func (c *Coordinator) startServer() {
	listener, err := net.Listen("tcp", "0.0.0.0:"+electionPort)
	if err != nil {
		log.Fatalf("Failed to start election server: %v", err)
	}
	defer listener.Close()
	
	log.Printf("Election server listening on port %s", electionPort)
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		
		go c.handleConnection(conn)
	}
}

// handleConnection handles incoming election messages
func (c *Coordinator) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		if err != io.EOF {
			log.Printf("Error reading message: %v", err)
		}
		return
	}
	
	message := string(buffer[:n])
	
	switch message {
	case msgElection:
		// Someone with lower ID is asking for election
		log.Printf("Received ELECTION message, responding with OK")
		conn.Write([]byte(msgOK))
		
		c.mu.RLock()
		isLeader := c.isLeader
		c.mu.RUnlock()
		
		// If I'm the leader, immediately send LEADER message to reaffirm authority
		if isLeader {
			log.Printf("I'm the leader, sending LEADER message to reaffirm")
			// Send LEADER message to all nodes
			go c.broadcastLeadership()
		} else {
			// Start our own election if we're not already leader
			go c.startElection()
		}
		
	case msgOK:
		// Someone with higher ID responded, they will handle it
		log.Printf("Received OK message, higher ID node will handle election")
		
	case msgLeader:
		// New leader announcement (heartbeat)
		log.Printf("Received LEADER heartbeat")
		
		// Reset heartbeat timer
		c.heartbeatMu.Lock()
		c.lastHeartbeat = time.Now()
		c.heartbeatMu.Unlock()
		
		c.mu.Lock()
		wasLeader := c.isLeader
		// Update leader ID if we don't know who the leader is
		if c.leaderID == -1 {
			c.leaderID = c.myID + 1 // Assume it's from a higher ID
		}
		c.isLeader = false
		c.mu.Unlock()
		
		if wasLeader {
			log.Printf("Lost leadership")
			c.leaderChan <- false
		}
		
	default:
		log.Printf("Unknown message: %s", message)
	}
}

// startElection initiates the Bully election algorithm
func (c *Coordinator) startElection() {
	log.Printf("Starting election process")
	
	// Send ELECTION to all nodes with higher IDs
	receivedOK := false
	
	for id := c.myID + 1; id <= c.totalReplicas; id++ {
		if c.sendMessage(id, msgElection) {
			receivedOK = true
		}
	}
	
	if receivedOK {
		// Higher ID node responded, they will handle leadership
		log.Printf("Higher ID node responded, waiting for leader announcement")
		// Don't do anything - the heartbeat monitor will detect if no leader emerges
	} else {
		// No higher ID responded, become leader
		c.becomeLeader()
	}
}

// becomeLeader makes this node the leader
func (c *Coordinator) becomeLeader() {
	c.mu.Lock()
	wasLeader := c.isLeader
	c.isLeader = true
	c.leaderID = c.myID
	c.mu.Unlock()
	
	log.Printf("*** I AM THE LEADER (ID=%d) ***", c.myID)
	
	// Announce leadership to all other nodes
	c.broadcastLeadership()
	
	// Start heartbeat loop
	go c.sendHeartbeats()
	
	// Notify main loop if we just became leader
	if !wasLeader {
		c.leaderChan <- true
	}
}

// broadcastLeadership sends LEADER message to all other nodes
func (c *Coordinator) broadcastLeadership() {
	for id := 1; id <= c.totalReplicas; id++ {
		if id != c.myID {
			c.sendMessage(id, msgLeader)
		}
	}
}

// sendHeartbeats periodically sends LEADER messages while this node is the leader
func (c *Coordinator) sendHeartbeats() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	
	log.Printf("Starting heartbeat broadcasts (every %v)", heartbeatInterval)
	
	for {
		select {
		case <-ticker.C:
			c.mu.RLock()
			isLeader := c.isLeader
			c.mu.RUnlock()
			
			if !isLeader {
				log.Printf("No longer leader, stopping heartbeats")
				return
			}
			
			// Send heartbeat to all followers
			for id := 1; id <= c.totalReplicas; id++ {
				if id != c.myID {
					c.sendMessage(id, msgLeader)
				}
			}
			
		case <-c.stopHeartbeat:
			log.Printf("Heartbeat stopped")
			return
		}
	}
}

// monitorElectionTimeout monitors if we haven't received heartbeats and starts election
func (c *Coordinator) monitorElectionTimeout() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.RLock()
		isLeader := c.isLeader
		c.mu.RUnlock()
		
		// Only followers check for election timeout
		if !isLeader {
			c.heartbeatMu.RLock()
			timeSinceLastHeartbeat := time.Since(c.lastHeartbeat)
			c.heartbeatMu.RUnlock()
			
			if timeSinceLastHeartbeat > electionTimeout {
				log.Printf("Election timeout: no heartbeat for %v, starting election", timeSinceLastHeartbeat)
				
				// Reset heartbeat timer to avoid multiple elections
				c.heartbeatMu.Lock()
				c.lastHeartbeat = time.Now()
				c.heartbeatMu.Unlock()
				
				// Reset leader ID
				c.mu.Lock()
				c.leaderID = -1
				c.mu.Unlock()
				
				go c.startElection()
			}
		}
	}
}

// sendMessage sends a message to a specific coordinator
func (c *Coordinator) sendMessage(targetID int, message string) bool {
	hostname := fmt.Sprintf("coordinator-%d", targetID)
	address := net.JoinHostPort(hostname, electionPort)
	
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		// Node is down or unreachable
		return false
	}
	defer conn.Close()
	
	_, err = conn.Write([]byte(message))
	if err != nil {
		return false
	}
	
	// For ELECTION messages, wait for OK response
	if message == msgElection {
		conn.SetReadDeadline(time.Now().Add(timeout))
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			return false
		}
		
		response := string(buffer[:n])
		return response == msgOK
	}
	
	return true
}

// IsLeader returns whether this node is currently the leader
func (c *Coordinator) IsLeader() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isLeader
}

// LeaderChan returns the channel that signals leadership changes
func (c *Coordinator) LeaderChan() <-chan bool {
	return c.leaderChan
}

// GetLeaderID returns the current leader ID
func (c *Coordinator) GetLeaderID() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.leaderID
}


