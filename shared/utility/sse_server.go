package utility

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Client represents a single SSE client connection
type Client struct {
	ID string // Added client identifier
	w  http.ResponseWriter
	f  http.Flusher
	mu sync.Mutex
	// Add done channel for cleanup
	done chan struct{}
}

// SSE represents the SSE server
type SSEServer struct {
	clients          map[string]*Client // Changed to use string keys
	mu               sync.RWMutex       // Single mutex for the SSE struct
	maxConns         int                // Maximum allowed connections
	keepAlive        time.Duration      // Keepalive interval
	origins          []string           // Allowed CORS origins
	broadcastTimeout time.Duration      // Timeout for broadcast operations
	logger           *log.Logger        // Logger for SSE server
}

// SSEConfig holds configuration for the SSE server
type SSEConfig struct {
	MaxConnections   int
	KeepAlive        time.Duration
	Origins          []string // Allowed CORS origins
	BroadcastTimeout time.Duration
	Logger           *log.Logger
}

// NewSSEDefault creates a new SSE instance with default configuration
func NewSSEDefault() *SSEServer {
	return NewSSEServer(SSEConfig{})
}

// NewSSE creates a new SSE instance with configuration
func NewSSEServer(config SSEConfig) *SSEServer {
	if config.MaxConnections <= 0 {
		config.MaxConnections = 10000 // Default max connections
	}
	if config.KeepAlive <= 0 {
		config.KeepAlive = 10 * time.Second // Default keepalive
	}
	if config.BroadcastTimeout <= 0 {
		config.BroadcastTimeout = 5 * time.Second // Default broadcast timeout
	}
	if config.Logger == nil {
		config.Logger = log.New(log.Writer(), "[SSE] ", log.LstdFlags)
	}

	return &SSEServer{
		clients:          make(map[string]*Client),
		maxConns:         config.MaxConnections,
		keepAlive:        config.KeepAlive,
		origins:          config.Origins,
		broadcastTimeout: config.BroadcastTimeout,
		logger:           config.Logger,
	}
}

// Message represents an SSE message with both SSE-standard and embedded formats
type Message struct {
	// Internal structure for JSON data
	EventType string `json:"event_type"`
	Data      any    `json:"data"`
}

// enableCors enables CORS for the response with proper origin validation
func enableCors(w http.ResponseWriter, origins []string, requestOrigin string) {
	// Default to strict CORS if no origins specified
	if len(origins) == 0 {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		return
	}

	// Check if the request origin is in the allowed list
	for _, allowedOrigin := range origins {
		if allowedOrigin == requestOrigin || allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", requestOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			return
		}
	}

	// If we get here, the origin wasn't in the allow list
	w.Header().Set("Access-Control-Allow-Origin", origins[0])
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// validateMessage validates a message for required fields
func (s *SSEServer) validateMessage(msg Message) error {
	if msg.EventType == "" {
		return fmt.Errorf("invalid message: eventType cannot be empty")
	}
	if msg.Data == nil {
		return fmt.Errorf("invalid message: data cannot be nil")
	}
	return nil
}

// addClient adds a client to the SSE instance
func (s *SSEServer) addClient(client *Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check max connections
	if len(s.clients) >= s.maxConns {
		return fmt.Errorf("maximum connections (%d) reached", s.maxConns)
	}

	s.clients[client.ID] = client
	return nil
}

// removeClient removes a client from the SSE instance
func (s *SSEServer) removeClient(clientID string) {
	var client *Client
	var exists bool

	s.mu.Lock()
	client, exists = s.clients[clientID]
	if exists {
		delete(s.clients, clientID)
	}
	s.mu.Unlock()

	if exists {
		close(client.done)
		s.logger.Printf("Client %s disconnected", clientID)
	}
}

// Note: Removed the unused disconnectClient method

// GetConnectedClientIDs returns a list of connected client IDs
func (s *SSEServer) GetConnectedClientIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.clients))
	for id := range s.clients {
		ids = append(ids, id)
	}
	return ids
}

// IsClientConnected checks if a client is connected
func (s *SSEServer) IsClientConnected(clientID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.clients[clientID]
	return exists
}

// GetConnectedClientCount returns the number of connected clients
func (s *SSEServer) GetConnectedClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.clients)
}

// SendToClients sends a message to specific clients or all clients if clientIDs is empty
func (s *SSEServer) SendToClients(ctx context.Context, msg Message, clientIDs ...string) error {
	// Validate message
	if err := s.validateMessage(msg); err != nil {
		return err
	}

	// Marshal the message data to JSON (do this once for all clients)
	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal message data: %w", err)
	}

	// Determine if this is a broadcast or targeted message
	isBroadcast := len(clientIDs) == 0

	// Get list of clients to send to
	var clients []*Client

	s.mu.RLock()
	if isBroadcast {
		// Get all clients for a broadcast
		clients = make([]*Client, 0, len(s.clients))
		for _, client := range s.clients {
			clients = append(clients, client)
		}
	} else {
		// Get only the specified clients
		clients = make([]*Client, 0, len(clientIDs))
		for _, id := range clientIDs {
			if client, exists := s.clients[id]; exists {
				clients = append(clients, client)
			}
		}
	}
	s.mu.RUnlock()

	// If no clients found, handle accordingly
	if len(clients) == 0 {
		if isBroadcast {
			return nil // No clients to broadcast to, not an error
		}
		return fmt.Errorf("no clients found from the specified IDs")
	}

	// Use a timeout context for the operation
	sendCtx, cancel := context.WithTimeout(ctx, s.broadcastTimeout)
	defer cancel()

	// Helper function to send message to a single client
	sendToClient := func(client *Client) error {
		client.mu.Lock()
		defer client.mu.Unlock()

		_, err := fmt.Fprintf(client.w, "event: %s\ndata: %s\n\n", msg.EventType, dataBytes)
		if err != nil {
			return err
		}
		client.f.Flush()
		return nil
	}

	// For a single client, handle synchronously for simplicity
	if len(clients) == 1 {
		select {
		case <-sendCtx.Done():
			return sendCtx.Err()
		default:
			err := sendToClient(clients[0])
			if err != nil {
				s.logger.Printf("Failed to send to client %s: %v", clients[0].ID, err)
				s.removeClient(clients[0].ID)
				return err
			}
			return nil
		}
	}

	// For multiple clients, handle concurrently
	var wg sync.WaitGroup
	errors := make(chan error, len(clients))

	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()

			select {
			case <-sendCtx.Done():
				errors <- sendCtx.Err()
				return
			default:
				err := sendToClient(c)
				if err != nil {
					s.logger.Printf("Failed to send to client %s: %v", c.ID, err)
					s.removeClient(c.ID)
					errors <- err
				}
			}
		}(client)
	}

	wg.Wait()
	close(errors)

	// Collect errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		if isBroadcast {
			return fmt.Errorf("failed to broadcast to %d/%d clients: %v",
				len(errs), len(clients), errs[0])
		}
		return fmt.Errorf("failed to send to %d/%d specified clients: %v",
			len(errs), len(clients), errs[0])
	}

	return nil
}

// setupClientConnection creates and initializes a new client connection
func (s *SSEServer) setupClientConnection(w http.ResponseWriter, r *http.Request) (*Client, error) {
	// Check if client supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}

	// Get client ID from query parameter or generate a new one
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("client-%d", time.Now().UnixNano())
	}

	// Create new client
	client := &Client{
		ID:   clientID,
		w:    w,
		f:    flusher,
		done: make(chan struct{}),
	}

	// Add client to broadcast list
	if err := s.addClient(client); err != nil {
		return nil, err
	}

	return client, nil
}

// sendConnectedEvent sends the initial connected event to a client
func (s *SSEServer) sendConnectedEvent(client *Client) error {
	// Create connected message
	connectMsg := Message{
		EventType: "connected",
		Data:      map[string]string{"client_id": client.ID},
	}

	// Use a background context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Send the connected event using SendToClients
	err := s.SendToClients(ctx, connectMsg, client.ID)

	if err != nil {
		return fmt.Errorf("failed to send connected event: %w", err)
	}

	s.logger.Printf("Client %s connected", client.ID)
	return nil
}

// startKeepalive starts the keepalive goroutine for a client
func (s *SSEServer) startKeepalive(client *Client, ctx context.Context) {
	ticker := time.NewTicker(s.keepAlive)
	defer ticker.Stop()

	for {
		select {
		case <-client.done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			client.mu.Lock()
			fmt.Fprintf(client.w, ": keepalive\n\n")
			client.f.Flush()
			client.mu.Unlock()
		}
	}
}

// HandleSSE handles the SSE connection
func (s *SSEServer) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Handle OPTIONS request for CORS
	if r.Method == "OPTIONS" {
		enableCors(w, s.origins, r.Header.Get("Origin"))
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	enableCors(w, s.origins, r.Header.Get("Origin"))

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Setup client connection
	client, err := s.setupClientConnection(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer s.removeClient(client.ID)

	// Send connected event
	if err := s.sendConnectedEvent(client); err != nil {
		s.logger.Printf("Failed to send connected event: %v", err)
		return
	}

	// Start keepalive goroutine
	go s.startKeepalive(client, r.Context())

	// Wait for client disconnect
	select {
	case <-r.Context().Done():
		s.logger.Printf("Client %s connection context done: %v", client.ID, r.Context().Err())
	case <-client.done:
		s.logger.Printf("Client %s connection closed", client.ID)
	}
}
