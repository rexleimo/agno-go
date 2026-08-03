# RAG (Retrieval-Augmented Generation) デモ

## 概要

この例では、Agno-Go を使用して RAG システムを構築する方法を示します。RAG は、知識ベースからの情報検索と LLM テキスト生成を組み合わせて、正確で根拠のある回答を提供します。このシステムは、ベクトルストレージに ChromaDB、OpenAI 埋め込み、セマンティック検索機能を有効にするカスタム RAG ツールキットを使用します。

## 学べること

- OpenAI 埋め込みの作成と使用方法
- ベクトルデータベースとして ChromaDB をセットアップする方法
- 最適な検索のためにドキュメントをチャンク化する方法
- エージェント用のカスタム RAG ツールキットを構築する方法
- 知識検索機能を持つエージェントを作成する方法
- RAG 実装のベストプラクティス

## 前提条件

- Go 1.21 以降
- OpenAI API キー
- ChromaDB がローカルで実行中 (Docker 経由)

## セットアップ

1. OpenAI API キーを設定します:
```bash
export OPENAI_API_KEY=sk-your-api-key-here
```

2. Docker を使用して ChromaDB を起動します:
```bash
docker pull chromadb/chroma
docker run -p 8000:8000 chromadb/chroma
```

3. サンプルディレクトリに移動します:
```bash
cd cmd/examples/rag_demo
```

## 完全なコード

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

## コードの説明

### 1. カスタム RAG ツールキット

```go
type RAGToolkit struct {
	*toolkit.BaseToolkit
	vectorDB vectordb.VectorDB
}
```

RAG ツールキットはベクトルデータベースをラップし、エージェントが呼び出せる `search_knowledge` 関数を公開します:

- クエリ文字列とオプションの制限を受け入れる
- セマンティック類似性を使用してベクトルデータベースを検索
- 関連性スコアを含む整形された結果を返す

### 2. OpenAI 埋め込み

```go
embedFunc, err := openaiembed.New(openaiembed.Config{
	APIKey: openaiKey,
	Model:  "text-embedding-3-small",
})
```

- OpenAI の `text-embedding-3-small` モデルを使用 (1536 次元)
- テキストを密なベクトル表現に変換
- セマンティック類似性検索を可能にする

### 3. ChromaDB ベクトルデータベース

```go
db, err := chromadb.New(chromadb.Config{
	BaseURL:           "http://localhost:8000",
	CollectionName:    "rag_demo",
	EmbeddingFunction: embedFunc,
})
```

- ローカル ChromaDB インスタンスに接続
- ドキュメント埋め込みを保存するためのコレクションを作成
- ドキュメントが追加されると自動的に埋め込みを生成

### 4. ドキュメントチャンキング

```go
chunker := knowledge.NewCharacterChunker(500, 50)
```

- ドキュメントを 500 文字のチャンクに分割
- チャンク間の 50 文字の重複
- チャンク境界を越えてコンテキストを保持
- 長いドキュメントの検索精度を向上

### 5. RAG 機能を持つエージェント

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

指示はエージェントに伝えます:
- 知識検索ツールをいつ使用するか
- 検索された情報をどのように組み込むか
- 透明性のためにソースを引用すること
- 情報が利用できない場合に正直であること

## サンプルの実行

```bash
# ChromaDB が実行中であることを確認
docker run -p 8000:8000 chromadb/chroma

# デモを実行
go run main.go
```

## 期待される出力

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

## 主要な概念

### RAG パイプライン

1. **取り込み**: ドキュメントが読み込まれ、チャンク化される
2. **埋め込み**: チャンクがベクトル埋め込みに変換される
3. **ストレージ**: 埋め込みがベクトルデータベースに保存される
4. **検索**: ユーザークエリが埋め込まれ、類似ドキュメントが見つかる
5. **生成**: LLM が検索されたコンテキストに基づいて回答を生成

### セマンティック検索

キーワード検索とは異なり、セマンティック検索は意味に基づいてドキュメントを見つけます:

- クエリ: "What is ML?"
- マッチ: "Machine Learning"、"neural networks"、"training models" に関するドキュメント
- 完全一致キーワードマッチングより優れている

### チャンキング戦略

ドキュメントのチャンキングパラメータは検索品質に影響します:

- **チャンクサイズ (500)**: コンテキストと精度のバランス
  - 小さすぎる: コンテキストを失う
  - 大きすぎる: 無関係なコンテンツを検索

- **重複 (50)**: 重要な情報の分割を防ぐ
  - チャンク間の連続性を確保
  - 境界をまたぐ文にとって重要

### カスタムツールキット

RAG ツールキットは、カスタムツールの構築方法を示します:

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

主要な要素:
- LLM が理解するための明確な名前と説明
- 型を持つ明確に定義されたパラメータ
- ロジックを実装するハンドラー関数

## 高度な機能

### メタデータフィルタリング

メタデータで結果をフィルタリングできます:

```go
results, err := db.Query(ctx, query, limit, map[string]interface{}{
	"topic": "Machine Learning",
	"date": map[string]interface{}{
		"$gte": "2025-01-01",
	},
})
```

### ハイブリッド検索

セマンティック検索とキーワード検索を組み合わせる:

```go
// 将来の機能 - まだ実装されていません
results, err := db.HybridQuery(ctx, query, limit, HybridConfig{
	SemanticWeight: 0.7,
	KeywordWeight:  0.3,
})
```

### リランキング

より強力なモデルでリランキングして結果を改善:

```go
// 将来の機能 - まだ実装されていません
reranked := reranker.Rerank(results, query, limit)
```

## ベストプラクティス

1. **チャンクサイズ**: 500-1000 文字から始め、コンテンツに基づいて調整
2. **重複**: チャンクサイズの 10-20% を重複に使用
3. **埋め込み**: 利用可能な場合はドメイン固有の埋め込みを使用
4. **メタデータ**: フィルタリングと引用のためにメタデータを含める
5. **エラー処理**: 検索失敗を常に適切に処理
6. **キャッシング**: 頻繁にアクセスされる埋め込みのキャッシングを検討
7. **監視**: 検索品質を追跡し、パラメータを調整

## トラブルシューティング

**エラー: "Failed to connect to ChromaDB"**
- ChromaDB が実行中であることを確認: `docker ps | grep chroma`
- ポートを確認 (デフォルト: 8000): `curl http://localhost:8000/api/v1/heartbeat`

**エラー: "OPENAI_API_KEY environment variable is required"**
- API キーを設定: `export OPENAI_API_KEY=sk-...`

**検索品質が低い**
- チャンクサイズと重複を調整
- 異なる埋め込みモデルを試す
- より関連性の高いドキュメントを追加
- メタデータフィルタリングを使用

**高レイテンシー**
- より小さい埋め込みモデルを使用
- 結果の数を減らす (limit パラメータ)
- 埋め込みのキャッシングを検討
- ローカル埋め込みモデルを使用 (将来の機能)

## 次のステップ

- 基本的なエージェント使用のために [Simple Agent](./simple-agent.md) を探索
- 複数のエージェントを使った [Team Collaboration](./team-demo.md) について学ぶ
- 複雑な RAG パイプライン用に [Workflow Engine](./workflow-demo.md) を試す
- [AgentOS API](../api/agentos.md) で本番 RAG を構築

## 追加リソース

- [OpenAI Embeddings ガイド](https://platform.openai.com/docs/guides/embeddings)
- [ChromaDB ドキュメント](https://docs.trychroma.com/)
- [RAG ベストプラクティス](https://docs.agno.com/advanced/rag)
- [ベクトルデータベース比較](https://docs.agno.com/storage/vector-dbs)
