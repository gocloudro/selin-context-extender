package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

	var response HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response.Status != "OK" {
		t.Errorf("expected status OK, got %s", response.Status)
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

	var response HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response.Status != "READY" {
		t.Errorf("expected status READY, got %s", response.Status)
	}
}

func TestQueryHandler(t *testing.T) {
	reqBody := QueryRequest{
		Prompt: "Hello, world!",
		UserID: "test_user",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/query", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(queryHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response QueryResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response.Response == "" {
		t.Error("expected non-empty response")
	}
}

func TestQueryHandlerInvalidMethod(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/query", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(queryHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestQueryHandlerEmptyPrompt(t *testing.T) {
	reqBody := QueryRequest{
		Prompt: "",
		UserID: "test_user",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/query", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(queryHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}
