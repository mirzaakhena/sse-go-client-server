package main

import (
	"fmt"
	"log"
	"net/http"
	"server/model"
	"server/wiring"
	"shared/utility"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {

	// Konfigurasi SSE
	// TODO put into env
	sseConfig := utility.SSEConfig{
		MaxConnections: 1000,
		KeepAlive:      15 * time.Second,
		Origins:        []string{"*"}, // Untuk development, bisa lebih spesifik untuk production
	}

	// TODO put into env
	// TODO change into proper database later
	db, err := gorm.Open(sqlite.Open("network_scanner.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&model.Client{})

	// Inisialisasi SSE server
	sseServer := utility.NewSSEServer(sseConfig)

	// inisialisasi HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("GET  /api/sse/connect", sseServer.HandleSSE)

	apiPrinter := utility.NewApiPrinter()

	// gabung semua komponen
	wiring.SetupDependency(mux, sseServer, apiPrinter, db)

	// TODO put into env
	port := 8080

	// Print API ke console dan openapi
	apiPrinter.
		PrintAPIDataTable().
		PublishAPI(mux, fmt.Sprintf("http://localhost:%d", port), "/openapi")

	// Default route
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Server is running")
	})

	// start server
	fmt.Printf("Server started at http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), mux))

}
