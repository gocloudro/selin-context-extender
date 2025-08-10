package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// MCP Tool definitions for Claude
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type MCPRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type MCPResponse struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ContentResult struct {
	ID             string    `json:"id"`
	SourceURL      string    `json:"source_url"`
	Author         string    `json:"author"`
	Timestamp      time.Time `json:"timestamp"`
	Tags           []string  `json:"tags"`
	ContentType    string    `json:"content_type"`
	SourcePlatform string    `json:"source_platform"`
	ContentSummary string    `json:"content_summary"`
	RelevanceScore float64   `json:"relevance_score"`
}

func main() {
	log.Println("🚀 Starting Selin MCP Server for Claude...")

	// Setup HTTP routes for MCP
	http.HandleFunc("/mcp/tools", toolsHandler)
	http.HandleFunc("/mcp/call", callHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	log.Printf("🔗 MCP Server starting on port %s", port)
	log.Printf("📡 MCP Endpoints:")
	log.Printf("  • Tools list: GET http://localhost:%s/mcp/tools", port)
	log.Printf("  • Tool calls: POST http://localhost:%s/mcp/call", port)
	log.Printf("  • Health: GET http://localhost:%s/health", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func toolsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tools := []MCPTool{
		{
			Name:        "search_content",
			Description: "Search Selin's knowledge base for content related to Go, blockchain, or cryptography",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (e.g., 'golang concurrency', 'cosmos blockchain')",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of results to return (default: 10)",
						"default":     10,
					},
					"platform": map[string]interface{}{
						"type":        "string",
						"description": "Filter by source platform (reddit, slack, file_upload)",
						"enum":        []string{"reddit", "slack", "file_upload", "all"},
						"default":     "all",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_learning_progress",
			Description: "Get the user's learning progress for specific topics",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"topic": map[string]interface{}{
						"type":        "string",
						"description": "Learning topic (e.g., 'golang', 'blockchain', 'cryptography')",
					},
				},
				"required": []string{"topic"},
			},
		},
		{
			Name:        "get_recent_content",
			Description: "Get recently collected content from Selin's knowledge base",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"hours": map[string]interface{}{
						"type":        "number",
						"description": "Number of hours back to look (default: 24)",
						"default":     24,
					},
					"platform": map[string]interface{}{
						"type":        "string",
						"description": "Filter by source platform",
						"enum":        []string{"reddit", "slack", "file_upload", "all"},
						"default":     "all",
					},
				},
			},
		},
		{
			Name:        "analyze_content_trends",
			Description: "Analyze trends in collected content and learning topics",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"days": map[string]interface{}{
						"type":        "number",
						"description": "Number of days to analyze (default: 7)",
						"default":     7,
					},
					"topic": map[string]interface{}{
						"type":        "string",
						"description": "Focus on specific topic",
					},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	})
}

func callHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	log.Printf("🔧 MCP Tool call: %s with args: %v", req.Name, req.Arguments)

	var response MCPResponse

	switch req.Name {
	case "search_content":
		response = handleSearchContent(req.Arguments)
	case "get_learning_progress":
		response = handleGetLearningProgress(req.Arguments)
	case "get_recent_content":
		response = handleGetRecentContent(req.Arguments)
	case "analyze_content_trends":
		response = handleAnalyzeTrends(req.Arguments)
	default:
		response = MCPResponse{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("Unknown tool: %s", req.Name),
			}},
			IsError: true,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleSearchContent(args map[string]interface{}) MCPResponse {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return errorResponse("Query parameter is required")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	platform := "all"
	if p, ok := args["platform"].(string); ok {
		platform = p
	}

	db, err := getDBConnection()
	if err != nil {
		return errorResponse(fmt.Sprintf("Database connection failed: %v", err))
	}
	defer db.Close()

	// Build SQL query
	sql := `
		SELECT id, source_url, author, timestamp, tags, content_type, 
		       source_platform, content_summary, relevance_score
		FROM content_metadata 
		WHERE (content_summary ILIKE $1 OR array_to_string(tags, ',') ILIKE $1)`

	args_sql := []interface{}{"%" + query + "%"}

	if platform != "all" {
		sql += " AND source_platform = $2"
		args_sql = append(args_sql, platform)
	}

	sql += " ORDER BY relevance_score DESC, created_at DESC LIMIT $" + strconv.Itoa(len(args_sql)+1)
	args_sql = append(args_sql, limit)

	rows, err := db.Query(sql, args_sql...)
	if err != nil {
		return errorResponse(fmt.Sprintf("Query failed: %v", err))
	}
	defer rows.Close()

	var results []ContentResult
	for rows.Next() {
		var result ContentResult
		var tagsStr string

		err := rows.Scan(&result.ID, &result.SourceURL, &result.Author,
			&result.Timestamp, &tagsStr, &result.ContentType,
			&result.SourcePlatform, &result.ContentSummary, &result.RelevanceScore)
		if err != nil {
			continue
		}

		// Parse tags array
		tagsStr = strings.Trim(tagsStr, "{}")
		if tagsStr != "" {
			result.Tags = strings.Split(tagsStr, ",")
		}

		results = append(results, result)
	}

	// Format response
	var responseText strings.Builder
	responseText.WriteString(fmt.Sprintf("🔍 Found %d results for '%s'\n\n", len(results), query))

	for i, result := range results {
		responseText.WriteString(fmt.Sprintf("**%d. %s** (Score: %.2f)\n", i+1, 
			strings.Split(result.ContentSummary, " ")[0], result.RelevanceScore))
		responseText.WriteString(fmt.Sprintf("   • Author: %s\n", result.Author))
		responseText.WriteString(fmt.Sprintf("   • Platform: %s\n", result.SourcePlatform))
		responseText.WriteString(fmt.Sprintf("   • Tags: %s\n", strings.Join(result.Tags, ", ")))
		responseText.WriteString(fmt.Sprintf("   • Summary: %s\n", result.ContentSummary))
		responseText.WriteString(fmt.Sprintf("   • URL: %s\n", result.SourceURL))
		responseText.WriteString(fmt.Sprintf("   • Date: %s\n\n", result.Timestamp.Format("2006-01-02 15:04")))
	}

	return MCPResponse{
		Content: []MCPContent{{
			Type: "text",
			Text: responseText.String(),
		}},
	}
}

func handleGetLearningProgress(args map[string]interface{}) MCPResponse {
	topic, ok := args["topic"].(string)
	if !ok || topic == "" {
		return errorResponse("Topic parameter is required")
	}

	db, err := getDBConnection()
	if err != nil {
		return errorResponse(fmt.Sprintf("Database connection failed: %v", err))
	}
	defer db.Close()

	// Get learning progress
	var skillLevel string
	var progressScore float64
	var totalContent, totalQueries int
	var lastUpdated time.Time

	err = db.QueryRow(`
		SELECT skill_level, progress_score, total_content_consumed, 
		       total_queries, last_updated
		FROM learning_progress 
		WHERE topic = $1`, topic).Scan(&skillLevel, &progressScore, &totalContent, &totalQueries, &lastUpdated)

	if err != nil {
		if err == sql.ErrNoRows {
			return MCPResponse{
				Content: []MCPContent{{
					Type: "text",
					Text: fmt.Sprintf("📚 No learning progress found for topic '%s'. Start by searching for content related to this topic!", topic),
				}},
			}
		}
		return errorResponse(fmt.Sprintf("Query failed: %v", err))
	}

	responseText := fmt.Sprintf(`📊 **Learning Progress for %s**

• **Skill Level**: %s
• **Progress Score**: %.1f/10.0
• **Content Consumed**: %d items
• **Queries Made**: %d
• **Last Updated**: %s

💡 Keep exploring content and asking questions to improve your progress!`,
		strings.Title(topic), skillLevel, progressScore, totalContent, totalQueries, lastUpdated.Format("2006-01-02 15:04"))

	return MCPResponse{
		Content: []MCPContent{{
			Type: "text",
			Text: responseText,
		}},
	}
}

func handleGetRecentContent(args map[string]interface{}) MCPResponse {
	hours := 24.0
	if h, ok := args["hours"].(float64); ok {
		hours = h
	}

	platform := "all"
	if p, ok := args["platform"].(string); ok {
		platform = p
	}

	db, err := getDBConnection()
	if err != nil {
		return errorResponse(fmt.Sprintf("Database connection failed: %v", err))
	}
	defer db.Close()

	sql := `
		SELECT source_platform, content_type, author, content_summary, 
		       relevance_score, created_at
		FROM content_metadata 
		WHERE created_at >= NOW() - INTERVAL '%d hours'`

	if platform != "all" {
		sql += " AND source_platform = '" + platform + "'"
	}

	sql += " ORDER BY created_at DESC LIMIT 20"

	rows, err := db.Query(fmt.Sprintf(sql, int(hours)))
	if err != nil {
		return errorResponse(fmt.Sprintf("Query failed: %v", err))
	}
	defer rows.Close()

	var responseText strings.Builder
	responseText.WriteString(fmt.Sprintf("📅 **Recent Content (Last %.0f hours)**\n\n", hours))

	count := 0
	for rows.Next() {
		var sourcePlatform, contentType, author, summary string
		var relevanceScore float64
		var createdAt time.Time

		err := rows.Scan(&sourcePlatform, &contentType, &author, &summary, &relevanceScore, &createdAt)
		if err != nil {
			continue
		}

		count++
		responseText.WriteString(fmt.Sprintf("**%d.** %s\n", count, summary))
		responseText.WriteString(fmt.Sprintf("   • %s from %s (Score: %.2f)\n", contentType, sourcePlatform, relevanceScore))
		responseText.WriteString(fmt.Sprintf("   • By: %s | %s\n\n", author, createdAt.Format("Jan 2 15:04")))
	}

	if count == 0 {
		responseText.WriteString("No recent content found. The collectors might need more time to gather data.")
	}

	return MCPResponse{
		Content: []MCPContent{{
			Type: "text",
			Text: responseText.String(),
		}},
	}
}

func handleAnalyzeTrends(args map[string]interface{}) MCPResponse {
	days := 7.0
	if d, ok := args["days"].(float64); ok {
		days = d
	}

	db, err := getDBConnection()
	if err != nil {
		return errorResponse(fmt.Sprintf("Database connection failed: %v", err))
	}
	defer db.Close()

	// Get content trends
	rows, err := db.Query(`
		SELECT source_platform, COUNT(*) as count, AVG(relevance_score) as avg_score
		FROM content_metadata 
		WHERE created_at >= NOW() - INTERVAL '%d days'
		GROUP BY source_platform
		ORDER BY count DESC`, int(days))

	if err != nil {
		return errorResponse(fmt.Sprintf("Trends query failed: %v", err))
	}
	defer rows.Close()

	var responseText strings.Builder
	responseText.WriteString(fmt.Sprintf("📈 **Content Trends (Last %.0f days)**\n\n", days))

	for rows.Next() {
		var platform string
		var count int
		var avgScore float64

		err := rows.Scan(&platform, &count, &avgScore)
		if err != nil {
			continue
		}

		responseText.WriteString(fmt.Sprintf("• **%s**: %d items (Avg Score: %.2f)\n", strings.Title(platform), count, avgScore))
	}

	return MCPResponse{
		Content: []MCPContent{{
			Type: "text",
			Text: responseText.String(),
		}},
	}
}

func getDBConnection() (*sql.DB, error) {
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5433"
	}
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "changmeplease"
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "selin"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	return sql.Open("postgres", connStr)
}

func errorResponse(message string) MCPResponse {
	return MCPResponse{
		Content: []MCPContent{{
			Type: "text",
			Text: fmt.Sprintf("❌ Error: %s", message),
		}},
		IsError: true,
	}
}

func respondWithError(w http.ResponseWriter, message string) {
	log.Printf("❌ MCP Error: %s", message)
	response := errorResponse(message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "OK",
		"timestamp": time.Now(),
		"service":   "selin-mcp-server",
		"version":   "1.0.0",
		"tools":     []string{"search_content", "get_learning_progress", "get_recent_content", "analyze_content_trends"},
	})
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	// Test database connection
	db, err := getDBConnection()
	if err != nil {
		http.Error(w, "Database not ready", http.StatusServiceUnavailable)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		http.Error(w, "Database not ready", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "READY",
		"timestamp": time.Now(),
		"database":  "connected",
	})
}
