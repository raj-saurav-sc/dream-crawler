#!/usr/bin/env python3
"""
Simple test script for ML API functionality without uvicorn
"""

import os
import sys
sys.path.append('./py-ml-service')

# Set environment variables
os.environ['DISABLE_LLM'] = '1'
os.environ['DISABLE_EMBEDDINGS'] = '1'

try:
    print("Testing ML API imports...")
    from vector_store import VectorStore
    print("✅ VectorStore imported successfully")
    
    from narrative import NarrativeGenerator
    print("✅ NarrativeGenerator imported successfully")
    
    print("\nTesting VectorStore initialization...")
    vs = VectorStore()
    print("✅ VectorStore initialized successfully")
    
    print("\nTesting embedding generation...")
    embedding = vs.get_embedding("Hello world")
    print(f"✅ Generated embedding with {len(embedding)} dimensions")
    
    print("\nTesting document storage...")
    success = vs.store_document(
        doc_id="test_123",
        url="https://news.ycombinator.com/item?id=123",
        title="Test Article",
        content="This is a test article about artificial intelligence and machine learning.",
        metadata={"source": "test"}
    )
    print(f"✅ Document storage: {success}")
    
    print("\nTesting semantic search...")
    results = vs.search_documents("artificial intelligence", limit=5)
    print(f"✅ Search returned {len(results)} results")
    
    print("\nTesting narrative generation...")
    ng = NarrativeGenerator()
    print("✅ NarrativeGenerator created successfully")
    
    print("\nAll tests passed! 🎉")
    
except Exception as e:
    print(f"❌ Test failed: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)
