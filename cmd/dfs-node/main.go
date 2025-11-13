package main

import (
	"fmt"
	"net/http"

	"github.com/timskillet/distributed-filestore/internal/node"
)

func main() {
	// TODO: Load nodes from DynamoDB
	nodes := []struct {
		ID      string
		Address string
		Port    string
	}{
		{"nodeA", "http://localhost", "8081"},
		{"nodeB", "http://localhost", "8082"},
	}

	for _, n := range nodes {
		mux := http.NewServeMux()
		mux.HandleFunc("/store-chunk", node.HandleStoreChunk(n.ID))
		mux.HandleFunc("/get-chunk", node.HandleGetChunk(n.ID))
		go func(port string, m *http.ServeMux) {
			fmt.Println("Starting mock node on port", port)
			http.ListenAndServe(":"+port, m)
		}(n.Port, mux)
	}

	select {}
}
