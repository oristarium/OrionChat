package broadcast

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Update represents a message to be broadcasted
type Update struct {
	Type string     `json:"type"`
	Data interface{} `json:"data"`

}

// SSEClient represents a Server-Sent Events client
type SSEClient chan string

// Broadcaster handles Server-Sent Events broadcasting
type Broadcaster struct {
	clients map[SSEClient]bool
	mu      sync.RWMutex
}

// New creates a new Broadcaster instance
func New() *Broadcaster {
	return &Broadcaster{
		clients: make(map[SSEClient]bool),
	}
}

// HandleSSE handles SSE connections
func (b *Broadcaster) HandleSSE(w http.ResponseWriter, r *http.Request) {
	headers := map[string]string{
		"Content-Type":                "text/event-stream",
		"Cache-Control":               "no-cache",
		"Connection":                  "keep-alive",
		"Access-Control-Allow-Origin": "*",
	}
	
	for key, value := range headers {
		w.Header().Set(key, value)
	}

	client := make(SSEClient)
	
	b.addClient(client)

	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		b.removeClient(client)
	}()

	for msg := range client {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		w.(http.Flusher).Flush()
	}

	b.removeClient(client)
}

// Broadcast sends an update to all connected clients
func (b *Broadcaster) Broadcast(update Update) error {
	log.Println("Starting broadcast...")
	message, err := json.Marshal(update)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return err
	}
	
	log.Printf("Broadcasting message: %s", string(message))
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	for client := range b.clients {
		client <- string(message)
	}
	log.Printf("Broadcast completed to %d clients", len(b.clients))
	return nil
}

func (b *Broadcaster) addClient(client SSEClient) {
	b.mu.Lock()
	b.clients[client] = true
	b.mu.Unlock()
}

func (b *Broadcaster) removeClient(client SSEClient) {
	b.mu.Lock()
	if _, exists := b.clients[client]; exists {
		delete(b.clients, client)
		close(client)
	}
	b.mu.Unlock()
} 