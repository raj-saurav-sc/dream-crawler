import argparse
import json
import logging
import os
import signal
import sys

from dataclasses import dataclass, field
from typing import List

from narrative import NarrativeGenerator
from confluent_kafka import Consumer, KafkaError, KafkaException

# --- Configuration ---
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)],
)

# --- Graceful Shutdown ---
running = True

def shutdown_handler(signum, frame):
    """Handle signals for a graceful shutdown."""
    global running
    logging.info("Shutdown signal received, stopping consumer...")
    running = False

signal.signal(signal.SIGINT, shutdown_handler)
signal.signal(signal.SIGTERM, shutdown_handler)

# --- Data Models ---
@dataclass
class ContentChunk:
    """Represents a semantic chunk of content, mirroring the Go struct."""
    id: str
    type: str  # headline, paragraph, quote, etc.
    text: str
    position: int
    confidence: float
    keywords: List[str] = field(default_factory=list)
    sentiment: str = ""
    entities: List[str] = field(default_factory=list)

@dataclass
class DreamingHints:
    """Represents the AI dream hints, mirroring the Go struct."""
    surrealism_potential: float = 0.0
    themes: List[str] = field(default_factory=list)
    emotions: List[str] = field(default_factory=list)
    motifs: List[str] = field(default_factory=list)
    tone: str = ""
    complexity: float = 0.0
    visual_cues: List[str] = field(default_factory=list)
    audio_cues: List[str] = field(default_factory=list)
    color_palette: List[str] = field(default_factory=list)
    abstractness: float = 0.0

@dataclass
class Document:
    """Represents a crawled document, mirroring the Go struct."""
    url: str
    dream_hints: DreamingHints
    title: str = ""
    text: str = ""
    clean_text: str = ""
    fetched_at: str = ""
    status: int = 0
    content_hash: str = ""
    chunks: List[ContentChunk] = field(default_factory=list)

def process_dream_seed(doc: Document, generator: NarrativeGenerator):
    """
    This is where the AI dream processing will happen.
    Now uses a typed Document object for clarity and safety.
    """
    logging.info("🔮 Processing dream seed for URL: %s", doc.url)
    logging.info("   Surrealism Potential: %.2f", doc.dream_hints.surrealism_potential)

    # Generate the surreal narrative
    logging.info("   Generating dream narrative...")
    narrative = generator.generate(doc)
    logging.info("   --- Dream Start ---")
    logging.info(narrative)
    logging.info("   --- Dream End ---")
    logging.info("-" * 40)


def main(broker: str, group_id: str, topic: str):
    """Main function to run the Kafka consumer."""
    conf = {
        "bootstrap.servers": broker,
        "group.id": group_id,
        "auto.offset.reset": "earliest",
        "enable.auto.commit": False,  # We will commit offsets manually
    }

    # Initialize the narrative generator
    try:
        generator = NarrativeGenerator()
        logging.info("Narrative generator loaded successfully.")
    except Exception as e:
        logging.error(f"Failed to load narrative generator: {e}", exc_info=True)
        sys.exit(1)

    consumer = Consumer(conf)
    logging.info(f"Kafka consumer configured for broker {broker} and group {group_id}")

    try:
        consumer.subscribe([topic])
        logging.info(f"Subscribed to topic: {topic}")

        while running:
            msg = consumer.poll(timeout=1.0)
            if msg is None:
                continue

            if msg.error():
                if msg.error().code() == KafkaError._PARTITION_EOF:
                    logging.info(f"Reached end of partition: {msg.topic()} [{msg.partition()}]")
                else:
                    raise KafkaException(msg.error())
            else:
                try:
                    # Deserialize into a dictionary first
                    raw_doc = json.loads(msg.value().decode("utf-8"))

                    # Pop nested objects and create their dataclasses
                    hints_data = raw_doc.pop("dream_hints", {}) or {}
                    hints = DreamingHints(**hints_data)

                    chunks_data = raw_doc.pop("chunks", []) or []
                    chunks = [ContentChunk(**c) for c in chunks_data]

                    # Create the main dataclass, passing the nested object
                    # and unpacking the rest of the dictionary.
                    # We ignore metadata, links, etc. for now.
                    main_fields = {k: v for k, v in raw_doc.items() if k in Document.__annotations__}
                    document = Document(dream_hints=hints, chunks=chunks, **main_fields)

                    process_dream_seed(document, generator)
                    consumer.commit(asynchronous=True)
                except (json.JSONDecodeError, TypeError, KeyError) as e:
                    logging.error(f"Failed to decode or deserialize message: {e}", exc_info=True)
                except Exception as e:
                    logging.error(f"Error processing message: {e}", exc_info=True)
    finally:
        logging.info("Closing Kafka consumer.")
        consumer.close()


if __name__ == "__main__":
    broker = os.environ.get("KAFKA_BROKER", "localhost:9092")
    topic = os.environ.get("KAFKA_DREAM_TOPIC", "dream.seeds")
    group_id = "dream-processor-group"

    main(broker, group_id, topic)