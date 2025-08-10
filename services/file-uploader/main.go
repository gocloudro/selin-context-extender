package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type UploadResponse struct {
	Success        bool     `json:"success"`
	Message        string   `json:"message"`
	FileID         string   `json:"file_id,omitempty"`
	Filename       string   `json:"filename,omitempty"`
	FileType       string   `json:"file_type,omitempty"`
	ProcessedItems int      `json:"processed_items,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

type SlackMessage struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
	Channel   string `json:"channel,omitempty"`
	Thread    string `json:"thread_ts,omitempty"`
}

type SlackExport struct {
	Messages []SlackMessage `json:"messages,omitempty"`
	Users    []SlackUser    `json:"users,omitempty"`
	Channels []SlackChannel `json:"channels,omitempty"`
}

type SlackUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Real string `json:"real_name"`
}

type SlackChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func main() {
	log.Println("🚀 Starting File Uploader Service...")

	// Create upload directory
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// Setup routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)
	http.HandleFunc("/upload/slack", slackUploadHandler)
	http.HandleFunc("/upload/file", fileUploadHandler)
	http.HandleFunc("/upload/chat", chatUploadHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("📁 File uploader service starting on port %s", port)
	log.Printf("🔗 Upload endpoints:")
	log.Printf("  • Slack export: POST http://localhost:%s/upload/slack", port)
	log.Printf("  • General files: POST http://localhost:%s/upload/file", port)
	log.Printf("  • Chat exports: POST http://localhost:%s/upload/chat", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "OK",
		"timestamp": time.Now(),
		"service":   "file-uploader",
		"version":   "1.0.0",
	})
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "READY",
		"timestamp": time.Now(),
	})
}

func slackUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("📤 Processing Slack export upload...")

	// Parse multipart form (32MB max)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondWithError(w, "Failed to parse form", err)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, "No file provided", err)
		return
	}
	defer file.Close()

	log.Printf("📁 Received file: %s (size: %d bytes)", handler.Filename, handler.Size)

	// Validate file type
	if !isValidSlackFile(handler.Filename) {
		respondWithError(w, "Invalid file type. Expected .json or .zip file", nil)
		return
	}

	// Save file
	fileID := uuid.New().String()
	savedPath, err := saveUploadedFile(file, handler, fileID)
	if err != nil {
		respondWithError(w, "Failed to save file", err)
		return
	}

	// Process Slack export
	processedItems, processingErrors := processSlackFile(savedPath, handler.Filename)

	response := UploadResponse{
		Success:        len(processingErrors) == 0,
		Message:        fmt.Sprintf("Processed %d items from Slack export", processedItems),
		FileID:         fileID,
		Filename:       handler.Filename,
		FileType:       "slack_export",
		ProcessedItems: processedItems,
		Errors:         processingErrors,
	}

	if len(processingErrors) > 0 {
		response.Message = fmt.Sprintf("Processed %d items with %d errors", processedItems, len(processingErrors))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("✅ Slack export processed: %d items, %d errors", processedItems, len(processingErrors))
}

func fileUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("📤 Processing general file upload...")

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondWithError(w, "Failed to parse form", err)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, "No file provided", err)
		return
	}
	defer file.Close()

	log.Printf("📁 Received file: %s (size: %d bytes)", handler.Filename, handler.Size)

	// Validate file type
	fileType := detectFileType(handler.Filename)
	if fileType == "unsupported" {
		respondWithError(w, "Unsupported file type. Expected: .md, .txt, .pdf, .json", nil)
		return
	}

	// Save file
	fileID := uuid.New().String()
	savedPath, err := saveUploadedFile(file, handler, fileID)
	if err != nil {
		respondWithError(w, "Failed to save file", err)
		return
	}

	// Process file based on type
	processedItems, processingErrors := processFile(savedPath, fileType, handler.Filename)

	response := UploadResponse{
		Success:        len(processingErrors) == 0,
		Message:        fmt.Sprintf("Processed %s file with %d items", fileType, processedItems),
		FileID:         fileID,
		Filename:       handler.Filename,
		FileType:       fileType,
		ProcessedItems: processedItems,
		Errors:         processingErrors,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("✅ File processed: %s (%d items, %d errors)", fileType, processedItems, len(processingErrors))
}

func chatUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("📤 Processing chat export upload...")

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondWithError(w, "Failed to parse form", err)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, "No file provided", err)
		return
	}
	defer file.Close()

	// Get chat platform from form
	platform := r.FormValue("platform")
	if platform == "" {
		platform = "unknown"
	}

	log.Printf("📱 Processing %s chat export: %s", platform, handler.Filename)

	// Save and process
	fileID := uuid.New().String()
	savedPath, err := saveUploadedFile(file, handler, fileID)
	if err != nil {
		respondWithError(w, "Failed to save file", err)
		return
	}

	processedItems, processingErrors := processChatFile(savedPath, platform, handler.Filename)

	response := UploadResponse{
		Success:        len(processingErrors) == 0,
		Message:        fmt.Sprintf("Processed %s chat export with %d messages", platform, processedItems),
		FileID:         fileID,
		Filename:       handler.Filename,
		FileType:       fmt.Sprintf("%s_chat", platform),
		ProcessedItems: processedItems,
		Errors:         processingErrors,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("✅ Chat export processed: %s (%d messages, %d errors)", platform, processedItems, len(processingErrors))
}

func isValidSlackFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".json" || ext == ".zip"
}

func detectFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md", ".markdown":
		return "markdown"
	case ".txt":
		return "text"
	case ".pdf":
		return "pdf"
	case ".json":
		return "json"
	default:
		return "unsupported"
	}
}

func saveUploadedFile(file multipart.File, handler *multipart.FileHeader, fileID string) (string, error) {
	// Create safe filename
	ext := filepath.Ext(handler.Filename)
	safeName := fmt.Sprintf("%s_%s%s", fileID, time.Now().Format("20060102_150405"), ext)
	savedPath := filepath.Join("uploads", safeName)

	// Create destination file
	dst, err := os.Create(savedPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copy file data
	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}

	return savedPath, nil
}

func processSlackFile(filePath, filename string) (int, []string) {
	log.Printf("🔄 Processing Slack file: %s", filename)

	// TODO: Implement actual Slack export processing
	// This would:
	// 1. Parse JSON/ZIP file
	// 2. Extract messages, users, channels
	// 3. Store in PostgreSQL content_metadata table
	// 4. Generate embeddings for messages

	// For now, simulate processing
	processedItems := 42 // Simulated number of messages
	errors := []string{} // No errors for now

	log.Printf("📊 Simulated processing: %d Slack messages extracted", processedItems)

	return processedItems, errors
}

func processFile(filePath, fileType, filename string) (int, []string) {
	log.Printf("🔄 Processing %s file: %s", fileType, filename)

	// TODO: Implement actual file processing based on type
	// This would:
	// 1. Parse file content
	// 2. Extract text/metadata
	// 3. Store in database
	// 4. Generate embeddings

	// For now, simulate processing
	processedItems := 1
	errors := []string{}

	switch fileType {
	case "markdown":
		processedItems = 5 // Simulated sections
	case "pdf":
		processedItems = 10 // Simulated pages
	case "json":
		processedItems = 15 // Simulated objects
	}

	log.Printf("📊 Simulated processing: %d items extracted from %s", processedItems, fileType)

	return processedItems, errors
}

func processChatFile(filePath, platform, filename string) (int, []string) {
	log.Printf("🔄 Processing %s chat file: %s", platform, filename)

	// TODO: Implement actual chat processing
	// Support for WhatsApp, Telegram, Discord, etc.

	// Simulate processing based on platform
	processedItems := 100 // Simulated messages
	errors := []string{}

	if platform == "whatsapp" {
		processedItems = 200
	} else if platform == "discord" {
		processedItems = 150
	}

	log.Printf("📊 Simulated processing: %d messages from %s", processedItems, platform)

	return processedItems, errors
}

func respondWithError(w http.ResponseWriter, message string, err error) {
	log.Printf("❌ Error: %s", message)
	if err != nil {
		log.Printf("❌ Details: %v", err)
	}

	response := UploadResponse{
		Success: false,
		Message: message,
		Errors:  []string{message},
	}

	if err != nil {
		response.Errors = append(response.Errors, err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}
