package main

import (
	"fmt"
	"lucid-platform/api"
	"lucid-platform/db"
	"net/http"
)

func main() {

	db.InitDB()

	// Route incoming traffic on /webhook to our handler function
	http.HandleFunc("/webhook", api.HandleWebhook)

	port := ":8080"
	fmt.Printf("🚀 Lucid Platform API is running on http://localhost%s\n", port)

	// Start the server
	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
