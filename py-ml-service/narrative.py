import logging
import os
from typing import List

# Delay heavy import; make optional
try:
	from ctransformers import AutoModelForCausalLM  # type: ignore
	CTRANS_AVAILABLE = True
except Exception as e:
	CTRANS_AVAILABLE = False
	AutoModelForCausalLM = None  # type: ignore

from dream_processor import Document, ContentChunk

logger = logging.getLogger(__name__)


class NarrativeGenerator:
	"""
	Uses a local language model to generate surreal narratives from crawled documents.
	If the model is unavailable, generates a simple placeholder narrative.
	"""

	def __init__(self):
		"""
		Initializes the generator and loads the language model from a path
		specified by the MODEL_PATH environment variable.
		Set DISABLE_LLM=1 to skip model loading.
		"""
		self.llm = None
		if os.environ.get("DISABLE_LLM") == "1":
			logger.warning("DISABLE_LLM=1 set; skipping LLM loading.")
			return

		model_path = os.environ.get("MODEL_PATH")
		if not model_path:
			logger.warning("MODEL_PATH not set; NarrativeGenerator will use stub output.")
			return

		if not CTRANS_AVAILABLE:
			logger.warning("ctransformers not available; NarrativeGenerator will use stub output.")
			return

		if not os.path.exists(model_path):
			logger.warning("Model file not found at path: %s; using stub output.", model_path)
			return

		try:
			logger.info("Loading model from: %s", model_path)
			config = {
				"max_new_tokens": 256,
				"repetition_penalty": 1.15,
				"temperature": 0.8,
				"top_k": 40,
				"top_p": 0.9,
				"stream": False,
			}
			# Use safe defaults; lib selection may vary by environment
			self.llm = AutoModelForCausalLM.from_pretrained(  # type: ignore
				model_path,
				model_type="llama",
				lib=os.environ.get("CTRANS_LIB", "avx2"),
				**config,
			)
			logger.info("Narrative model loaded successfully.")
		except Exception as e:
			logger.warning("Failed to load LLM (%s); falling back to stub output.", e)
			self.llm = None

	def _create_prompt(self, doc: Document) -> str:
		"""Creates a rich, structured prompt for the language model."""
		relevant_chunks: List[ContentChunk] = [
			c for c in doc.chunks if c.type in ["headline", "paragraph"]
		][:4]

		chunk_text = "\n".join([f"- {c.text}" for c in relevant_chunks])

		return f"""
You are a surrealist poet. Your task is to read the provided web page content and generate a short, surreal, dream-like narrative based on it.

**Source Content Analysis:**
- Title: {doc.title}
- Key Themes: {', '.join(doc.dream_hints.themes) if hasattr(doc.dream_hints, 'themes') else ''}
- Detected Motifs: {', '.join(doc.dream_hints.motifs) if hasattr(doc.dream_hints, 'motifs') else ''}

**Source Text Snippets:**
{chunk_text}

**Your Task:**
Weave these elements into a strange and evocative dream narrative. The narrative should be abstract and metaphorical, not a literal summary. Write a single, dense paragraph.

**Dream Narrative:**
""".strip()

	def generate(self, doc: Document) -> str:
		"""Generates a surreal narrative for the given document."""
		prompt = self._create_prompt(doc)
		if self.llm is None:
			# Fallback stub to keep service healthy
			return (
				f"A lucid fragment drifts across {doc.title or doc.url}, where ideas echo like constellations. "
				"Shadows of meaning cross a quiet lake; symbols rearrange until the night exhales a metaphor."
			)
		try:
			response = self.llm(prompt)  # type: ignore
			return str(response).strip()
		except Exception as e:
			logger.error("Error during model inference: %s", e)
			return (
				"A dream flickered, but was lost in the static; the moon kept its counsel."
			)