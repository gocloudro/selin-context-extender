package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics
var (
	connectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_connections_total",
			Help: "Total number of WebSocket connections",
		},
		[]string{"status"},
	)
	activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ws_active_connections",
			Help: "Number of active WebSocket connections",
		},
	)
	messagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"type", "direction"},
	)
)

func init() {
	prometheus.MustRegister(connectionsTotal)
	prometheus.MustRegister(activeConnections)
	prometheus.MustRegister(messagesTotal)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking for security
		return true
	},
}

type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	hub      *Hub
	userID   string
	clientID string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id,omitempty"`
}

type StreamUpdate struct {
	RequestID string `json:"request_id"`
	Content   string `json:"content"`
	Status    string `json:"status"` // "streaming", "complete", "error"
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			activeConnections.Inc()
			connectionsTotal.WithLabelValues("connected").Inc()
			log.Printf("Client %s connected. Total clients: %d", client.clientID, len(h.clients))

			// Send welcome message
			welcome := Message{
				Type:      "welcome",
				Data:      "Connected to Selin WebSocket service",
				Timestamp: time.Now(),
			}
			if msg, err := json.Marshal(welcome); err == nil {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				activeConnections.Dec()
				connectionsTotal.WithLabelValues("disconnected").Inc()
				log.Printf("Client %s disconnected. Total clients: %d", client.clientID, len(h.clients))
			}

		case message := <-h.broadcast:
			messagesTotal.WithLabelValues("broadcast", "outbound").Inc()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
					activeConnections.Dec()
				}
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		messagesTotal.WithLabelValues("client", "inbound").Inc()

		// Handle client messages (like ping/pong, preferences updates)
		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			log.Printf("Received message from client %s: %s", c.clientID, msg.Type)
			// TODO: Handle different message types (e.g., preferences, subscriptions)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
			messagesTotal.WithLabelValues("server", "outbound").Inc()

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func generateClientID() string {
	nano := time.Now().UnixNano()
	return time.Now().Format("20060102150405") + "_" + fmt.Sprintf("%d", nano%100000)
}

func wsHandler(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}

	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		hub:      hub,
		userID:   userID,
		clientID: generateClientID(),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// Health endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "OK",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Readiness endpoint
func readyHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "READY",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	hub := newHub()
	go hub.run()

	// Setup HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(hub, w, r)
	})

	// Health and metrics endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler)
	mux.Handle("/metrics", promhttp.Handler())

	// Setup server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
		// Security timeouts
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("WebSocket service starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down WebSocket service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("WebSocket service stopped")
}
