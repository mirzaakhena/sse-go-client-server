package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ClientConnection mengelola koneksi SSE untuk setiap client
type ClientConnection struct {
	ID        string
	Channel   chan []byte
	Connected bool
}

// ServerState mengelola state server dan koneksi client
type ServerState struct {
	Clients map[string]*ClientConnection
	mu      sync.RWMutex
}

// ProcessResult struktur untuk menyimpan hasil proses dari client
type ProcessResult struct {
	ClientID string  `json:"client_id"`
	Values   []int64 `json:"values"`
}

// RequestPayload struktur untuk payload yang dikirim ke client
type RequestPayload struct {
	Count int `json:"count"`
}

// NewServerState inisialisasi server state
func NewServerState() *ServerState {
	return &ServerState{
		Clients: make(map[string]*ClientConnection),
	}
}

var serverState = NewServerState()

func main() {
	// Menangani route untuk SSE handshake
	http.HandleFunc("/api/sse/connect", handleSSEConnect)

	// Menangani route untuk memicu request ke client
	http.HandleFunc("/api/trigger-request", handleTriggerRequest)

	// Menangani route untuk menerima hasil proses dari client
	http.HandleFunc("/api/collect-result", handleCollectResult)

	// Default route
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Server is running")
	})

	port := 8080
	fmt.Printf("Server started at http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))

	// TODO: Implement HTTPS support
	// Generate self-signed certificate:
	// openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
	// Then use:
	// log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", port), "cert.pem", "key.pem", nil))
}

// handleSSEConnect menangani permintaan untuk membuka koneksi SSE
func handleSSEConnect(w http.ResponseWriter, r *http.Request) {
	// Mengatur header untuk SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Flush header segera
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	// Membuat ID client unik
	clientID := time.Now().UnixNano()
	clientIDStr := strconv.FormatInt(clientID, 10)

	// Membuat channel untuk client
	messageChan := make(chan []byte)

	// Register client
	serverState.mu.Lock()
	serverState.Clients[clientIDStr] = &ClientConnection{
		ID:        clientIDStr,
		Channel:   messageChan,
		Connected: true,
	}
	serverState.mu.Unlock()

	// Kirim ID client sebagai konfirmasi handshake
	clientInfo := map[string]string{"client_id": clientIDStr}
	clientInfoJSON, _ := json.Marshal(clientInfo)
	fmt.Fprintf(w, "event: connected\ndata: %s\n\n", clientInfoJSON)
	flusher.Flush()

	log.Printf("Client connected: %s", clientIDStr)

	// Hapus client ketika koneksi ditutup
	closeNotify := r.Context().Done()
	go func() {
		<-closeNotify
		serverState.mu.Lock()
		delete(serverState.Clients, clientIDStr)
		serverState.mu.Unlock()
		close(messageChan)
		log.Printf("Client disconnected: %s", clientIDStr)
	}()

	// Kirim pesan ke client ketika tersedia
	for msg := range messageChan {
		fmt.Fprintf(w, "event: request\ndata: %s\n\n", msg)
		flusher.Flush()
	}
}

// handleTriggerRequest menangani permintaan untuk trigger request ke client
func handleTriggerRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse parameter count dari query string
	countStr := r.URL.Query().Get("count")
	if countStr == "" {
		countStr = "1" // Default value
	}

	count, err := strconv.Atoi(countStr)
	if err != nil || count < 1 {
		http.Error(w, "Invalid count parameter", http.StatusBadRequest)
		return
	}

	// Parse clientID dari query string
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "Missing client_id parameter", http.StatusBadRequest)
		return
	}

	// Cek apakah client dengan ID tersebut terhubung
	serverState.mu.RLock()
	client, exists := serverState.Clients[clientID]
	serverState.mu.RUnlock()

	if !exists || !client.Connected {
		http.Error(w, "Client not connected", http.StatusBadRequest)
		return
	}

	// Buat payload untuk dikirim ke client
	payload := RequestPayload{
		Count: count,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Error creating payload", http.StatusInternalServerError)
		return
	}

	// Kirim payload ke client melalui SSE
	client.Channel <- payloadJSON

	// Kirim response ke caller
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Request sent to client %s with count %d", clientID, count),
	}
	json.NewEncoder(w).Encode(response)
}

// handleCollectResult menangani penerimaan hasil dari client
func handleCollectResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode body request
	var result ProcessResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log hasil yang diterima
	log.Printf("Received result from client %s: %v", result.ClientID, result.Values)

	// Kirim respons sukses
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "Result received successfully",
		"data":    result,
	}
	json.NewEncoder(w).Encode(response)
}