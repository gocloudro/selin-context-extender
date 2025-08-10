package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response["status"] != "OK" {
		t.Errorf("expected status OK, got %s", response["status"])
	}
}

func TestReadyHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ready", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(readyHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response["status"] != "READY" {
		t.Errorf("expected status READY, got %s", response["status"])
	}
}

func TestHubOperations(t *testing.T) {
	hub := newHub()

	// Test hub initialization
	if hub.clients == nil {
		t.Error("hub clients map should be initialized")
	}

	if hub.broadcast == nil {
		t.Error("hub broadcast channel should be initialized")
	}

	if hub.register == nil {
		t.Error("hub register channel should be initialized")
	}

	if hub.unregister == nil {
		t.Error("hub unregister channel should be initialized")
	}
}

func TestWebSocketHandler(t *testing.T) {
	hub := newHub()
	go hub.run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsHandler(hub, w, r)
	}))
	defer server.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.1
	u := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("Could not open a ws connection on %s: %v", u, err)
	}
	defer ws.Close()

	// Read welcome message
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Could not read message: %v", err)
	}

	var welcomeMsg Message
	if err := json.Unmarshal(message, &welcomeMsg); err != nil {
		t.Fatalf("Could not unmarshal welcome message: %v", err)
	}

	if welcomeMsg.Type != "welcome" {
		t.Errorf("Expected welcome message, got %s", welcomeMsg.Type)
	}

	// Test sending a message
	testMsg := Message{
		Type:      "test",
		Data:      "Hello, WebSocket!",
		Timestamp: time.Now(),
	}

	msgBytes, _ := json.Marshal(testMsg)
	if err := ws.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		t.Fatalf("Could not send message: %v", err)
	}
}

func TestClientIDGeneration(t *testing.T) {
	id1 := generateClientID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	id2 := generateClientID()

	if id1 == id2 {
		t.Error("Client IDs should be unique")
	}

	if len(id1) == 0 {
		t.Error("Client ID should not be empty")
	}
}
