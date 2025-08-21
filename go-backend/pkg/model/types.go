package model

import (
	"time"
)

// Document represents the enhanced structured data extracted from a web page
type Document struct {
	URL         string           `json:"url"`
	Title       string           `json:"title"`
	Text        string           `json:"text"`
	CleanText   string           `json:"clean_text"`
	FetchedAt   time.Time        `json:"fetched_at"`
	Status      int              `json:"status"`
	ContentHash string           `json:"content_hash"`
	Metadata    DocumentMetadata `json:"metadata"`
	Chunks      []ContentChunk   `json:"chunks"`
	Links       []ExtractedLink  `json:"links"`
	Media       []MediaAsset     `json:"media"`
	DreamHints  DreamingHints    `json:"dream_hints"`
}

// DocumentMetadata contains enriched metadata for AI processing
type DocumentMetadata struct {
	Domain      string            `json:"domain"`
	Language    string            `json:"language,omitempty"`
	WordCount   int               `json:"word_count"`
	Author      string            `json:"author,omitempty"`
	PublishedAt *time.Time        `json:"published_at,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Category    string            `json:"category,omitempty"`
	Headers     map[string]string `json:"headers"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
}

// ContentChunk represents semantic chunks for AI processing
type ContentChunk struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"` // headline, paragraph, quote, list, etc.
	Text       string   `json:"text"`
	Position   int      `json:"position"`
	Confidence float64  `json:"confidence"`
	Keywords   []string `json:"keywords,omitempty"`
	Sentiment  string   `json:"sentiment,omitempty"`
	Entities   []string `json:"entities,omitempty"`
}

// ExtractedLink contains enriched link information
type ExtractedLink struct {
	URL      string `json:"url"`
	Text     string `json:"text"`
	Type     string `json:"type"` // internal, external, media
	Context  string `json:"context,omitempty"`
	Priority int    `json:"priority"` // for crawl prioritization
}

// MediaAsset represents images, videos, etc. found on the page
type MediaAsset struct {
	URL     string `json:"url"`
	Type    string `json:"type"` // image, video, audio
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
	Size    string `json:"size,omitempty"`
	Format  string `json:"format,omitempty"`
}

// DreamingHints provides context clues for AI dreaming
type DreamingHints struct {
	Emotions     []string `json:"emotions"`
	Themes       []string `json:"themes"`
	Motifs       []string `json:"motifs"`
	Tone         string   `json:"tone"`
	Complexity   float64  `json:"complexity"`
	Surrealism   float64  `json:"surrealism_potential"`
	VisualCues   []string `json:"visual_cues"`
	AudioCues    []string `json:"audio_cues"`
	ColorPalette []string `json:"color_palette,omitempty"`
	Abstractness float64  `json:"abstractness"`
}

// DreamOutput represents the AI-generated dream content
type DreamOutput struct {
	DocumentID  string    `json:"document_id"`
	URL         string    `json:"url"`
	GeneratedAt time.Time `json:"generated_at"`
	Narrative   string    `json:"narrative"`
	Embeddings  []float64 `json:"embeddings,omitempty"`
	Confidence  float64   `json:"confidence"`
	Model       string    `json:"model"`
}

// CrawlJob represents a crawling task
type CrawlJob struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"` // pending, running, completed, failed
	MaxDepth  int       `json:"max_depth"`
	MaxPages  int       `json:"max_pages"`
	Filters   []string  `json:"filters,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	RateLimit int       `json:"rate_limit,omitempty"`
}

// SearchQuery represents a search request
type SearchQuery struct {
	Query      string   `json:"query"`
	Filters    []string `json:"filters,omitempty"`
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
	SearchType string   `json:"search_type"` // text, semantic, dream
	SortBy     string   `json:"sort_by,omitempty"`
	DateRange  string   `json:"date_range,omitempty"`
}

// SearchResult represents a search result
type SearchResult struct {
	Document   Document      `json:"document"`
	Score      float64       `json:"score"`
	Highlights []string      `json:"highlights,omitempty"`
	Dreams     []DreamOutput `json:"dreams,omitempty"`
}

// Kafka message types
const (
	TopicRawContent   = "raw.content"
	TopicCleanContent = "clean.content"
	TopicDreamOutputs = "dream.outputs"
	TopicCrawlJobs    = "crawl.jobs"
	TopicCrawlResults = "crawl.results"
)

// KafkaMessage represents a message sent through Kafka
type KafkaMessage struct {
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      interface{}       `json:"data"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}
