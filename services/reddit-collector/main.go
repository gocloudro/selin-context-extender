package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type RedditPost struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	SelfText    string  `json:"selftext"`
	Author      string  `json:"author"`
	Subreddit   string  `json:"subreddit"`
	URL         string  `json:"url"`
	Score       int     `json:"score"`
	CreatedUTC  float64 `json:"created_utc"`
	Permalink   string  `json:"permalink"`
	NumComments int     `json:"num_comments"`
}

type RedditResponse struct {
	Data struct {
		Children []struct {
			Data RedditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type ContentMetadata struct {
	ID             string    `json:"id"`
	SourceURL      string    `json:"source_url"`
	Author         string    `json:"author"`
	Timestamp      time.Time `json:"timestamp"`
	Tags           []string  `json:"tags"`
	ContentType    string    `json:"content_type"`
	SourcePlatform string    `json:"source_platform"`
	Language       string    `json:"language"`
	ContentSummary string    `json:"content_summary"`
	RelevanceScore float64   `json:"relevance_score"`
}

func main() {
	log.Println("🚀 Starting Reddit Collector...")

	// Configuration from environment or defaults
	subreddits := getSubreddits()
	userAgent := os.Getenv("REDDIT_USER_AGENT")
	if userAgent == "" {
		userAgent = "selin-bot/1.0"
	}

	log.Printf("📡 Collecting from subreddits: %v", subreddits)

	// Start HTTP server for health checks
	go startHealthServer()

	// Collection loop
	for {
		for _, subreddit := range subreddits {
			log.Printf("🔍 Collecting from r/%s...", subreddit)
			posts, err := collectFromSubreddit(subreddit, userAgent)
			if err != nil {
				log.Printf("❌ Error collecting from r/%s: %v", subreddit, err)
				continue
			}

			log.Printf("📊 Found %d posts in r/%s", len(posts), subreddit)

			// Process and store posts
			for _, post := range posts {
				content := convertToContentMetadata(post)
				if shouldStore(content) {
					if err := storeContent(content); err != nil {
						log.Printf("❌ Error storing post %s: %v", post.ID, err)
					} else {
						log.Printf("✅ Stored post: %s", post.Title[:min(50, len(post.Title))])
					}
				}
			}
		}

		// Wait before next collection
		log.Println("😴 Waiting 5 minutes before next collection...")
		time.Sleep(5 * time.Minute)
	}
}

func getSubreddits() []string {
	subredditStr := os.Getenv("REDDIT_SUBREDDITS")
	if subredditStr == "" {
		// Default subreddits for Go, blockchain, and cryptography
		return []string{"golang", "cosmosdev", "cryptography", "programming", "kubernetes"}
	}
	return strings.Split(subredditStr, ",")
}

func collectFromSubreddit(subreddit, userAgent string) ([]RedditPost, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=25", subreddit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit API returned status %d", resp.StatusCode)
	}

	var redditResp RedditResponse
	if err := json.NewDecoder(resp.Body).Decode(&redditResp); err != nil {
		return nil, err
	}

	posts := make([]RedditPost, len(redditResp.Data.Children))
	for i, child := range redditResp.Data.Children {
		posts[i] = child.Data
	}

	return posts, nil
}

func convertToContentMetadata(post RedditPost) ContentMetadata {
	// Generate content summary
	content := post.Title
	if post.SelfText != "" {
		content += " " + post.SelfText
	}

	summary := content
	if len(summary) > 200 {
		summary = summary[:200] + "..."
	}

	// Calculate relevance score based on keywords
	relevanceScore := calculateRelevanceScore(content)

	// Extract tags
	tags := extractTags(content, post.Subreddit)

	return ContentMetadata{
		ID:             uuid.New().String(),
		SourceURL:      "https://reddit.com" + post.Permalink,
		Author:         post.Author,
		Timestamp:      time.Unix(int64(post.CreatedUTC), 0),
		Tags:           tags,
		ContentType:    "reddit_post",
		SourcePlatform: "reddit",
		Language:       "en",
		ContentSummary: summary,
		RelevanceScore: relevanceScore,
	}
}

func calculateRelevanceScore(content string) float64 {
	content = strings.ToLower(content)
	score := 0.0

	// High-value keywords for our learning focus
	highValueKeywords := []string{
		"golang", "go programming", "concurrency", "goroutine",
		"blockchain", "cosmos", "tendermint", "celestia",
		"cryptography", "encryption", "hash", "merkle tree",
		"kubernetes", "k8s", "docker", "microservices",
	}

	for _, keyword := range highValueKeywords {
		if strings.Contains(content, keyword) {
			score += 0.2
		}
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

func extractTags(content, subreddit string) []string {
	content = strings.ToLower(content)
	tags := []string{subreddit}

	tagMap := map[string]string{
		"golang":         "golang",
		"go programming": "golang",
		"goroutine":      "concurrency",
		"channel":        "concurrency",
		"blockchain":     "blockchain",
		"cosmos":         "cosmos",
		"tendermint":     "tendermint",
		"celestia":       "celestia",
		"cryptography":   "cryptography",
		"encryption":     "cryptography",
		"kubernetes":     "kubernetes",
		"k8s":            "kubernetes",
		"docker":         "containerization",
	}

	for keyword, tag := range tagMap {
		if strings.Contains(content, keyword) {
			tags = append(tags, tag)
		}
	}

	return removeDuplicates(tags)
}

func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

func shouldStore(content ContentMetadata) bool {
	// Store if relevance score is above threshold
	return content.RelevanceScore > 0.1
}

func storeContent(content ContentMetadata) error {
	// Get database connection details from environment
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

	// Create connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Convert tags slice to PostgreSQL array format
	tagsArray := fmt.Sprintf("{%s}", strings.Join(content.Tags, ","))

	// Insert content into database
	query := `
		INSERT INTO content_metadata (
			id, source_url, author, timestamp, tags, content_type,
			source_platform, language, content_summary, relevance_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (source_url) DO UPDATE SET
			relevance_score = EXCLUDED.relevance_score,
			updated_at = now()`

	_, err = db.Exec(query,
		content.ID,
		content.SourceURL,
		content.Author,
		content.Timestamp,
		tagsArray,
		content.ContentType,
		content.SourcePlatform,
		content.Language,
		content.ContentSummary,
		content.RelevanceScore,
	)

	if err != nil {
		return fmt.Errorf("failed to insert content: %v", err)
	}

	log.Printf("💾 Stored in DB: %s (score: %.2f, tags: %v)",
		content.ContentSummary[:min(100, len(content.ContentSummary))],
		content.RelevanceScore,
		content.Tags)

	return nil
}

func startHealthServer() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "OK",
			"timestamp": time.Now(),
			"service":   "reddit-collector",
			"version":   "1.0.0",
		})
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "READY",
			"timestamp": time.Now(),
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("🏥 Health server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
