package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/drawnparadox/web-crawler-that-dreams/go-backend/pkg/model"
)

var (
	port = flag.String("port", "8080", "API server port")
)

type APIServer struct {
	router *mux.Router
	// In a real implementation, you'd have database connections here
}

func NewAPIServer() *APIServer {
	server := &APIServer{
		router: mux.NewRouter(),
	}
	
	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
	
	// Crawling endpoints
	s.router.HandleFunc("/crawl", s.createCrawlJob).Methods("POST")
	s.router.HandleFunc("/crawl/{id}", s.getCrawlJob).Methods("GET")
	s.router.HandleFunc("/crawl/{id}/status", s.getCrawlStatus).Methods("GET")
	
	// Search endpoints
	s.router.HandleFunc("/search", s.searchDocuments).Methods("GET")
	s.router.HandleFunc("/search/semantic", s.semanticSearch).Methods("GET")
	s.router.HandleFunc("/search/dreams", s.searchDreams).Methods("GET")
	
	// Document endpoints
	s.router.HandleFunc("/documents/{id}", s.getDocument).Methods("GET")
	s.router.HandleFunc("/documents/{id}/dreams", s.getDocumentDreams).Methods("GET")
	
	// Stats and analytics
	s.router.HandleFunc("/stats", s.getStats).Methods("GET")
	s.router.HandleFunc("/stats/crawling", s.getCrawlingStats).Methods("GET")
	
	// Middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.corsMiddleware)
}

func (s *APIServer) Start() error {
	log.Printf("Starting API server on port %s", *port)
	return http.ListenAndServe(":"+*port, s.router)
}

// Health check endpoint
func (s *APIServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "web-crawler-api",
	})
}

// Create a new crawl job
func (s *APIServer) createCrawlJob(w http.ResponseWriter, r *http.Request) {
	var job model.CrawlJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Generate job ID and set defaults
	job.ID = fmt.Sprintf("job_%d", time.Now().Unix())
	job.CreatedAt = time.Now()
	job.Status = "pending"
	
	if job.MaxDepth == 0 {
		job.MaxDepth = 2
	}
	if job.MaxPages == 0 {
		job.MaxPages = 100
	}
	if job.RateLimit == 0 {
		job.RateLimit = 10
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

// Get crawl job details
func (s *APIServer) getCrawlJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]
	
	// Mock response - in real implementation, fetch from database
	job := model.CrawlJob{
		ID:        jobID,
		URL:       "https://example.com",
		Status:    "completed",
		CreatedAt: time.Now().Add(-time.Hour),
		MaxDepth:  2,
		MaxPages:  100,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// Get crawl job status
func (s *APIServer) getCrawlStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]
	
	// Mock response
	status := map[string]interface{}{
		"job_id":     jobID,
		"status":     "completed",
		"progress":   100,
		"pages_crawled": 45,
		"errors":     0,
		"started_at": time.Now().Add(-time.Hour),
		"completed_at": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Search documents
func (s *APIServer) searchDocuments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}
	
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	
	// Mock search results
	results := []model.SearchResult{
		{
			Document: model.Document{
				URL:       "https://example.com/article1",
				Title:     "Sample Article",
				CleanText: "This is a sample article about " + query,
			},
			Score: 0.95,
		},
	}
	
	response := map[string]interface{}{
		"query":   query,
		"results": results,
		"total":   len(results),
		"limit":   limit,
		"offset":  offset,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Semantic search
func (s *APIServer) semanticSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}
	
	// Mock semantic search results
	results := []model.SearchResult{
		{
			Document: model.Document{
				URL:       "https://example.com/semantic1",
				Title:     "Semantic Result",
				CleanText: "This document is semantically related to: " + query,
			},
			Score: 0.87,
		},
	}
	
	response := map[string]interface{}{
		"query":   query,
		"type":    "semantic",
		"results": results,
		"total":   len(results),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Search dreams
func (s *APIServer) searchDreams(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}
	
	// Mock dream search results
	results := []model.SearchResult{
		{
			Document: model.Document{
				URL:       "https://example.com/dream1",
				Title:     "Dream Result",
				CleanText: "A dream about: " + query,
			},
			Score: 0.92,
			Dreams: []model.DreamOutput{
				{
					Narrative: "In the dream, " + query + " becomes a surreal landscape...",
					Confidence: 0.88,
				},
			},
		},
	}
	
	response := map[string]interface{}{
		"query":   query,
		"type":    "dream",
		"results": results,
		"total":   len(results),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get document by ID
func (s *APIServer) getDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docID := vars["id"]
	
	// Mock document
	doc := model.Document{
		URL:       "https://example.com/" + docID,
		Title:     "Document " + docID,
		CleanText: "This is the content of document " + docID,
		FetchedAt: time.Now().Add(-time.Hour),
		Status:    200,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// Get document dreams
func (s *APIServer) getDocumentDreams(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docID := vars["id"]
	
	// Mock dreams
	dreams := []model.DreamOutput{
		{
			DocumentID:  docID,
			URL:         "https://example.com/" + docID,
			GeneratedAt: time.Now().Add(-30 * time.Minute),
			Narrative:   "A surreal dream about document " + docID + "...",
			Confidence:  0.85,
			Model:       "tinyllama-1.1b-chat",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dreams)
}

// Get general stats
func (s *APIServer) getStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"total_documents": 1234,
		"total_dreams":   567,
		"active_crawls":  3,
		"last_updated":   time.Now().UTC(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Get crawling stats
func (s *APIServer) getCrawlingStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"crawls_today":    15,
		"crawls_this_week": 89,
		"pages_crawled":   1234,
		"errors":          5,
		"avg_speed":       "2.3 pages/sec",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Middleware
func (s *APIServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func (s *APIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func main() {
	flag.Parse()
	
	server := NewAPIServer()
	
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
