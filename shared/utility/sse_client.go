package utility

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSEClient adalah struct yang mengelola koneksi SSE dari sisi client
type SSEClient struct {
	serverURL    string
	clientID     string
	handlers     map[string][]EventHandlerFunc
	isConnected  bool
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	disconnected chan struct{}
}

// EventHandlerFunc adalah function signature untuk handler event
type EventHandlerFunc func(eventData []byte) error

// SSEClientConfig berisi konfigurasi untuk SSE client
type SSEClientConfig struct {
	ServerURL string
	ClientID  string // Optional, akan dibuat oleh server jika kosong
}

// NewSSEClient membuat instance baru SSEClient
func NewSSEClient(config SSEClientConfig) *SSEClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &SSEClient{
		serverURL:    config.ServerURL,
		clientID:     config.ClientID,
		handlers:     make(map[string][]EventHandlerFunc),
		isConnected:  false,
		ctx:          ctx,
		cancel:       cancel,
		disconnected: make(chan struct{}),
	}
}

// AddEventHandler menambahkan handler untuk event tertentu
func (c *SSEClient) AddEventHandler(eventType string, handler EventHandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handlers[eventType] == nil {
		c.handlers[eventType] = []EventHandlerFunc{}
	}

	c.handlers[eventType] = append(c.handlers[eventType], handler)
}

// Connect membuat koneksi ke SSE server
func (c *SSEClient) Connect() error {
	c.mu.Lock()
	if c.isConnected {
		c.mu.Unlock()
		return nil // Sudah terhubung
	}
	c.mu.Unlock()

	return c.connectWithRetry(10, 1*time.Second)
}

// connectWithRetry mencoba koneksi dengan backoff eksponensial
func (c *SSEClient) connectWithRetry(maxRetries int, initialBackoff time.Duration) error {
	var err error
	retryCount := 0
	backoff := initialBackoff

	for retryCount < maxRetries {
		err = c.establishConnection()
		if err == nil {
			return nil // Koneksi berhasil
		}

		retryCount++
		fmt.Printf("Koneksi gagal (attempt %d/%d): %v. Mencoba kembali dalam %v...\n",
			retryCount, maxRetries, err, backoff)

		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		case <-time.After(backoff):
			// Tingkatkan backoff untuk percobaan berikutnya
			backoff *= 2
			if backoff > 1*time.Minute {
				backoff = 1 * time.Minute
			}
		}
	}

	return fmt.Errorf("tidak dapat terhubung setelah %d percobaan: %v", maxRetries, err)
}

// establishConnection membuat koneksi ke server SSE
func (c *SSEClient) establishConnection() error {
	var sseURL string
	if c.clientID != "" {
		sseURL = fmt.Sprintf("%s/api/sse/connect?client_id=%s", c.serverURL, c.clientID)
	} else {
		sseURL = fmt.Sprintf("%s/api/sse/connect", c.serverURL)
	}

	fmt.Printf("Menghubungkan ke SSE endpoint: %s\n", sseURL)

	req, err := http.NewRequestWithContext(c.ctx, "GET", sseURL, nil)
	if err != nil {
		return fmt.Errorf("error membuat request: %v", err)
	}

	client := &http.Client{
		Timeout: 0, // Tidak ada timeout untuk koneksi SSE
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error menghubungi server: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("server mengembalikan status non-OK: %d", resp.StatusCode)
	}

	// Update status koneksi
	c.mu.Lock()
	c.isConnected = true
	c.mu.Unlock()

	fmt.Println("Koneksi SSE berhasil dibuat")

	// Start goroutine untuk membaca events
	go c.readEvents(resp)

	return nil
}

// readEvents membaca event dari respons SSE
func (c *SSEClient) readEvents(resp *http.Response) {
	defer resp.Body.Close()
	defer c.handleDisconnect()

	scanner := bufio.NewScanner(resp.Body)
	var eventType string
	var eventData string

	for scanner.Scan() {
		select {
		case <-c.ctx.Done():
			return
		default:
			line := scanner.Text()

			// Skip keepalive comments
			if strings.HasPrefix(line, ":") {
				continue
			}

			// Parse event type
			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				eventData = strings.TrimPrefix(line, "data: ")
			} else if line == "" && eventType != "" && eventData != "" {
				// Event complete, proses
				c.processEvent(eventType, eventData)
				eventType = ""
				eventData = ""
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error membaca event: %v\n", err)
	}
}

// processEvent memproses event dari server
func (c *SSEClient) processEvent(eventType, eventData string) {
	// Khusus untuk event connected, simpan clientID
	if eventType == "connected" {
		var connectEvent struct {
			ClientID string `json:"client_id"`
		}
		if err := json.Unmarshal([]byte(eventData), &connectEvent); err == nil {
			c.mu.Lock()
			c.clientID = connectEvent.ClientID
			c.mu.Unlock()
			fmt.Printf("Terhubung dengan client ID: %s\n", connectEvent.ClientID)
		}
	}

	// Panggil semua handler untuk event ini
	c.mu.RLock()
	handlers, exists := c.handlers[eventType]
	c.mu.RUnlock()

	if !exists {
		fmt.Printf("Menerima event tanpa handler: %s\n", eventType)
		return
	}

	for _, handler := range handlers {
		if err := handler([]byte(eventData)); err != nil {
			fmt.Printf("Error pada handler untuk event %s: %v\n", eventType, err)
		}
	}
}

// handleDisconnect menangani saat koneksi terputus
func (c *SSEClient) handleDisconnect() {
	c.mu.Lock()
	c.isConnected = false
	c.mu.Unlock()

	fmt.Println("Koneksi SSE terputus")
	close(c.disconnected)
}

// IsConnected mengembalikan status koneksi
func (c *SSEClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// GetClientID mengembalikan clientID
func (c *SSEClient) GetClientID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.clientID
}

// WaitForDisconnect menunggu hingga koneksi terputus
func (c *SSEClient) WaitForDisconnect() {
	<-c.disconnected
}

// Close menutup koneksi SSE client
func (c *SSEClient) Close() {
	c.cancel()
}
