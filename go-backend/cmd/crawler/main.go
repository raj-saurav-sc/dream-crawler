package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/temoto/robotstxt"
	"golang.org/x/time/rate"
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

// Enhanced crawler config
var (
	workers         = flag.Int("workers", 10, "number of crawler workers")
	queueSize       = flag.Int("queue", 1000, "url queue buffer size")
	timeoutSec      = flag.Int("timeout", 15, "http client timeout in seconds")
	kafkaBroker     = flag.String("kafka-broker", "localhost:9092", "Kafka broker address")
	kafkaTopic      = flag.String("kafka-topic", "raw.content", "Kafka topic for raw content")
	dreamTopic      = flag.String("dream-topic", "dream.seeds", "Kafka topic for dream-ready content")
	maxDepth        = flag.Int("max-depth", 3, "maximum crawl depth")
	enableDreaming  = flag.Bool("enable-dreaming", true, "enable AI dream hint generation")
	domainWhitelist = flag.String("domains", "", "comma-separated list of allowed domains")
)

// hostPolicies stores the robots.txt data and rate limiter for a specific host
type hostPolicies struct {
	robots *robotstxt.RobotsData
	lim    *rate.Limiter
}

// URLMetadata tracks crawl metadata
type URLMetadata struct {
	depth    int
	parent   string
	priority int
}

func main() {
	flag.Parse()
	seeds := flag.Args()
	if len(seeds) == 0 {
		log.Fatalf("usage: crawler [flags] <seed-url-1> <seed-url-2> ...")
	}

	// Kafka Producer setup
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": *kafkaBroker,
		"batch.size":        16384,
		"linger.ms":         10,
	})
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %s", err)
	}
	defer producer.Close()

	// Enhanced delivery reports handling
	go handleKafkaEvents(producer)

	// Enhanced channels and context
	urlQueue := make(chan URLWithMetadata, *queueSize)
	rawOut := make(chan Document)
	dreamOut := make(chan Document)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Shared state
	var hpMu sync.Mutex
	hostMap := make(map[string]*hostPolicies)
	seen := sync.Map{}
	stats := &CrawlerStats{}

	// Domain whitelist processing
	var allowedDomains map[string]bool
	if *domainWhitelist != "" {
		allowedDomains = make(map[string]bool)
		for _, domain := range strings.Split(*domainWhitelist, ",") {
			allowedDomains[strings.TrimSpace(domain)] = true
		}
	}

	// Shared HTTP client with better configuration
	client := &http.Client{
		Timeout: time.Duration(*timeoutSec) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Start enhanced crawler workers
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			enhancedWorker(ctx, id, urlQueue, rawOut, client, &hpMu, hostMap, &seen, stats, allowedDomains)
		}(i)
	}

	// Dream processor (if enabled)
	if *enableDreaming {
		go dreamProcessor(ctx, rawOut, dreamOut)
	} else {
		// If dreaming is disabled, just pass through
		go func() {
			for doc := range rawOut {
				dreamOut <- doc
			}
		}()
	}

	// Seed the queue
	go func() {
		for _, s := range seeds {
			urlQueue <- URLWithMetadata{URL: s, Metadata: URLMetadata{depth: 0, priority: 10}}
		}
	}()

	// Enhanced producer with multiple topics
	go enhancedProducer(producer, dreamOut)

	// Stats reporter
	go statsReporter(ctx, stats)

	// Enhanced runtime with graceful shutdown
	log.Println("Enhanced Dream Crawler starting...")
	timer := time.NewTimer(180 * time.Second) // 3 minutes for demo
	<-timer.C

	log.Println("Shutting down gracefully...")
	cancel()
	wg.Wait()
	producer.Flush(15 * 1000)
	close(rawOut)
	close(dreamOut)

	// Final stats
	log.Printf("Crawl complete. Pages processed: %d, Errors: %d, Dreams generated: %d",
		stats.PagesProcessed, stats.Errors, stats.DreamsGenerated)
}

// URLWithMetadata wraps URL with crawl metadata
type URLWithMetadata struct {
	URL      string
	Metadata URLMetadata
}

// CrawlerStats tracks crawler performance
type CrawlerStats struct {
	mu              sync.Mutex
	PagesProcessed  int64
	Errors          int64
	DreamsGenerated int64
	BytesProcessed  int64
	AveragePageSize float64
}

func (s *CrawlerStats) IncrementPages() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PagesProcessed++
}

func (s *CrawlerStats) IncrementErrors() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Errors++
}

func (s *CrawlerStats) IncrementDreams() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DreamsGenerated++
}

func (s *CrawlerStats) AddBytes(bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BytesProcessed += bytes
	s.AveragePageSize = float64(s.BytesProcessed) / float64(s.PagesProcessed)
}

// Enhanced worker with AI-ready content extraction
func enhancedWorker(ctx context.Context, id int, urlQueue chan URLWithMetadata, out chan<- Document,
	client *http.Client, hpMu *sync.Mutex, hostMap map[string]*hostPolicies,
	seen *sync.Map, stats *CrawlerStats, allowedDomains map[string]bool) {

	for {
		select {
		case <-ctx.Done():
			return
		case urlMeta := <-urlQueue:
			if urlMeta.URL == "" {
				continue
			}

			// Skip if already seen
			if _, loaded := seen.LoadOrStore(urlMeta.URL, true); loaded {
				continue
			}

			// Respect max depth
			if urlMeta.Metadata.depth > *maxDepth {
				continue
			}

			parsed, err := url.Parse(urlMeta.URL)
			if err != nil {
				log.Printf("worker %d: bad url %s: %v", id, urlMeta.URL, err)
				stats.IncrementErrors()
				continue
			}

			// Domain whitelist check
			if allowedDomains != nil && !allowedDomains[parsed.Host] {
				continue
			}

			host := parsed.Host

			// Get/create host policies
			hpMu.Lock()
			hp, ok := hostMap[host]
			if !ok {
				hp = &hostPolicies{lim: rate.NewLimiter(rate.Every(500*time.Millisecond), 1)}
				hostMap[host] = hp
				go fetchRobotsTxt(client, parsed, hp)
			}
			hpMu.Unlock()

			// Robots.txt check
			if hp.robots != nil && !hp.robots.TestAgent(parsed.Path, "WebCrawlerThatDreams/1.0") {
				log.Printf("worker %d: disallowed by robots: %s", id, urlMeta.URL)
				continue
			}

			// Rate limiting
			if err := hp.lim.Wait(ctx); err != nil {
				continue
			}

			// Enhanced fetch and parse
			log.Printf("worker %d: fetching %s (depth: %d)", id, urlMeta.URL, urlMeta.Metadata.depth)
			doc, newLinks, err := enhancedFetchAndParse(ctx, client, urlMeta.URL, urlMeta.Metadata)
			if err != nil {
				log.Printf("worker %d: fetch error %s: %v", id, urlMeta.URL, err)
				stats.IncrementErrors()
				continue
			}

			stats.IncrementPages()
			stats.AddBytes(int64(len(doc.Text)))
			out <- doc

			// Queue new links with incremented depth
			for _, link := range newLinks {
				if link.Priority > 0 { // Only queue high-priority links
					newMeta := URLMetadata{
						depth:    urlMeta.Metadata.depth + 1,
						parent:   urlMeta.URL,
						priority: link.Priority,
					}
					select {
					case urlQueue <- URLWithMetadata{URL: link.URL, Metadata: newMeta}:
					default:
						// Queue full, drop low priority links
						if link.Priority >= 5 {
							log.Printf("worker %d: queue full, dropping link: %s", id, link.URL)
						}
					}
				}
			}
		}
	}
}

// Enhanced fetch and parse with AI-ready extraction
func enhancedFetchAndParse(ctx context.Context, client *http.Client, rawurl string, metadata URLMetadata) (Document, []ExtractedLink, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawurl, nil)
	if err != nil {
		return Document{}, nil, err
	}
	req.Header.Set("User-Agent", "WebCrawlerThatDreams/1.0 (+https://github.com/dreamweaver/crawler)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return Document{}, nil, err
	}
	defer resp.Body.Close()

	// Initialize document with enhanced metadata
	doc := Document{
		URL:       rawurl,
		FetchedAt: time.Now(),
		Status:    resp.StatusCode,
		Metadata: DocumentMetadata{
			Headers:     make(map[string]string),
			ContentType: resp.Header.Get("Content-Type"),
			Size:        resp.ContentLength,
		},
	}

	// Capture response headers
	for key, values := range resp.Header {
		if len(values) > 0 {
			doc.Metadata.Headers[key] = values[0]
		}
	}

	if resp.StatusCode != http.StatusOK {
		return doc, nil, nil
	}

	// Parse with goquery
	gqDoc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return doc, nil, err
	}

	// Enhanced content extraction
	doc.Title = strings.TrimSpace(gqDoc.Find("title").First().Text())
	doc.Text = extractText(gqDoc)
	doc.CleanText = cleanText(doc.Text)
	doc.ContentHash = fmt.Sprintf("%x", md5.Sum([]byte(doc.CleanText)))
	doc.Metadata.Domain = extractDomain(rawurl)
	doc.Metadata.WordCount = len(strings.Fields(doc.CleanText))

	// Extract metadata
	extractMetadata(gqDoc, &doc.Metadata)

	// Extract semantic chunks
	doc.Chunks = extractContentChunks(gqDoc, doc.CleanText)

	// Extract links with priority
	links := extractLinksWithPriority(gqDoc, rawurl, metadata.depth)

	// Extract media assets
	doc.Media = extractMediaAssets(gqDoc, rawurl)

	// Generate dream hints
	doc.DreamHints = generateDreamHints(doc)

	return doc, links, nil
}

// Extract enhanced metadata from HTML
func extractMetadata(doc *goquery.Document, metadata *DocumentMetadata) {
	// Author extraction
	doc.Find("meta[name='author'], meta[property='article:author']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists && metadata.Author == "" {
			metadata.Author = strings.TrimSpace(content)
		}
	})

	// Published date
	doc.Find("meta[property='article:published_time'], meta[name='date']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			if publishedAt, err := time.Parse(time.RFC3339, content); err == nil {
				metadata.PublishedAt = &publishedAt
			}
		}
	})

	// Tags/Keywords
	doc.Find("meta[name='keywords'], meta[property='article:tag']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			tags := strings.Split(content, ",")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					metadata.Tags = append(metadata.Tags, tag)
				}
			}
		}
	})

	// Category
	doc.Find("meta[property='article:section'], meta[name='category']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists && metadata.Category == "" {
			metadata.Category = strings.TrimSpace(content)
		}
	})

	// Language
	if lang, exists := doc.Find("html").Attr("lang"); exists {
		metadata.Language = lang
	}
}

// Extract content chunks for AI processing
func extractContentChunks(doc *goquery.Document, cleanText string) []ContentChunk {
	var chunks []ContentChunk
	chunkID := 0

	// Headlines
	doc.Find("h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" && len(text) > 5 {
			chunks = append(chunks, ContentChunk{
				ID:         fmt.Sprintf("h_%d", chunkID),
				Type:       "headline",
				Text:       text,
				Position:   chunkID,
				Confidence: 0.9,
				Keywords:   extractKeywords(text),
			})
			chunkID++
		}
	})

	// Paragraphs
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" && len(text) > 20 {
			chunks = append(chunks, ContentChunk{
				ID:         fmt.Sprintf("p_%d", chunkID),
				Type:       "paragraph",
				Text:       text,
				Position:   chunkID,
				Confidence: 0.8,
				Keywords:   extractKeywords(text),
				Sentiment:  detectSentiment(text),
				Entities:   extractEntities(text),
			})
			chunkID++
		}
	})

	// Quotes
	doc.Find("blockquote, q").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			chunks = append(chunks, ContentChunk{
				ID:         fmt.Sprintf("q_%d", chunkID),
				Type:       "quote",
				Text:       text,
				Position:   chunkID,
				Confidence: 0.85,
				Keywords:   extractKeywords(text),
				Sentiment:  detectSentiment(text),
			})
			chunkID++
		}
	})

	return chunks
}

// Extract links with priority scoring
func extractLinksWithPriority(doc *goquery.Document, baseURL string, currentDepth int) []ExtractedLink {
	var links []ExtractedLink
	base, _ := url.Parse(baseURL)

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || len(href) < 2 || strings.HasPrefix(href, "#") {
			return
		}

		resolvedURL, err := base.Parse(href)
		if err != nil {
			return
		}

		if resolvedURL.Scheme != "http" && resolvedURL.Scheme != "https" {
			return
		}

		linkText := strings.TrimSpace(s.Text())
		linkType := "external"
		priority := 1

		// Internal vs external
		if resolvedURL.Host == base.Host {
			linkType = "internal"
			priority = 3
		}

		// Priority based on context and content
		if strings.Contains(strings.ToLower(linkText), "article") ||
			strings.Contains(strings.ToLower(linkText), "news") ||
			strings.Contains(strings.ToLower(linkText), "blog") {
			priority += 2
		}

		// Reduce priority for deep links
		if currentDepth >= 2 {
			priority = max(1, priority-1)
		}

		links = append(links, ExtractedLink{
			URL:      resolvedURL.String(),
			Text:     linkText,
			Type:     linkType,
			Priority: priority,
		})
	})

	return links
}

// Extract media assets
func extractMediaAssets(doc *goquery.Document, baseURL string) []MediaAsset {
	var media []MediaAsset
	base, _ := url.Parse(baseURL)

	// Images
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists {
			return
		}

		resolvedURL, err := base.Parse(src)
		if err != nil {
			return
		}

		alt, _ := s.Attr("alt")
		media = append(media, MediaAsset{
			URL:    resolvedURL.String(),
			Type:   "image",
			Alt:    alt,
			Format: getFileExtension(src),
		})
	})

	// Videos
	doc.Find("video source, video").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists {
			return
		}

		resolvedURL, err := base.Parse(src)
		if err != nil {
			return
		}

		media = append(media, MediaAsset{
			URL:    resolvedURL.String(),
			Type:   "video",
			Format: getFileExtension(src),
		})
	})

	return media
}

// Generate AI dream hints from content
func generateDreamHints(doc Document) DreamingHints {
	text := strings.ToLower(doc.CleanText + " " + doc.Title)

	hints := DreamingHints{
		Emotions:     detectEmotions(text),
		Themes:       detectThemes(text),
		Motifs:       extractVisualMotifs(text),
		Tone:         detectTone(text),
		VisualCues:   extractVisualCues(text),
		AudioCues:    extractAudioCues(text),
		ColorPalette: extractColors(text),
	}

	// Calculate complexity and surrealism potential
	hints.Complexity = calculateComplexity(doc)
	hints.Surrealism = calculateSurrealismPotential(doc, hints)
	hints.Abstractness = calculateAbstractness(text, hints)

	return hints
}

// Dream processor - prepares content for AI dreaming
func dreamProcessor(ctx context.Context, input <-chan Document, output chan<- Document) {
	for {
		select {
		case <-ctx.Done():
			return
		case doc := <-input:
			// Process document for dreaming
			if doc.DreamHints.Surrealism > 0.3 && len(doc.CleanText) > 100 {
				// This document has dream potential
				log.Printf("Dream processor: High surrealism potential (%.2f) for %s",
					doc.DreamHints.Surrealism, doc.URL)
			}

			output <- doc
		}
	}
}

// Enhanced Kafka producer
func enhancedProducer(producer *kafka.Producer, input <-chan Document) {
	for doc := range input {
		docBytes, err := json.Marshal(doc)
		if err != nil {
			log.Printf("JSON marshal error: %v", err)
			continue
		}

		// Send to raw content topic
		producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: kafkaTopic, Partition: kafka.PartitionAny},
			Value:          docBytes,
			Key:            []byte(doc.URL),
			Headers: []kafka.Header{
				{Key: "content_type", Value: []byte("application/json")},
				{Key: "crawler_version", Value: []byte("dream-crawler-v1.0")},
				{Key: "surrealism_score", Value: []byte(fmt.Sprintf("%.2f", doc.DreamHints.Surrealism))},
			},
		}, nil)

		// Send high-surrealism content to dream topic
		if doc.DreamHints.Surrealism > 0.5 {
			producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: dreamTopic, Partition: kafka.PartitionAny},
				Value:          docBytes,
				Key:            []byte(doc.URL),
				Headers: []kafka.Header{
					{Key: "dream_ready", Value: []byte("true")},
					{Key: "surrealism_score", Value: []byte(fmt.Sprintf("%.2f", doc.DreamHints.Surrealism))},
				},
			}, nil)
		}
	}
}

// Handle Kafka events
func handleKafkaEvents(producer *kafka.Producer) {
	for e := range producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Printf("Kafka delivery failed: %v", ev.TopicPartition)
			}
		}
	}
}

// Stats reporter
func statsReporter(ctx context.Context, stats *CrawlerStats) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats.mu.Lock()
			log.Printf("Stats: Pages: %d, Errors: %d, Dreams: %d, Avg Size: %.1f bytes",
				stats.PagesProcessed, stats.Errors, stats.DreamsGenerated, stats.AveragePageSize)
			stats.mu.Unlock()
		}
	}
}

// Helper functions for AI analysis
func detectEmotions(text string) []string {
	emotions := []string{}

	positiveWords := []string{"amazing", "beautiful", "wonderful", "great", "love", "happy", "joy", "success"}
	negativeWords := []string{"terrible", "awful", "hate", "sad", "fear", "anger", "pain", "failure"}
	mysticalWords := []string{"mystery", "magic", "dream", "vision", "spirit", "soul", "ethereal", "cosmic"}

	for _, word := range positiveWords {
		if strings.Contains(text, word) {
			emotions = append(emotions, "positive")
			break
		}
	}

	for _, word := range negativeWords {
		if strings.Contains(text, word) {
			emotions = append(emotions, "dark")
			break
		}
	}

	for _, word := range mysticalWords {
		if strings.Contains(text, word) {
			emotions = append(emotions, "mystical")
			break
		}
	}

	if len(emotions) == 0 {
		emotions = append(emotions, "neutral")
	}

	return emotions
}

func detectThemes(text string) []string {
	themes := []string{}

	techWords := []string{"technology", "ai", "computer", "digital", "software", "algorithm"}
	artWords := []string{"art", "creative", "design", "visual", "aesthetic", "beauty"}
	scienceWords := []string{"science", "research", "discovery", "experiment", "analysis"}

	for _, word := range techWords {
		if strings.Contains(text, word) {
			themes = append(themes, "technology")
			break
		}
	}

	for _, word := range artWords {
		if strings.Contains(text, word) {
			themes = append(themes, "creative")
			break
		}
	}

	for _, word := range scienceWords {
		if strings.Contains(text, word) {
			themes = append(themes, "scientific")
			break
		}
	}

	return themes
}

func extractVisualMotifs(text string) []string {
	visualWords := []string{"light", "shadow", "color", "bright", "dark", "crystal", "liquid", "flowing", "geometric", "organic"}
	motifs := []string{}

	for _, word := range visualWords {
		if strings.Contains(text, word) {
			motifs = append(motifs, word)
		}
	}

	return motifs
}

func extractVisualCues(text string) []string {
	return []string{"ethereal lighting", "flowing forms", "crystalline structures"}
}

func extractAudioCues(text string) []string {
	return []string{"ambient whispers", "digital harmonics", "pulsing rhythms"}
}

func extractColors(text string) []string {
	colors := []string{}
	colorWords := []string{"red", "blue", "green", "yellow", "purple", "orange", "pink", "white", "black", "gold", "silver"}

	for _, color := range colorWords {
		if strings.Contains(text, color) {
			colors = append(colors, color)
		}
	}

	return colors
}

func calculateComplexity(doc Document) float64 {
	// Based on text length, chunk diversity, and metadata richness
	complexity := float64(doc.Metadata.WordCount) / 1000.0
	complexity += float64(len(doc.Chunks)) / 10.0
	complexity += float64(len(doc.Media)) / 5.0

	return min(1.0, complexity)
}

func calculateSurrealismPotential(doc Document, hints DreamingHints) float64 {
	score := 0.0

	// Emotional diversity increases surrealism
	if len(hints.Emotions) > 1 {
		score += 0.3
	}

	// Mystical/abstract themes boost surrealism
	for _, emotion := range hints.Emotions {
		if emotion == "mystical" {
			score += 0.4
		}
	}

	// Creative/artistic content is more surreal
	for _, theme := range hints.Themes {
		if theme == "creative" {
			score += 0.3
		}
	}

	// Visual motifs indicate surreal potential
	score += float64(len(hints.Motifs)) * 0.05

	// Complex content tends to be more surreal
	score += hints.Complexity * 0.2

	return min(1.0, score)
}

func calculateAbstractness(text string, hints DreamingHints) float64 {
	abstractWords := []string{"concept", "idea", "essence", "meaning", "philosophy", "abstract", "theory", "metaphor"}
	score := 0.0

	for _, word := range abstractWords {
		if strings.Contains(text, word) {
			score += 0.1
		}
	}

	// High emotion diversity suggests abstractness
	score += float64(len(hints.Emotions)) * 0.05

	return min(1.0, score)
}

func detectTone(text string) string {
	formalWords := []string{"therefore", "furthermore", "consequently", "analysis", "research"}
	casualWords := []string{"really", "pretty", "quite", "basically", "actually"}
	dramaticWords := []string{"incredible", "amazing", "shocking", "revolutionary", "breakthrough"}

	formalCount := 0
	casualCount := 0
	dramaticCount := 0

	for _, word := range formalWords {
		if strings.Contains(text, word) {
			formalCount++
		}
	}

	for _, word := range casualWords {
		if strings.Contains(text, word) {
			casualCount++
		}
	}

	for _, word := range dramaticWords {
		if strings.Contains(text, word) {
			dramaticCount++
		}
	}

	if dramaticCount > formalCount && dramaticCount > casualCount {
		return "dramatic"
	} else if formalCount > casualCount {
		return "formal"
	} else if casualCount > 0 {
		return "casual"
	}

	return "neutral"
}

func detectSentiment(text string) string {
	positiveWords := []string{"good", "great", "excellent", "amazing", "wonderful", "love", "best"}
	negativeWords := []string{"bad", "terrible", "awful", "hate", "worst", "horrible"}

	positiveCount := 0
	negativeCount := 0

	for _, word := range positiveWords {
		positiveCount += strings.Count(strings.ToLower(text), word)
	}

	for _, word := range negativeWords {
		negativeCount += strings.Count(strings.ToLower(text), word)
	}

	if positiveCount > negativeCount {
		return "positive"
	} else if negativeCount > positiveCount {
		return "negative"
	}

	return "neutral"
}

func extractKeywords(text string) []string {
	// Simple keyword extraction - in production you'd use proper NLP
	words := strings.Fields(strings.ToLower(text))
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
		"this": true, "that": true, "these": true, "those": true, "i": true, "you": true,
		"he": true, "she": true, "it": true, "we": true, "they": true,
	}

	keywords := []string{}
	wordCount := make(map[string]int)

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 3 && !stopWords[word] {
			wordCount[word]++
		}
	}

	// Get top keywords
	for word, count := range wordCount {
		if count >= 2 || len(word) > 6 {
			keywords = append(keywords, word)
		}
		if len(keywords) >= 10 {
			break
		}
	}

	return keywords
}

func extractEntities(text string) []string {
	// Simple entity extraction - looks for capitalized words
	re := regexp.MustCompile(`\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\b`)
	matches := re.FindAllString(text, -1)

	entities := []string{}
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 3 && !seen[match] {
			entities = append(entities, match)
			seen[match] = true
		}
		if len(entities) >= 5 {
			break
		}
	}

	return entities
}

// Enhanced text extraction with better cleaning
func extractText(d *goquery.Document) string {
	// Remove non-content elements
	d.Find("script, style, noscript, nav, footer, header, aside, .advertisement, .ad, .sidebar").Remove()

	// Get text from main content areas
	var textParts []string

	// Try to find main content areas first
	mainContent := d.Find("main, article, .content, .post, .entry, #main, #content")
	if mainContent.Length() > 0 {
		mainContent.Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if len(text) > 50 {
				textParts = append(textParts, text)
			}
		})
	} else {
		// Fallback to body
		text := strings.TrimSpace(d.Find("body").Text())
		if text != "" {
			textParts = append(textParts, text)
		}
	}

	return strings.Join(textParts, "\n\n")
}

func cleanText(text string) string {
	// Remove excessive whitespace
	re := regexp.MustCompile(`\s+`)
	cleaned := re.ReplaceAllString(text, " ")

	// Remove special characters but keep punctuation
	re = regexp.MustCompile(`[^\w\s\.,!?;:'"()-]`)
	cleaned = re.ReplaceAllString(cleaned, "")

	return strings.TrimSpace(cleaned)
}

func extractDomain(rawurl string) string {
	parsed, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

// Robots.txt fetching (unchanged from original)
func fetchRobotsTxt(client *http.Client, base *url.URL, hp *hostPolicies) {
	robotsURL := base.Scheme + "://" + base.Host + "/robots.txt"
	resp, err := client.Get(robotsURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}
	defer resp.Body.Close()

	data, err := robotstxt.FromResponse(resp)
	if err != nil {
		return
	}
	hp.robots = data

	group := data.FindGroup("WebCrawlerThatDreams/1.0")
	if group != nil {
		if delay := group.CrawlDelay; delay > 0 {
			hp.lim.SetLimit(rate.Every(delay))
		}
	}
}

// Utility functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
