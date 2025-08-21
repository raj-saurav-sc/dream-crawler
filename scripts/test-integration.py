#!/usr/bin/env python3
"""
Integration test script for the Web Crawler That Dreams system.
Tests both Go REST API and Python ML API endpoints.
"""

import requests
import json
import time
import sys
from typing import Dict, Any

# Configuration
GO_API_BASE = "http://localhost:8080"
PY_API_BASE = "http://localhost:8001"
TIMEOUT = 10

def test_go_api_health() -> bool:
    """Test Go API health endpoint."""
    try:
        response = requests.get(f"{GO_API_BASE}/health", timeout=TIMEOUT)
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Go API Health: {data.get('status', 'unknown')}")
            return True
        else:
            print(f"❌ Go API Health failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Go API Health error: {e}")
        return False

def test_py_api_health() -> bool:
    """Test Python ML API health endpoint."""
    try:
        response = requests.get(f"{PY_API_BASE}/health", timeout=TIMEOUT)
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Python ML API Health: {data.get('status', 'unknown')}")
            print(f"   Vector Store: {data.get('vector_store', False)}")
            print(f"   Narrative Generator: {data.get('narrative_generator', False)}")
            return True
        else:
            print(f"❌ Python ML API Health failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Python ML API Health error: {e}")
        return False

def test_go_api_crawl() -> bool:
    """Test Go API crawl job creation."""
    try:
        crawl_data = {
            "url": "https://example.com",
            "max_depth": 2,
            "max_pages": 10,
            "rate_limit": 5
        }
        
        response = requests.post(
            f"{GO_API_BASE}/crawl",
            json=crawl_data,
            timeout=TIMEOUT
        )
        
        if response.status_code == 201:
            data = response.json()
            print(f"✅ Go API Crawl Job Created: {data.get('id', 'unknown')}")
            return True
        else:
            print(f"❌ Go API Crawl Job failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Go API Crawl Job error: {e}")
        return False

def test_go_api_search() -> bool:
    """Test Go API search functionality."""
    try:
        response = requests.get(
            f"{GO_API_BASE}/search?q=test&limit=5",
            timeout=TIMEOUT
        )
        
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Go API Search: {data.get('total', 0)} results")
            return True
        else:
            print(f"❌ Go API Search failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Go API Search error: {e}")
        return False

def test_py_api_embedding() -> bool:
    """Test Python ML API embedding generation."""
    try:
        embed_data = {
            "text": "This is a test text for embedding generation."
        }
        
        response = requests.post(
            f"{PY_API_BASE}/embed",
            json=embed_data,
            timeout=TIMEOUT
        )
        
        if response.status_code == 200:
            data = response.json()
            embedding_length = len(data.get('embedding', []))
            print(f"✅ Python ML API Embedding: {embedding_length} dimensions")
            return True
        else:
            print(f"❌ Python ML API Embedding failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Python ML API Embedding error: {e}")
        return False

def test_py_api_semantic_search() -> bool:
    """Test Python ML API semantic search."""
    try:
        search_data = {
            "query": "artificial intelligence and machine learning",
            "limit": 5
        }
        
        response = requests.post(
            f"{PY_API_BASE}/search/semantic",
            json=search_data,
            timeout=TIMEOUT
        )
        
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Python ML API Semantic Search: {data.get('total', 0)} results")
            return True
        else:
            print(f"❌ Python ML API Semantic Search failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Python ML API Semantic Search error: {e}")
        return False

def test_py_api_vector_store_stats() -> bool:
    """Test Python ML API vector store statistics."""
    try:
        response = requests.get(
            f"{PY_API_BASE}/stats/vector-store",
            timeout=TIMEOUT
        )
        
        if response.status_code == 200:
            data = response.json()
            stats = data.get('data', {})
            doc_count = stats.get('documents', {}).get('count', 0)
            dream_count = stats.get('dreams', {}).get('count', 0)
            print(f"✅ Python ML API Vector Store Stats: {doc_count} docs, {dream_count} dreams")
            return True
        else:
            print(f"❌ Python ML API Vector Store Stats failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Python ML API Vector Store Stats error: {e}")
        return False

def main():
    """Run all integration tests."""
    print("🌌 Web Crawler That Dreams - Integration Tests")
    print("=" * 50)
    
    tests = [
        ("Go API Health", test_go_api_health),
        ("Python ML API Health", test_py_api_health),
        ("Go API Crawl Job", test_go_api_crawl),
        ("Go API Search", test_go_api_search),
        ("Python ML API Embedding", test_py_api_embedding),
        ("Python ML API Semantic Search", test_py_api_semantic_search),
        ("Python ML API Vector Store Stats", test_py_api_vector_store_stats),
    ]
    
    passed = 0
    total = len(tests)
    
    for test_name, test_func in tests:
        print(f"\n🧪 Testing: {test_name}")
        if test_func():
            passed += 1
        time.sleep(1)  # Small delay between tests
    
    print("\n" + "=" * 50)
    print(f"📊 Test Results: {passed}/{total} tests passed")
    
    if passed == total:
        print("🎉 All tests passed! System is working correctly.")
        return 0
    else:
        print("⚠️  Some tests failed. Check service logs for details.")
        return 1

if __name__ == "__main__":
    sys.exit(main())
