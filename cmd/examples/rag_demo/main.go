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
