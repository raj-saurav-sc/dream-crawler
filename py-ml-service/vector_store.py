import logging
import os
import hashlib
from typing import List, Optional, Dict, Any

from qdrant_client import QdrantClient
from qdrant_client.models import (
	Distance, VectorParams, PointStruct, Filter, FieldCondition, MatchValue
)

logger = logging.getLogger(__name__)

# Optional heavy dependency; may not be available on constrained systems
try:
	from sentence_transformers import SentenceTransformer  # type: ignore
	ST_AVAILABLE = True
except Exception:
	SentenceTransformer = None  # type: ignore
	ST_AVAILABLE = False


class VectorStore:
	"""
	Manages vector storage and retrieval using Qdrant.
	Handles both document embeddings and dream outputs.
	"""
	
	def __init__(self, qdrant_url: str = "localhost", qdrant_port: int = 6333):
		self.client = QdrantClient(host=qdrant_url, port=qdrant_port)
		self.embedding_model = None
		self.embedding_size = 384

		# Lazy-load model unless disabled
		if os.environ.get("DISABLE_EMBEDDINGS") != "1" and ST_AVAILABLE:
			try:
				self.embedding_model = SentenceTransformer(os.environ.get("EMBEDDINGS_MODEL", 'all-MiniLM-L6-v2'))
				logger.info("Loaded sentence-transformers model.")
			except Exception as e:
				logger.warning("Failed to load embeddings model (%s); using hashing fallback.", e)
		else:
			if os.environ.get("DISABLE_EMBEDDINGS") == "1":
				logger.warning("DISABLE_EMBEDDINGS=1 set; using hashing fallback embeddings.")
			else:
				logger.warning("sentence-transformers not available; using hashing fallback embeddings.")
		
		# Collection names
		self.documents_collection = "documents"
		self.dreams_collection = "dreams"
		
		# Initialize collections
		self._init_collections()
	
	def _init_collections(self):
		"""Initialize Qdrant collections if they don't exist."""
		for collection in (self.documents_collection, self.dreams_collection):
			try:
				self.client.get_collection(collection)
				logger.info("Collection '%s' already exists", collection)
			except Exception:
				self.client.create_collection(
					collection_name=collection,
					vectors_config=VectorParams(
						size=self.embedding_size,
						distance=Distance.COSINE
					)
				)
				logger.info("Created collection '%s'", collection)
	
	def _hashing_embedding(self, text: str) -> List[float]:
		"""A lightweight, deterministic fallback embedding using hashing.
		This is not semantically meaningful but enables the pipeline and testing.
		"""
		if not text:
			text = " "
		digest = hashlib.sha256(text.encode("utf-8")).digest()
		# Repeat digest bytes to reach embedding_size and normalize
		vec = list(digest) * (self.embedding_size // len(digest) + 1)
		vec = vec[:self.embedding_size]
		norm = sum(v * v for v in vec) ** 0.5 or 1.0
		return [v / norm for v in vec]
	
	def get_embedding(self, text: str) -> List[float]:
		"""Generate embedding for given text, with hashing fallback."""
		try:
			if self.embedding_model is not None:
				emb = self.embedding_model.encode(text)
				return emb.tolist() if hasattr(emb, 'tolist') else list(emb)
		except Exception as e:
			logger.warning("Embedding model failed (%s); using fallback.", e)
		return self._hashing_embedding(text)
	
	def store_document(self, doc_id: str, url: str, title: str, content: str, 
					  metadata: Dict[str, Any]) -> bool:
		"""Store document with its embedding."""
		try:
			text_for_embedding = f"{title} {content}"
			embedding = self.get_embedding(text_for_embedding)
			self.client.upsert(
				collection_name=self.documents_collection,
				points=[
					PointStruct(
						id=doc_id,
						vector=embedding,
						payload={
							"url": url,
							"title": title,
							"content": content,
							"metadata": metadata,
							"type": "document"
						}
					)
				]
			)
			logger.info("Stored document %s in vector store", doc_id)
			return True
		except Exception as e:
			logger.error("Error storing document %s: %s", doc_id, e)
			return False
	
	def store_dream(self, dream_id: str, doc_id: str, narrative: str, 
				  confidence: float, metadata: Dict[str, Any]) -> bool:
		"""Store dream output with its embedding."""
		try:
			embedding = self.get_embedding(narrative)
			self.client.upsert(
				collection_name=self.dreams_collection,
				points=[
					PointStruct(
						id=dream_id,
						vector=embedding,
						payload={
							"doc_id": doc_id,
							"narrative": narrative,
							"confidence": confidence,
							"metadata": metadata,
							"type": "dream"
						}
					)
				]
			)
			logger.info("Stored dream %s in vector store", dream_id)
			return True
		except Exception as e:
			logger.error("Error storing dream %s: %s", dream_id, e)
			return False
	
	def search_documents(self, query: str, limit: int = 10, 
					   filters: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
		"""Search documents by semantic similarity."""
		try:
			query_embedding = self.get_embedding(query)
			search_filter = None
			if filters:
				conditions = [FieldCondition(key=k, match=MatchValue(value=v)) for k, v in filters.items()]
				if conditions:
					search_filter = Filter(must=conditions)
			search_result = self.client.search(
				collection_name=self.documents_collection,
				query_vector=query_embedding,
				query_filter=search_filter,
				limit=limit
			)
			return [{"id": p.id, "score": p.score, "payload": p.payload} for p in search_result]
		except Exception as e:
			logger.error("Error searching documents: %s", e)
			return []
	
	def search_dreams(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
		"""Search dreams by semantic similarity."""
		try:
			query_embedding = self.get_embedding(query)
			search_result = self.client.search(
				collection_name=self.dreams_collection,
				query_vector=query_embedding,
				limit=limit
			)
			return [{"id": p.id, "score": p.score, "payload": p.payload} for p in search_result]
		except Exception as e:
			logger.error("Error searching dreams: %s", e)
			return []
	
	def get_similar_dreams(self, dream_id: str, limit: int = 5) -> List[Dict[str, Any]]:
		"""Find dreams similar to a given dream."""
		try:
			dream = self.client.retrieve(collection_name=self.dreams_collection, ids=[dream_id])
			if not dream:
				return []
			dream_vector = dream[0].vector
			search_result = self.client.search(
				collection_name=self.dreams_collection,
				query_vector=dream_vector,
				limit=limit + 1
			)
			results = []
			for p in search_result:
				if p.id != dream_id:
					results.append({"id": p.id, "score": p.score, "payload": p.payload})
			return results[:limit]
		except Exception as e:
			logger.error("Error finding similar dreams: %s", e)
			return []
	
	def get_collection_stats(self) -> Dict[str, Any]:
		"""Get statistics about the collections."""
		try:
			doc_stats = self.client.get_collection(self.documents_collection)
			dream_stats = self.client.get_collection(self.dreams_collection)
			return {
				"documents": {
					"count": doc_stats.points_count,
					"vectors_count": doc_stats.vectors_count,
				},
				"dreams": {
					"count": dream_stats.points_count,
					"vectors_count": dream_stats.vectors_count,
				},
			}
		except Exception as e:
			logger.error("Error getting collection stats: %s", e)
			return {}
