package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config berisi konfigurasi client
type Config struct {
	ServerURL string `json:"server_url"`
}

// RequestPayload struktur untuk payload yang diterima dari server
type RequestPayload struct {
	Count int `json:"count"`
}

// ProcessResult struktur untuk hasil proses yang dikirim ke server
type ProcessResult struct {
	ClientID string  `json:"client_id"`
	Values   []int64 `json:"values"`
}

// ConnectEvent struktur untuk event koneksi
type ConnectEvent struct {
	ClientID string `json:"client_id"`
}

var clientID string
var config Config

func main() {
	// Inisialisasi random seed
	rand.Seed(time.Now().UnixNano())

	// Load konfigurasi
	loadConfig()

	// Mulai koneksi SSE ke server
	go connectToSSE()

	// Menunggu input dari user untuk keluar
	fmt.Println("Client is running. Press Enter to exit.")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	fmt.Println("Client shutting down...")
}

// loadConfig memuat konfigurasi dari file atau environment
func loadConfig() {
	// Default config
	config.ServerURL = "http://localhost:8080"

	// Baca dari environment variable jika ada
	if serverURL := os.Getenv("SERVER_URL"); serverURL != "" {
		config.ServerURL = serverURL
	}

	fmt.Printf("Using server URL: %s\n", config.ServerURL)
}

// connectToSSE membuat koneksi SSE ke server
func connectToSSE() {
	sseURL := fmt.Sprintf("%s/api/sse/connect", config.ServerURL)
	fmt.Printf("Connecting to SSE endpoint: %s\n", sseURL)

	// Buat request untuk koneksi SSE
	req, err := http.NewRequest("GET", sseURL, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	// Buat client HTTP
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	defer resp.Body.Close()

	// Cek respons
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server returned non-OK status: %d", resp.StatusCode)
	}

	fmt.Println("SSE connection established")

	// Baca event dari response
	scanner := bufio.NewScanner(resp.Body)
	var eventType, eventData string

	for scanner.Scan() {
		line := scanner.Text()

		// Parse event
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			eventData = strings.TrimPrefix(line, "data: ")
		} else if line == "" && eventType != "" && eventData != "" {
			// Event complete, process it
			handleEvent(eventType, eventData)
			eventType = ""
			eventData = ""
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("SSE connection closed: %v", err)
	}
}

// handleEvent menangani event dari server
func handleEvent(eventType, eventData string) {
	fmt.Printf("Received event: %s\n", eventType)

	switch eventType {
	case "connected":
		var connectEvent ConnectEvent
		if err := json.Unmarshal([]byte(eventData), &connectEvent); err != nil {
			log.Printf("Error parsing connect event: %v", err)
			return
		}
		clientID = connectEvent.ClientID
		fmt.Printf("Connected with client ID: %s\n", clientID)

	case "request":
		var payload RequestPayload
		if err := json.Unmarshal([]byte(eventData), &payload); err != nil {
			log.Printf("Error parsing request payload: %v", err)
			return
		}
		fmt.Printf("Received request with count: %d\n", payload.Count)

		// Process request in a goroutine
		go processRequest(payload)
	}
}

// processRequest memproses request dari server
func processRequest(payload RequestPayload) {
	fmt.Printf("Processing request with count: %d\n", payload.Count)
	fmt.Println("Processing will take 5 seconds...")

	// Simulasi proses yang memakan waktu
	time.Sleep(5 * time.Second)

	// Generate random values sebanyak count
	values := make([]int64, payload.Count)
	for i := 0; i < payload.Count; i++ {
		values[i] = rand.Int63n(1000)
	}

	// Buat result
	result := ProcessResult{
		ClientID: clientID,
		Values:   values,
	}

	// Kirim result ke server
	sendResultToServer(result)
}

// sendResultToServer mengirim hasil proses ke server
func sendResultToServer(result ProcessResult) {
	url := fmt.Sprintf("%s/api/collect-result", config.ServerURL)

	// Marshal result ke JSON
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error marshaling result: %v", err)
		return
	}

	// Kirim POST request ke server
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error sending result to server: %v", err)
		return
	}
	defer resp.Body.Close()

	// Cek respons
	if resp.StatusCode != http.StatusOK {
		log.Printf("Server returned non-OK status: %d", resp.StatusCode)
		return
	}

	fmt.Println("Result sent to server successfully")

	// Log values yang dikirim
	fmt.Printf("Sent values: %v\n", result.Values)
}