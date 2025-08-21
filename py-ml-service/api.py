import logging
import os
import uuid
from datetime import datetime
from typing import List, Optional, Dict, Any

from fastapi import FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
import uvicorn

from dream_processor import Document  # use dataclass for document only
from narrative import NarrativeGenerator  # correct import location
from vector_store import VectorStore

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Web Crawler That Dreams - ML Service",
    description="AI/ML service for generating dreams and semantic search",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Initialize services
vector_store = VectorStore(
    qdrant_url=os.getenv("QDRANT_HOST", "localhost"),
    qdrant_port=int(os.getenv("QDRANT_PORT", "6333"))
)

narrative_generator = None
try:
    narrative_generator = NarrativeGenerator()
    logger.info("Narrative generator initialized successfully")
except Exception as e:
    logger.warning(f"Could not initialize narrative generator: {e}")

# Pydantic models
class EmbeddingRequest(BaseModel):
    text: str = Field(..., description="Text to generate embedding for")

class EmbeddingResponse(BaseModel):
    embedding: List[float] = Field(..., description="Generated embedding vector")
    model: str = Field(..., description="Model used for embedding")

class SearchRequest(BaseModel):
    query: str = Field(..., description="Search query")
    limit: int = Field(10, description="Maximum number of results")
    filters: Optional[Dict[str, Any]] = Field(None, description="Search filters")

class SearchResult(BaseModel):
    id: str
    score: float
    payload: Dict[str, Any]

class SearchResponse(BaseModel):
    query: str
    results: List[SearchResult]
    total: int
    search_type: str

class DreamRequest(BaseModel):
    document_id: str = Field(..., description="ID of the document to dream about")
    url: str = Field(..., description="URL of the document")
    title: str = Field(..., description="Title of the document")
    content: str = Field(..., description="Content of the document")
    dream_hints: Optional[Dict[str, Any]] = Field(None, description="Hints for dream generation")

class DreamResponse(BaseModel):
    dream_id: str
    narrative: str
    confidence: float
    model: str
    generated_at: datetime

class HealthResponse(BaseModel):
    status: str
    timestamp: datetime
    service: str
    vector_store: bool
    narrative_generator: bool

# Health check endpoint
@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Check the health of the ML service."""
    return HealthResponse(
        status="healthy",
        timestamp=datetime.utcnow(),
        service="ml-service",
        vector_store=vector_store is not None,
        narrative_generator=narrative_generator is not None
    )

# Embedding endpoint
@app.post("/embed", response_model=EmbeddingResponse)
async def generate_embedding(request: EmbeddingRequest):
    """Generate embedding for given text."""
    try:
        embedding = vector_store.get_embedding(request.text)
        return EmbeddingResponse(
            embedding=embedding,
            model="all-MiniLM-L6-v2"
        )
    except Exception as e:
        logger.error(f"Error generating embedding: {e}")
        raise HTTPException(status_code=500, detail="Failed to generate embedding")

# Semantic search endpoint
@app.post("/search/semantic", response_model=SearchResponse)
async def semantic_search(request: SearchRequest):
    """Perform semantic search on documents."""
    try:
        results = vector_store.search_documents(
            query=request.query,
            limit=request.limit,
            filters=request.filters
        )
        
        # Convert to response format
        search_results = [
            SearchResult(
                id=result["id"],
                score=result["score"],
                payload=result["payload"]
            )
            for result in results
        ]
        
        return SearchResponse(
            query=request.query,
            results=search_results,
            total=len(search_results),
            search_type="semantic"
        )
    except Exception as e:
        logger.error(f"Error performing semantic search: {e}")
        raise HTTPException(status_code=500, detail="Failed to perform semantic search")

# Dream search endpoint
@app.post("/search/dreams", response_model=SearchResponse)
async def search_dreams(request: SearchRequest):
    """Search dreams by semantic similarity."""
    try:
        results = vector_store.search_dreams(
            query=request.query,
            limit=request.limit
        )
        
        # Convert to response format
        search_results = [
            SearchResult(
                id=result["id"],
                score=result["score"],
                payload=result["payload"]
            )
            for result in results
        ]
        
        return SearchResponse(
            query=request.query,
            results=search_results,
            total=len(search_results),
            search_type="dream"
        )
    except Exception as e:
        logger.error(f"Error searching dreams: {e}")
        raise HTTPException(status_code=500, detail="Failed to search dreams")

# Dream generation endpoint
@app.post("/dream", response_model=DreamResponse)
async def generate_dream(request: DreamRequest):
    """Generate a dream narrative for a document."""
    if narrative_generator is None:
        raise HTTPException(
            status_code=503, 
            detail="Narrative generator not available"
        )
    
    try:
        # Create document object
        doc = Document(
            url=request.url,
            title=request.title,
            text=request.content,
            clean_text=request.content,
            fetched_at=datetime.utcnow().isoformat(),
            status=200,
            content_hash=str(uuid.uuid4()),
            chunks=[],  # Will be processed by the generator
            dream_hints=request.dream_hints or {}
        )
        
        # Generate dream narrative
        narrative = narrative_generator.generate(doc)
        
        # Generate dream ID
        dream_id = str(uuid.uuid4())
        
        # Store in vector database
        metadata = {
            "document_id": request.document_id,
            "url": request.url,
            "title": request.title,
            "generated_at": datetime.utcnow().isoformat()
        }
        
        vector_store.store_dream(
            dream_id=dream_id,
            doc_id=request.document_id,
            narrative=narrative,
            confidence=0.85,  # Default confidence
            metadata=metadata
        )
        
        return DreamResponse(
            dream_id=dream_id,
            narrative=narrative,
            confidence=0.85,
            model="tinyllama-1.1b-chat",
            generated_at=datetime.utcnow()
        )
        
    except Exception as e:
        logger.error(f"Error generating dream: {e}")
        raise HTTPException(status_code=500, detail="Failed to generate dream")

# Get similar dreams endpoint
@app.get("/dreams/{dream_id}/similar", response_model=List[SearchResult])
async def get_similar_dreams(
    dream_id: str,
    limit: int = Query(5, description="Maximum number of similar dreams")
):
    """Find dreams similar to a given dream."""
    try:
        results = vector_store.get_similar_dreams(dream_id, limit)
        
        return [
            SearchResult(
                id=result["id"],
                score=result["score"],
                payload=result["payload"]
            )
            for result in results
        ]
    except Exception as e:
        logger.error(f"Error finding similar dreams: {e}")
        raise HTTPException(status_code=500, detail="Failed to find similar dreams")

# Vector store statistics endpoint
@app.get("/stats/vector-store")
async def get_vector_store_stats():
    """Get statistics about the vector store."""
    try:
        stats = vector_store.get_collection_stats()
        return {
            "status": "success",
            "data": stats,
            "timestamp": datetime.utcnow()
        }
    except Exception as e:
        logger.error(f"Error getting vector store stats: {e}")
        raise HTTPException(status_code=500, detail="Failed to get vector store stats")

# Store document endpoint
@app.post("/documents")
async def store_document(
    document_id: str,
    url: str,
    title: str,
    content: str,
    metadata: Optional[Dict[str, Any]] = None
):
    """Store a document in the vector store."""
    try:
        success = vector_store.store_document(
            doc_id=document_id,
            url=url,
            title=title,
            content=content,
            metadata=metadata or {}
        )
        
        if success:
            return {
                "status": "success",
                "message": f"Document {document_id} stored successfully",
                "timestamp": datetime.utcnow()
            }
        else:
            raise HTTPException(status_code=500, detail="Failed to store document")
            
    except Exception as e:
        logger.error(f"Error storing document: {e}")
        raise HTTPException(status_code=500, detail="Failed to store document")

if __name__ == "__main__":
    uvicorn.run(
        "api:app",
        host="0.0.0.0",
        port=8000,
        reload=True
    )
