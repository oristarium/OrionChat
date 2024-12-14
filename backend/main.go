package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Update struct {
	Message string `json:"message"`
	Lang    string `json:"lang"`
	Audio   string `json:"audio"`
}

type SSEClient chan string
var (
	clients    = make(map[SSEClient]bool)
	clientsMux sync.RWMutex
)

func main() {
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("static")))
	
	// SSE endpoint
	http.HandleFunc("/sse", handleSSE)
	
	// Update endpoint
	http.HandleFunc("/update", handleUpdate)

	// Redirect root to control
	http.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/control.html")
	})

	http.HandleFunc("/display", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/display.html")
	})

	log.Printf("Server starting on http://localhost:7777")
	log.Fatal(http.ListenAndServe(":7777", nil))
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := make(SSEClient)
	
	clientsMux.Lock()
	clients[client] = true
	clientsMux.Unlock()

	defer func() {
		clientsMux.Lock()
		delete(clients, client)
		clientsMux.Unlock()
	}()

	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		clientsMux.Lock()
		delete(clients, client)
		close(client)
		clientsMux.Unlock()
	}()

	for {
		msg, ok := <-client
		if !ok {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", msg)
		w.(http.Flusher).Flush()
	}
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast to all SSE clients
	message, _ := json.Marshal(update)
	
	clientsMux.RLock()
	for client := range clients {
		client <- string(message)
	}
	clientsMux.RUnlock()

	w.WriteHeader(http.StatusOK)
} 