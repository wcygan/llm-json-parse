package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/wcygan/llm-json-parse/internal/client"
	"github.com/wcygan/llm-json-parse/internal/server"
)

func main() {
	llmServerURL := os.Getenv("LLM_SERVER_URL")
	if llmServerURL == "" {
		llmServerURL = "http://localhost:8080"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	llmClient := client.NewLlamaServerClient(llmServerURL)
	srv := server.NewServer(llmClient)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	addr := ":" + port
	fmt.Printf("Server starting on %s\n", addr)
	fmt.Printf("LLM Server URL: %s\n", llmServerURL)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
