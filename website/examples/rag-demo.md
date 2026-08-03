# RAG (Retrieval-Augmented Generation) Demo

## Overview

This example demonstrates how to build a RAG system using HNO. RAG combines information retrieval from a knowledge base with LLM text generation to provide accurate, grounded answers. The system uses ChromaDB for vector storage, OpenAI embeddings, and a custom RAG toolkit to enable semantic search capabilities.

## What You'll Learn

- How to create and use OpenAI embeddings
- How to set up ChromaDB as a vector database
- How to chunk documents for optimal retrieval
- How to build a custom RAG toolkit for agents
- How to create an agent with knowledge retrieval capabilities
- Best practices for RAG implementation

## Prerequisites

- Go 1.21 or higher
- OpenAI API key
- ChromaDB running locally (via Docker)

## Setup

1. Set your OpenAI API key:
```bash
export OPENAI_API_KEY=sk-your-api-key-here
```

2. Start ChromaDB using Docker:
```bash
docker pull chromadb/chroma
docker run -p 8000:8000 chromadb/chroma
```

3. Navigate to the example directory:
```bash
cd cmd/examples/rag_demo
```

## Complete Code

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	openaiembed "github.com/rexleimo/agno-go/pkg/hno/embeddings/openai"
	"github.com/rexleimo/agno-go/pkg/hno/knowledge"
	openaimodel "github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
	"github.com/rexleimo/agno-go/pkg/hno/vectordb/chromadb"
)

// RAGToolkit provides knowledge retrieval tools for the agent
type RAGToolkit struct {
	*toolkit.BaseToolkit
	vectorDB vectordb.VectorDB
}

// NewRAGToolkit creates a new RAG toolkit
func NewRAGToolkit(db vectordb.VectorDB) *RAGToolkit {
	t := &RAGToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("knowledge_retrieval"),
		vectorDB:    db,
	}

	// Register search function
	t.RegisterFunction(&toolkit.Function{
		Name:        "search_knowledge",
		Description: "Search the knowledge base for relevant information. Use this to find answers to user questions.",
		Parameters: map[string]toolkit.Parameter{
			"query": {
				Type:        "string",
				Description: "The search query or question",
				Required:    true,
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of results to return (default: 3)",
				Required:    false,
			},
		},
		Handler: t.searchKnowledge,
	})

	return t
}

func (t *RAGToolkit) searchKnowledge(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	limit := 3
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	results, err := t.vectorDB.Query(ctx, query, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge base: %w", err)
	}

	// Format results for the agent
	var formattedResults []map[string]interface{}
	for i, result := range results {
		formattedResults = append(formattedResults, map[string]interface{}{
			"rank":     i + 1,
			"content":  result.Content,
			"score":    result.Score,
			"metadata": result.Metadata,
		})
	}

	return formattedResults, nil
}

func main() {
	fmt.Println("🚀 RAG (Retrieval-Augmented Generation) Demo")
	fmt.Println("This example demonstrates:")
	fmt.Println("1. Loading documents from files")
	fmt.Println("2. Chunking text into smaller pieces")
	fmt.Println("3. Generating embeddings with OpenAI")
	fmt.Println("4. Storing in ChromaDB vector database")
	fmt.Println("5. Using RAG with an Agent to answer questions")
	fmt.Println()

	// Check environment variables
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	ctx := context.Background()

	// Step 1: Create embedding function
	fmt.Println("📊 Step 1: Creating OpenAI embedding function...")
	embedFunc, err := openaiembed.New(openaiembed.Config{
		APIKey: openaiKey,
		Model:  "text-embedding-3-small",
	})
	if err != nil {
		log.Fatalf("Failed to create embedding function: %v", err)
	}
	fmt.Printf("   ✅ Created embedding function (model: %s, dimensions: %d)\n\n",
		embedFunc.GetModel(), embedFunc.GetDimensions())

	// Step 2: Create ChromaDB vector database
	fmt.Println("💾 Step 2: Connecting to ChromaDB...")
	db, err := chromadb.New(chromadb.Config{
		BaseURL:           "http://localhost:8000",
		CollectionName:    "rag_demo",
		EmbeddingFunction: embedFunc,
	})
	if err != nil {
		log.Fatalf("Failed to create ChromaDB: %v", err)
	}
	defer db.Close()

	// Create collection
	err = db.CreateCollection(ctx, "", map[string]interface{}{
		"description": "RAG demo knowledge base",
	})
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}
	fmt.Println("   ✅ Connected to ChromaDB and created collection")

	// Step 3: Load and process documents
	fmt.Println("📚 Step 3: Loading and processing documents...")

	// Sample documents about AI and ML
	sampleDocs := []knowledge.Document{
		{
			ID:      "doc1",
			Content: "Artificial Intelligence (AI) is the simulation of human intelligence by machines. AI systems can perform tasks that typically require human intelligence, such as visual perception, speech recognition, decision-making, and language translation. Modern AI is based on machine learning algorithms that can learn from data.",
			Metadata: map[string]interface{}{
				"topic": "AI Overview",
				"date":  "2025-01-01",
			},
		},
		{
			ID:      "doc2",
			Content: "Machine Learning (ML) is a subset of AI that focuses on creating systems that learn from data. Instead of being explicitly programmed, ML models improve their performance through experience. Common ML algorithms include neural networks, decision trees, and support vector machines.",
			Metadata: map[string]interface{}{
				"topic": "Machine Learning",
				"date":  "2025-01-01",
			},
		},
		{
			ID:      "doc3",
			Content: "Vector databases are specialized databases designed to store and query high-dimensional vector embeddings. They enable semantic search by finding similar vectors using distance metrics like cosine similarity or Euclidean distance. Vector databases are essential for RAG (Retrieval-Augmented Generation) systems.",
			Metadata: map[string]interface{}{
				"topic": "Vector Databases",
				"date":  "2025-01-01",
			},
		},
		{
			ID:      "doc4",
			Content: "Retrieval-Augmented Generation (RAG) combines information retrieval with text generation. It first retrieves relevant documents from a knowledge base, then uses a language model to generate responses based on the retrieved context. RAG improves accuracy and reduces hallucinations in AI systems.",
			Metadata: map[string]interface{}{
				"topic": "RAG",
				"date":  "2025-01-01",
			},
		},
		{
			ID:      "doc5",
			Content: "Large Language Models (LLMs) like GPT-4 are neural networks trained on vast amounts of text data. They can understand and generate human-like text, perform reasoning, answer questions, and even write code. LLMs are the foundation of modern AI assistants and chatbots.",
			Metadata: map[string]interface{}{
				"topic": "Large Language Models",
				"date":  "2025-01-01",
			},
		},
	}

	// Chunk documents (optional, useful for large documents)
	chunker := knowledge.NewCharacterChunker(500, 50)
	var allChunks []knowledge.Chunk
	for _, doc := range sampleDocs {
		chunks, err := chunker.Chunk(doc)
		if err != nil {
			log.Printf("Warning: Failed to chunk document %s: %v", doc.ID, err)
			continue
		}
		allChunks = append(allChunks, chunks...)
	}
	fmt.Printf("   ✅ Loaded %d documents, created %d chunks\n", len(sampleDocs), len(allChunks))

	// Step 4: Generate embeddings and store in vector DB
	fmt.Println("\n🔢 Step 4: Generating embeddings and storing in ChromaDB...")

	var vdbDocs []vectordb.Document
	for _, chunk := range allChunks {
		vdbDocs = append(vdbDocs, vectordb.Document{
			ID:       chunk.ID,
			Content:  chunk.Content,
			Metadata: chunk.Metadata,
			// Embedding will be generated automatically by ChromaDB
		})
	}

	err = db.Add(ctx, vdbDocs)
	if err != nil {
		log.Fatalf("Failed to add documents to vector DB: %v", err)
	}

	count, _ := db.Count(ctx)
	fmt.Printf("   ✅ Stored %d documents in vector database\n\n", count)

	// Step 5: Test retrieval
	fmt.Println("🔍 Step 5: Testing knowledge retrieval...")
	testQuery := "What is machine learning?"
	results, err := db.Query(ctx, testQuery, 2, nil)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Printf("   Query: \"%s\"\n", testQuery)
	fmt.Printf("   Found %d relevant documents:\n", len(results))
	for i, result := range results {
		fmt.Printf("   %d. [Score: %.4f] %s\n", i+1, result.Score,
			truncate(result.Content, 80))
	}
	fmt.Println()

	// Step 6: Create RAG-powered Agent
	fmt.Println("🤖 Step 6: Creating RAG-powered Agent...")

	// Create OpenAI model
	model, err := openaimodel.New("gpt-4o-mini", openaimodel.Config{
		APIKey:      openaiKey,
		Temperature: 0.7,
		MaxTokens:   500,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create RAG toolkit
	ragToolkit := NewRAGToolkit(db)

	// Create agent with RAG capabilities
	ag, err := agent.New(agent.Config{
		Name:     "RAG Assistant",
		Model:    model,
		Toolkits: []toolkit.Toolkit{ragToolkit},
		Instructions: `You are a helpful AI assistant with access to a knowledge base.
When users ask questions:
1. Use the search_knowledge tool to find relevant information
2. Base your answer on the retrieved information
3. Cite the sources when possible
4. If you can't find relevant information, say so

Always be helpful, accurate, and concise.`,
		MaxLoops: 5,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	fmt.Println("   ✅ Agent created with RAG capabilities")

	// Step 7: Interactive Q&A
	fmt.Println("💬 Step 7: Interactive Q&A (RAG in action)")
	fmt.Println("=" + string(make([]byte, 60)) + "=")

	questions := []string{
		"What is artificial intelligence?",
		"Explain the difference between AI and machine learning",
		"What are vector databases used for?",
		"How does RAG improve AI systems?",
	}

	for i, question := range questions {
		fmt.Printf("\n[Question %d] User: %s\n", i+1, question)

		output, err := ag.Run(ctx, question)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		fmt.Printf("Assistant: %s\n", output.Content)
	}

	fmt.Println("\n" + string(make([]byte, 60)) + "=")
	fmt.Println("\n✅ RAG Demo completed successfully!")
	fmt.Println("\nKey Takeaways:")
	fmt.Println("• Documents are chunked and embedded automatically")
	fmt.Println("• Vector database enables semantic search")
	fmt.Println("• Agent uses RAG to provide accurate, grounded answers")
	fmt.Println("• Citations and sources improve trustworthiness")

	// Cleanup
	fmt.Println("\n🧹 Cleaning up...")
	err = db.DeleteCollection(ctx, "rag_demo")
	if err != nil {
		log.Printf("Warning: Failed to delete collection: %v", err)
	} else {
		fmt.Println("   ✅ Deleted demo collection")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
```

## Code Explanation

### 1. Custom RAG Toolkit

```go
type RAGToolkit struct {
	*toolkit.BaseToolkit
	vectorDB vectordb.VectorDB
}
```

The RAG toolkit wraps the vector database and exposes a `search_knowledge` function that the agent can call:

- Accepts a query string and optional limit
- Searches the vector database using semantic similarity
- Returns formatted results with relevance scores

### 2. OpenAI Embeddings

```go
embedFunc, err := openaiembed.New(openaiembed.Config{
	APIKey: openaiKey,
	Model:  "text-embedding-3-small",
})
```

- Uses OpenAI's `text-embedding-3-small` model (1536 dimensions)
- Converts text into dense vector representations
- Enables semantic similarity search

### 3. ChromaDB Vector Database

```go
db, err := chromadb.New(chromadb.Config{
	BaseURL:           "http://localhost:8000",
	CollectionName:    "rag_demo",
	EmbeddingFunction: embedFunc,
})
```

- Connects to local ChromaDB instance
- Creates a collection for storing document embeddings
- Automatically generates embeddings when documents are added

### 4. Document Chunking

```go
chunker := knowledge.NewCharacterChunker(500, 50)
```

- Splits documents into 500-character chunks
- 50-character overlap between chunks
- Preserves context across chunk boundaries
- Improves retrieval accuracy for long documents

### 5. Agent with RAG Capabilities

```go
ag, err := agent.New(agent.Config{
	Name:     "RAG Assistant",
	Model:    model,
	Toolkits: []toolkit.Toolkit{ragToolkit},
	Instructions: `You are a helpful AI assistant with access to a knowledge base.
When users ask questions:
1. Use the search_knowledge tool to find relevant information
2. Base your answer on the retrieved information
3. Cite the sources when possible
4. If you can't find relevant information, say so`,
	MaxLoops: 5,
})
```

The instructions tell the agent:
- When to use the knowledge retrieval tool
- How to incorporate retrieved information
- To cite sources for transparency
- To be honest when information isn't available

## Running the Example

```bash
# Make sure ChromaDB is running
docker run -p 8000:8000 chromadb/chroma

# Run the demo
go run main.go
```

## Expected Output

```
🚀 RAG (Retrieval-Augmented Generation) Demo
This example demonstrates:
1. Loading documents from files
2. Chunking text into smaller pieces
3. Generating embeddings with OpenAI
4. Storing in ChromaDB vector database
5. Using RAG with an Agent to answer questions

📊 Step 1: Creating OpenAI embedding function...
   ✅ Created embedding function (model: text-embedding-3-small, dimensions: 1536)

💾 Step 2: Connecting to ChromaDB...
   ✅ Connected to ChromaDB and created collection

📚 Step 3: Loading and processing documents...
   ✅ Loaded 5 documents, created 5 chunks

🔢 Step 4: Generating embeddings and storing in ChromaDB...
   ✅ Stored 5 documents in vector database

🔍 Step 5: Testing knowledge retrieval...
   Query: "What is machine learning?"
   Found 2 relevant documents:
   1. [Score: 0.8234] Machine Learning (ML) is a subset of AI that focuses on creating syste...
   2. [Score: 0.7156] Artificial Intelligence (AI) is the simulation of human intelligence b...

🤖 Step 6: Creating RAG-powered Agent...
   ✅ Agent created with RAG capabilities

💬 Step 7: Interactive Q&A (RAG in action)
============================================================

[Question 1] User: What is artificial intelligence?
Assistant: Based on the knowledge base, Artificial Intelligence (AI) is the simulation of human intelligence by machines. AI systems can perform tasks that typically require human intelligence, such as visual perception, speech recognition, decision-making, and language translation. Modern AI is based on machine learning algorithms that can learn from data.

[Question 2] User: Explain the difference between AI and machine learning
Assistant: AI (Artificial Intelligence) is the broader concept of machines simulating human intelligence to perform various tasks. Machine Learning (ML) is a subset of AI that specifically focuses on creating systems that learn from data rather than being explicitly programmed. So while all ML is AI, not all AI is ML. ML improves performance through experience, using algorithms like neural networks and decision trees.

[Question 3] User: What are vector databases used for?
Assistant: According to the knowledge base, vector databases are specialized databases designed to store and query high-dimensional vector embeddings. They enable semantic search by finding similar vectors using distance metrics like cosine similarity or Euclidean distance. Vector databases are essential for RAG (Retrieval-Augmented Generation) systems.

[Question 4] User: How does RAG improve AI systems?
Assistant: RAG (Retrieval-Augmented Generation) improves AI systems by combining information retrieval with text generation. The process first retrieves relevant documents from a knowledge base, then uses a language model to generate responses based on the retrieved context. This approach improves accuracy and reduces hallucinations in AI systems by grounding responses in actual knowledge.

============================================================

✅ RAG Demo completed successfully!

Key Takeaways:
• Documents are chunked and embedded automatically
• Vector database enables semantic search
• Agent uses RAG to provide accurate, grounded answers
• Citations and sources improve trustworthiness

🧹 Cleaning up...
   ✅ Deleted demo collection
```

## Key Concepts

### RAG Pipeline

1. **Ingestion**: Documents are loaded and chunked
2. **Embedding**: Chunks are converted to vector embeddings
3. **Storage**: Embeddings stored in vector database
4. **Retrieval**: User query is embedded and similar documents found
5. **Generation**: LLM generates answer based on retrieved context

### Semantic Search

Unlike keyword search, semantic search finds documents based on meaning:

- Query: "What is ML?"
- Matches: Documents about "Machine Learning", "neural networks", "training models"
- Better than exact keyword matching

### Chunking Strategy

Document chunking parameters affect retrieval quality:

- **Chunk size (500)**: Balance between context and precision
  - Too small: Loses context
  - Too large: Retrieves irrelevant content

- **Overlap (50)**: Prevents splitting important information
  - Ensures continuity across chunks
  - Critical for sentences spanning boundaries

### Custom Toolkits

The RAG toolkit demonstrates how to build custom tools:

```go
t.RegisterFunction(&toolkit.Function{
	Name:        "search_knowledge",
	Description: "Search the knowledge base...",
	Parameters: map[string]toolkit.Parameter{
		"query": {
			Type:        "string",
			Description: "The search query",
			Required:    true,
		},
	},
	Handler: t.searchKnowledge,
})
```

Key elements:
- Clear name and description for LLM understanding
- Well-defined parameters with types
- Handler function that implements the logic

## Advanced Features

### Metadata Filtering

You can filter results by metadata:

```go
results, err := db.Query(ctx, query, limit, map[string]interface{}{
	"topic": "Machine Learning",
	"date": map[string]interface{}{
		"$gte": "2025-01-01",
	},
})
```

### Hybrid Search

Combine semantic and keyword search:

```go
// Future feature - not yet implemented
results, err := db.HybridQuery(ctx, query, limit, HybridConfig{
	SemanticWeight: 0.7,
	KeywordWeight:  0.3,
})
```

### Reranking

Improve results by reranking with a more powerful model:

```go
// Future feature - not yet implemented
reranked := reranker.Rerank(results, query, limit)
```

## Best Practices

1. **Chunk Size**: Start with 500-1000 characters, adjust based on your content
2. **Overlap**: Use 10-20% of chunk size for overlap
3. **Embeddings**: Use domain-specific embeddings when available
4. **Metadata**: Include metadata for filtering and citation
5. **Error Handling**: Always handle retrieval failures gracefully
6. **Caching**: Consider caching frequently accessed embeddings
7. **Monitoring**: Track retrieval quality and adjust parameters

## Troubleshooting

**Error: "Failed to connect to ChromaDB"**
- Ensure ChromaDB is running: `docker ps | grep chroma`
- Check the port (default: 8000): `curl http://localhost:8000/api/v1/heartbeat`

**Error: "OPENAI_API_KEY environment variable is required"**
- Set your API key: `export OPENAI_API_KEY=sk-...`

**Low retrieval quality**
- Adjust chunk size and overlap
- Try different embedding models
- Add more relevant documents
- Use metadata filtering

**High latency**
- Use smaller embedding models
- Reduce the number of results (limit parameter)
- Consider caching embeddings
- Use local embedding models (future feature)

## Next Steps

- Explore [Simple Agent](./simple-agent.md) for basic agent usage
- Learn about [Team Collaboration](./team-demo.md) with multiple agents
- Try [Workflow Engine](./workflow-demo.md) for complex RAG pipelines
- Build production RAG with [AgentOS API](../api/agentos.md)

## Additional Resources

- [OpenAI Embeddings Guide](https://platform.openai.com/docs/guides/embeddings)
- [ChromaDB Documentation](https://docs.trychroma.com/)
- [RAG Best Practices](https://docs.agno.com/advanced/rag)
- [Vector Database Comparison](https://docs.agno.com/storage/vector-dbs)
