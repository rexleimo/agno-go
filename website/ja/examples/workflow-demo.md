# Workflow Engine の例

## 概要

この例では、HNO の強力な Workflow エンジンと 5 つの基本構成要素を示します: Step、Condition、Loop、Parallel、Router。Workflow は決定論的で制御された実行フローを提供します - 正確な制御と可観測性を必要とする複雑な多段階プロセスに最適です。

## 学べること

- 5 つのプリミティブタイプで Workflow を構築する方法
- 順次、条件付き、並列実行パターン
- カスタム終了条件でループを作成する方法
- 実行を動的にルーティングする方法
- プリミティブを組み合わせて複雑な Workflow を作成する方法

## 前提条件

- Go 1.21 以降
- OpenAI API キー

## セットアップ

```bash
export OPENAI_API_KEY=sk-your-api-key-here
cd cmd/examples/workflow_demo
```

## Workflow プリミティブ

### 1. Step - 基本実行単位

エージェントまたはカスタム関数を実行します。

```go
step, _ := workflow.NewStep(workflow.StepConfig{
	ID:    "research",
	Agent: researchAgent,
})
```

### 2. Condition - 分岐ロジック

条件に基づいて異なるパスにルーティングします。

```go
condition, _ := workflow.NewCondition(workflow.ConditionConfig{
	ID: "sentiment-check",
	Condition: func(ctx *workflow.ExecutionContext) bool {
		return strings.Contains(ctx.Output, "positive")
	},
	TrueNode:  positiveStep,
	FalseNode: negativeStep,
})
```

### 3. Loop - 反復実行

条件が満たされるまでステップを繰り返します。

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
	ID:   "refinement",
	Body: refineStep,
	Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
		return iteration < 3  // 3 回実行
	},
})
```

### 4. Parallel - 並行実行

複数のステップを同時に実行します。

```go
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
	ID:    "analysis",
	Nodes: []workflow.Node{techStep, bizStep, ethicsStep},
})
```

### 5. Router - 動的ルーティング

ランタイムロジックに基づいて異なるステップにルーティングします。

```go
router, _ := workflow.NewRouter(workflow.RouterConfig{
	ID: "task-router",
	Router: func(ctx *workflow.ExecutionContext) string {
		if strings.Contains(ctx.Output, "calculation") {
			return "calc"
		}
		return "general"
	},
	Routes: map[string]workflow.Node{
		"calc":    calcStep,
		"general": generalStep,
	},
})
```

## 完全な例

### デモ 1: Sequential Workflow

基本パイプライン: 調査 → 分析 → 執筆

```go
func runSequentialWorkflow(ctx context.Context, apiKey string) {
	// Create agents for pipeline
	researcher := createAgent("researcher", apiKey,
		"You are a researcher. Gather facts about the topic.")
	analyzer := createAgent("analyzer", apiKey,
		"You are an analyst. Analyze the facts and draw conclusions.")
	writer := createAgent("writer", apiKey,
		"You are a writer. Write a concise summary.")

	// Create steps
	step1, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "research",
		Agent: researcher,
	})

	step2, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "analyze",
		Agent: analyzer,
	})

	step3, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "write",
		Agent: writer,
	})

	// Create workflow
	wf, _ := workflow.New(workflow.Config{
		Name:  "Content Pipeline",
		Steps: []workflow.Node{step1, step2, step3},
	})

	result, _ := wf.Run(ctx, "The impact of renewable energy on climate change")
	fmt.Printf("Final Output: %s\n", result.Output)
}
```

**フロー:**
```
入力 → Researcher → Analyzer → Writer → 出力
```

### デモ 2: Conditional Workflow

分岐ロジックを使用した感情分析。

```go
func runConditionalWorkflow(ctx context.Context, apiKey string) {
	classifier := createAgent("classifier", apiKey,
		"Classify the sentiment as positive or negative. Respond with just 'positive' or 'negative'.")
	positiveHandler := createAgent("positive", apiKey,
		"You handle positive feedback. Thank the user warmly.")
	negativeHandler := createAgent("negative", apiKey,
		"You handle negative feedback. Apologize and offer help.")

	// Classification step
	classifyStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "classify",
		Agent: classifier,
	})

	// Positive branch
	positiveStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "positive-response",
		Agent: positiveHandler,
	})

	// Negative branch
	negativeStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "negative-response",
		Agent: negativeHandler,
	})

	// Conditional node
	condition, _ := workflow.NewCondition(workflow.ConditionConfig{
		ID: "sentiment-branch",
		Condition: func(ctx *workflow.ExecutionContext) bool {
			return strings.Contains(strings.ToLower(ctx.Output), "positive")
		},
		TrueNode:  positiveStep,
		FalseNode: negativeStep,
	})

	// Create workflow
	wf, _ := workflow.New(workflow.Config{
		Name:  "Sentiment Handler",
		Steps: []workflow.Node{classifyStep, condition},
	})

	result, _ := wf.Run(ctx, "Your product is amazing! I love it!")
	fmt.Printf("Response: %s\n", result.Output)
}
```

**フロー:**
```
入力 → 分類 → [Positive?] → Positive ハンドラー
               ↓ [Negative]
               → Negative ハンドラー
```

### デモ 3: Loop Workflow

反復的なテキスト改善。

```go
func runLoopWorkflow(ctx context.Context, apiKey string) {
	refiner := createAgent("refiner", apiKey,
		"Refine and improve the given text. Make it more concise.")

	// Loop body
	refineStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "refine",
		Agent: refiner,
	})

	// Loop 3 times for iterative refinement
	loop, _ := workflow.NewLoop(workflow.LoopConfig{
		ID:   "refinement-loop",
		Body: refineStep,
		Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
			return iteration < 3
		},
	})

	wf, _ := workflow.New(workflow.Config{
		Name:  "Iterative Refinement",
		Steps: []workflow.Node{loop},
	})

	result, _ := wf.Run(ctx, "AI is a technology that enables machines...")

	iterations, _ := result.Get("loop_refinement-loop_iterations")
	fmt.Printf("Refined after %v iterations: %s\n", iterations, result.Output)
}
```

**フロー:**
```
入力 → [改善] → [反復 < 3?] → [Yes] → 再度改善
        ↑                     ↓ [No]
        └───────────────────── 出力
```

### デモ 4: Parallel Workflow

同時に実行される多視点分析。

```go
func runParallelWorkflow(ctx context.Context, apiKey string) {
	techAgent := createAgent("tech", apiKey, "Analyze technical aspects in 1-2 sentences.")
	bizAgent := createAgent("biz", apiKey, "Analyze business aspects in 1-2 sentences.")
	ethicsAgent := createAgent("ethics", apiKey, "Analyze ethical aspects in 1-2 sentences.")

	techStep, _ := workflow.NewStep(workflow.StepConfig{ID: "tech-analysis", Agent: techAgent})
	bizStep, _ := workflow.NewStep(workflow.StepConfig{ID: "biz-analysis", Agent: bizAgent})
	ethicsStep, _ := workflow.NewStep(workflow.StepConfig{ID: "ethics-analysis", Agent: ethicsAgent})

	parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
		ID:    "multi-perspective",
		Nodes: []workflow.Node{techStep, bizStep, ethicsStep},
	})

	wf, _ := workflow.New(workflow.Config{
		Name:  "Parallel Analysis",
		Steps: []workflow.Node{parallel},
	})

	result, _ := wf.Run(ctx, "The use of facial recognition technology in public spaces")

	// Access individual results
	tech, _ := result.Get("parallel_multi-perspective_branch_0_output")
	biz, _ := result.Get("parallel_multi-perspective_branch_1_output")
	ethics, _ := result.Get("parallel_multi-perspective_branch_2_output")

	fmt.Printf("Tech: %v\nBusiness: %v\nEthics: %v\n", tech, biz, ethics)
}
```

**フロー:**
```
入力 → [Tech Agent]    → 結合
     → [Biz Agent]     → 結果
     → [Ethics Agent]  → 出力
     (すべて同時実行)
```

### デモ 5: Router を使用した複雑な Workflow

タスクタイプに基づく動的ルーティング。

```go
func runComplexWorkflow(ctx context.Context, apiKey string) {
	// Router determines task type
	router := createAgent("router", apiKey,
		"Determine if this is a 'calculation' or 'general' task. Respond with just the word.")

	// Calculation route
	calcAgent := createAgent("calculator", apiKey,
		"You perform calculations.", calculator.New())

	// General route
	generalAgent := createAgent("general", apiKey,
		"You handle general questions.")

	// Create steps
	routerStep, _ := workflow.NewStep(workflow.StepConfig{ID: "router", Agent: router})
	calcStep, _ := workflow.NewStep(workflow.StepConfig{ID: "calc-task", Agent: calcAgent})
	generalStep, _ := workflow.NewStep(workflow.StepConfig{ID: "general-task", Agent: generalAgent})

	// Router node
	routerNode, _ := workflow.NewRouter(workflow.RouterConfig{
		ID: "task-router",
		Router: func(ctx *workflow.ExecutionContext) string {
			if strings.Contains(strings.ToLower(ctx.Output), "calculation") {
				return "calc"
			}
			return "general"
		},
		Routes: map[string]workflow.Node{
			"calc":    calcStep,
			"general": generalStep,
		},
	})

	wf, _ := workflow.New(workflow.Config{
		Name:  "Smart Router",
		Steps: []workflow.Node{routerStep, routerNode},
	})

	// Test with calculation
	result1, _ := wf.Run(ctx, "What is 25 * 4 + 100?")
	fmt.Printf("Calculation result: %s\n", result1.Output)

	// Test with general question
	result2, _ := wf.Run(ctx, "What is the capital of France?")
	fmt.Printf("General result: %s\n", result2.Output)
}
```

**フロー:**
```
入力 → Router → [Type?] → Calc Agent    (計算の場合)
                        → General Agent  (一般の場合)
```

## サンプルの実行

```bash
go run main.go
```

## 期待される出力

```
=== Demo 1: Sequential Workflow ===
Final Output: Renewable energy significantly reduces greenhouse gas emissions, helping combat climate change by replacing fossil fuels with clean power sources like solar and wind.

=== Demo 2: Conditional Workflow ===
Response: Thank you so much for your wonderful feedback! We're thrilled that you love our product!

=== Demo 3: Loop Workflow ===
Refined after 3 iterations: AI enables machines to learn, reason, and understand language.

=== Demo 4: Parallel Workflow ===
Tech: Uses computer vision and deep learning for pattern recognition in real-time.
Business: Creates new security markets but raises privacy-related costs.
Ethics: Raises serious concerns about surveillance, consent, and civil liberties.

=== Demo 5: Complex Workflow with Router ===
Calculation result: 25 * 4 + 100 equals 200.
General result: The capital of France is Paris.
```

## 実行コンテキスト

Workflow の状態と結果にアクセス:

```go
result, _ := wf.Run(ctx, input)

// メイン出力
fmt.Println(result.Output)

// コンテキストから特定の値を取得
value, exists := result.Get("step_id_output")

// 実行成功を確認
if result.Error != nil {
	log.Printf("Workflow error: %v", result.Error)
}
```

### コンテキストキー

Workflow は予測可能なキーでデータを保存します:

- **Step 出力**: `step_{step-id}_output`
- **Loop 反復**: `loop_{loop-id}_iterations`
- **Parallel ブランチ**: `parallel_{parallel-id}_branch_{index}_output`
- **Condition 結果**: `condition_{condition-id}_result`

## Workflow vs Team

| 特徴 | Workflow | Team |
|---------|----------|------|
| **制御** | 高 - 明示的なフロー | 低 - エージェントが自己組織化 |
| **柔軟性** | 低 - 事前定義されたパス | 高 - 動的な協力 |
| **可観測性** | 高 - すべてのステップを追跡 | 中程度 - エージェント出力 |
| **ユースケース** | 決定論的プロセス | 創造的な協力 |
| **デバッグ** | 簡単 - ステップバイステップ | 難しい - 創発的な動作 |

**Workflow を選択する場合:**
- 実行の正確な制御が必要
- デバッグと可観測性が重要
- プロセスに明確に定義されたステップがある
- コンプライアンス/監査要件が存在

**Team を選択する場合:**
- 解決パスが不明確
- エージェントに創造的に協力してほしい
- タスクが複数の視点から利益を得る
- 柔軟性が制御よりも重要

## デザインパターン

### 1. Pipeline パターン
```go
Steps: []workflow.Node{step1, step2, step3}
// 線形フロー: A → B → C
```

### 2. Branch パターン
```go
Condition → TrueNode
         → FalseNode
// If-else ロジック
```

### 3. Retry パターン
```go
Loop with condition checking success
// 成功または最大試行回数まで再試行
```

### 4. Fan-Out パターン
```go
Parallel → Multiple agents simultaneously
// 作業を分散、結果を収集
```

### 5. State Machine パターン
```go
Router → Different states based on output
// 状態に基づく動的ルーティング
```

## ベストプラクティス

### 1. 可観測性のために設計

```go
// ✅ 良い: 明確で説明的な ID
workflow.StepConfig{
	ID:    "validate-input",
	Agent: validator,
}

// ❌ 悪い: 曖昧な ID
workflow.StepConfig{
	ID:    "step1",
	Agent: validator,
}
```

### 2. エラーを適切に処理

```go
result, err := wf.Run(ctx, input)
if err != nil {
	log.Printf("Workflow failed: %v", err)
	// フォールバックロジック
}

if result.Error != nil {
	log.Printf("Step error: %v", result.Error)
}
```

### 3. Step を集中させる

```go
// ✅ 良い: 単一責任
extractStep  := "Extract data from input"
validateStep := "Validate extracted data"
saveStep     := "Save validated data"

// ❌ 悪い: 1 つのステップに多すぎる
megaStep := "Extract, validate, transform, and save data"
```

### 4. 並列実行を最適化

```go
// 独立したタスクには parallel を使用
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
	Nodes: []workflow.Node{
		fetchUserData,
		fetchOrderData,
		fetchInventoryData,
	},
})
```

### 5. 共有状態にコンテキストを使用

```go
// Step は以前の出力にアクセス可能
analyzer := createAgent("analyzer", apiKey,
	"Analyze the research data provided in the previous step.")
```

## 高度な機能

### Step でのカスタム関数

```go
step, _ := workflow.NewStep(workflow.StepConfig{
	ID: "custom-processing",
	Function: func(ctx context.Context, input string) (string, error) {
		// エージェントなしのカスタムロジック
		return processData(input), nil
	},
})
```

### 動的ループ条件

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
	Body: improveStep,
	Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
		// 品質閾値に達したら終了
		quality := assessQuality(ctx.Output)
		return quality < 0.9 && iteration < 10
	},
})
```

### 条件付きルーティング

```go
router, _ := workflow.NewRouter(workflow.RouterConfig{
	Router: func(ctx *workflow.ExecutionContext) string {
		// コンテンツに基づいてルート
		if containsCode(ctx.Output) {
			return "code-review"
		}
		if containsData(ctx.Output) {
			return "data-analysis"
		}
		return "general"
	},
	Routes: map[string]workflow.Node{...},
})
```

## パフォーマンスのヒント

1. **Step を最小化**: 各ステップがレイテンシーを追加
2. **Parallel を使用**: 独立したタスクは同時実行すべき
3. **Loop を制限**: 合理的な最大反復を設定
4. **結果をキャッシュ**: 高価な計算をコンテキストに保存
5. **高速モデルを選択**: 速度が重要なステップには gpt-4o-mini を使用

## 次のステップ

- 異なるユースケースのために [Team Collaboration](./team-demo.md) と比較
- 検索ステップを使用した [RAG Workflows](./rag-demo.md) を構築
- 異なる [Model Providers](./claude-agent.md) と Workflow を組み合わせる
- Workflow ステップ用の [Custom Tools](./simple-agent.md) を作成

## トラブルシューティング

**Workflow がループでスタック:**
- ループ条件ロジックを確認
- 最大反復制限を追加
- デバッグのために反復カウントをログ

**Parallel ステップが同時実行されない:**
- Parallel ノードにあることを確認、順次ステップではない
- 共有リソース/ロックを確認

**コンテキスト値にアクセスできない:**
- 正しいキー形式を使用: `{type}_{id}_{field}`
- ステップ ID が正確に一致するか確認
- ステップが正常に実行されたか確認

**Router が常に同じパスを取る:**
- Router 関数の出力をログ
- 条件ロジックを確認
- ルートが適切に定義されているか確認
