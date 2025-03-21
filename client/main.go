package main

import (
	"bufio"
	"client/wiring"
	"fmt"
	"log"
	"os"
	"shared/utility"
)

func main() {

	// Default config
	configServerURL := "http://localhost:8080"
	configClientID := ""

	// Baca dari environment variable jika ada
	if serverURL := os.Getenv("SERVER_URL"); serverURL != "" {
		configServerURL = serverURL
	}

	// Baca client ID dari environment jika ada
	if cID := os.Getenv("CLIENT_ID"); cID != "" {
		configClientID = cID
	}

	fmt.Printf("Using server URL: %s\n", configServerURL)
	if configClientID != "" {
		fmt.Printf("Using client ID: %s\n", configClientID)
	}

	// Inisialisasi SSE client
	sseClient := utility.NewSSEClient(utility.SSEClientConfig{
		ServerURL: configServerURL,
		ClientID:  configClientID,
	})

	// gabung semua komponen
	wiring.SetupDependency(sseClient)

	// Mulai koneksi
	if err := sseClient.Connect(); err != nil {
		log.Fatalf("Gagal terhubung ke server: %v", err)
	}

	// Menunggu input dari user untuk keluar
	fmt.Println("Client is running. Press Enter to exit.")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	// Close SSE client
	sseClient.Close()
	fmt.Println("Client shutting down...")

}
