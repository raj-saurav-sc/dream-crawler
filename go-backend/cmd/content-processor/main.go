package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/drawnparadox/web-crawler-that-dreams/go-backend/pkg/model"
)

var (
	kafkaBroker = flag.String("kafka-broker", "localhost:9092", "Kafka broker address")
	groupID     = flag.String("group-id", "content-processor", "Kafka consumer group ID")
)

type ContentProcessor struct {
	consumer *kafka.Consumer
	producer *kafka.Producer
}

func NewContentProcessor(broker, groupID string) (*ContentProcessor, error) {
	// Consumer config
	consumerConfig := &kafka.ConfigMap{
		"bootstrap.servers":  broker,
		"group.id":           groupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	}

	consumer, err := kafka.NewConsumer(consumerConfig)
	if err != nil {
		return nil, err
	}

	// Producer config
	producerConfig := &kafka.ConfigMap{
		"bootstrap.servers": broker,
	}

	producer, err := kafka.NewProducer(producerConfig)
	if err != nil {
		consumer.Close()
		return nil, err
	}

	return &ContentProcessor{
		consumer: consumer,
		producer: producer,
	}, nil
}

func (cp *ContentProcessor) Start() error {
	// Subscribe to raw content topic
	err := cp.consumer.Subscribe(model.TopicRawContent, nil)
	if err != nil {
		return err
	}

	log.Println("Content processor started, consuming from:", model.TopicRawContent)

	for {
		msg, err := cp.consumer.ReadMessage(-1)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}

		// Process the message
		go cp.processMessage(msg)
	}
}

func (cp *ContentProcessor) processMessage(msg *kafka.Message) {
	var document model.Document
	if err := json.Unmarshal(msg.Value, &document); err != nil {
		log.Printf("Error unmarshaling document: %v", err)
		return
	}

	log.Printf("Processing document: %s", document.URL)

	// Clean and normalize the content
	cleanedDoc := cp.cleanDocument(document)

	// Publish to clean content topic
	cleanedData, err := json.Marshal(cleanedDoc)
	if err != nil {
		log.Printf("Error marshaling cleaned document: %v", err)
		return
	}

	topic := model.TopicCleanContent
	cp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: cleanedData,
	}, nil)

	// Commit the offset
	cp.consumer.CommitMessage(msg)
}

func (cp *ContentProcessor) cleanDocument(doc model.Document) model.Document {
	// Clean text content
	doc.CleanText = cp.cleanText(doc.Text)

	// Extract and enhance metadata
	doc.Metadata = cp.enhanceMetadata(doc.Metadata, doc.Text)

	// Process content chunks
	doc.Chunks = cp.processChunks(doc.Text)

	// Analyze content for dreaming hints
	doc.DreamHints = cp.analyzeDreamHints(doc)

	return doc
}

func (cp *ContentProcessor) cleanText(text string) string {
	// Remove extra whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Remove common HTML artifacts
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")

	// Remove excessive punctuation
	text = strings.ReplaceAll(text, "!!", "!")
	text = strings.ReplaceAll(text, "??", "?")

	return strings.TrimSpace(text)
}

func (cp *ContentProcessor) enhanceMetadata(metadata model.DocumentMetadata, text string) model.DocumentMetadata {
	// Count words
	words := strings.Fields(text)
	metadata.WordCount = len(words)

	// Detect language (simple heuristic)
	if strings.Contains(text, "the") || strings.Contains(text, "and") || strings.Contains(text, "of") {
		metadata.Language = "en"
	}

	// Extract tags from common patterns
	tags := []string{}
	if strings.Contains(strings.ToLower(text), "technology") {
		tags = append(tags, "technology")
	}
	if strings.Contains(strings.ToLower(text), "science") {
		tags = append(tags, "science")
	}
	if strings.Contains(strings.ToLower(text), "art") {
		tags = append(tags, "art")
	}
	metadata.Tags = tags

	return metadata
}

func (cp *ContentProcessor) processChunks(text string) []model.ContentChunk {
	chunks := []model.ContentChunk{}
	sentences := strings.Split(text, ". ")

	for i, sentence := range sentences {
		if len(strings.TrimSpace(sentence)) < 10 {
			continue
		}

		chunkType := "paragraph"
		if i == 0 || strings.Contains(strings.ToUpper(sentence), "BREAKING") {
			chunkType = "headline"
		}

		chunks = append(chunks, model.ContentChunk{
			ID:         fmt.Sprintf("chunk_%d", i),
			Type:       chunkType,
			Text:       strings.TrimSpace(sentence),
			Position:   i,
			Confidence: 0.8,
		})
	}

	return chunks
}

func (cp *ContentProcessor) analyzeDreamHints(doc model.Document) model.DreamingHints {
	hints := model.DreamingHints{}

	text := strings.ToLower(doc.Text)

	// Analyze emotions
	emotions := []string{}
	if strings.Contains(text, "amazing") || strings.Contains(text, "wonderful") {
		emotions = append(emotions, "wonder")
	}
	if strings.Contains(text, "fear") || strings.Contains(text, "terrifying") {
		emotions = append(emotions, "fear")
	}
	if strings.Contains(text, "love") || strings.Contains(text, "beautiful") {
		emotions = append(emotions, "love")
	}
	hints.Emotions = emotions

	// Analyze themes
	themes := []string{}
	if strings.Contains(text, "future") || strings.Contains(text, "technology") {
		themes = append(themes, "futurism")
	}
	if strings.Contains(text, "nature") || strings.Contains(text, "earth") {
		themes = append(themes, "nature")
	}
	if strings.Contains(text, "space") || strings.Contains(text, "cosmos") {
		themes = append(themes, "cosmos")
	}
	hints.Themes = themes

	// Calculate surrealism potential
	surrealism := 0.0
	if len(hints.Emotions) > 0 {
		surrealism += 0.3
	}
	if len(hints.Themes) > 0 {
		surrealism += 0.3
	}
	if doc.Metadata.WordCount > 500 {
		surrealism += 0.2
	}
	hints.Surrealism = surrealism

	return hints
}

func (cp *ContentProcessor) Close() {
	if cp.consumer != nil {
		cp.consumer.Close()
	}
	if cp.producer != nil {
		cp.producer.Close()
	}
}

func main() {
	flag.Parse()

	processor, err := NewContentProcessor(*kafkaBroker, *groupID)
	if err != nil {
		log.Fatalf("Failed to create content processor: %v", err)
	}
	defer processor.Close()

	if err := processor.Start(); err != nil {
		log.Fatalf("Failed to start content processor: %v", err)
	}
}
