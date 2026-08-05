---
layout: home

hero:
  name: "HNO"
  text: "Go ネイティブ・マルチエージェントフレームワーク"
  tagline: "未検証の推定ではなく、明示的で再現可能な証拠で説明する Go 実装。"
  actions:
    - theme: brand
      text: はじめる
      link: /ja/guide/quick-start
    - theme: alt
      text: GitHub で見る
      link: https://github.com/rexleimo/agno-go

features:
  - title: 推測ではなく計測
    details: Agent 作成用の Go benchmark を同梱し、Performance ページにコマンド、環境、範囲、制限を記録しています。

  - title: Provider アダプター
    details: Provider 実装は Model インターフェースの背後にあります。現在のソースにはトップレベルの Provider パッケージが 17 個ありますが、互換性や遅延の保証ではありません。

  - title: 共有オーケストレーション
    details: Agent、Team、Workflow は Go ソースの実行コンポーネントと Tool dispatch 抽象を共有します。

  - title: 可観測性の統合
    details: OpenTelemetry と構造化ランタイム計測を利用できます。計測の有効化によるコストは構成に依存するため、対象サービスで測定します。

  - title: Skills・MCP・メモリ
    details: Agent Skills、MCP ブリッジ、プラグ可能なメモリとセッションストレージをオプション機能として提供します。

  - title: 正確なプロトコル説明
    details: 自動テストはアダプターと request/response マッピングを対象にします。実 Provider の検証には認証情報が必要で、テスト応答を本番の証拠とは表現しません。

---

## クロスフレームワーク benchmark スナップショット

これは制御されたローカル benchmark であり、本番や LLM の benchmark ではありません。

| 操作 | HNO 平均 | Agno 平均 | HNO 中央値 | Agno 中央値 |
| --- | ---: | ---: | ---: | ---: |
| Agent 作成 | 255.8 ns/op | 7,105.5 ns/op | 252.4 | 6,869.1 |
| Agent 作成（Tool 1個） | 298.4 ns/op | 6,603.7 ns/op | 296.8 | 6,394.4 |
| 新規 Agent + local dummy run | 6,431.0 ns/op | 187,208.6 ns/op | 6,528.0 | 180,537.0 |

環境は Windows amd64、12th Gen Intel Core i5-12400F、Go 1.26.4、Python 3.11.15、
`agno==2.8.6`、`langgraph==1.2.10`。Python 20 回、Go 10 回を実行し、すべて
固定のローカル dummy 応答を使います。構成だけの Agno/HNO 平均比は 27.8 倍と
22.1 倍、fresh run は 29.1 倍でした。これはこの workload 限定であり、本番や
エンドツーエンドの速度比ではありません。

LangGraph は Agent オブジェクトではなくグラフのコンパイルを行うため別に報告します。
最小 `StateGraph` の平均は `356,598.2 ns/op`、中央値は `352,408.5 ns/op`、
範囲は `332,839.0-394,720.0 ns/op` です。

元のサンプル、hash、バージョン、コマンドは
[`benchmarks/framework_comparison/`](https://github.com/rexleimo/agno-go/tree/main/benchmarks/framework_comparison) にあります。

```bash
uv run --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  python benchmarks/framework_comparison/compare.py --repeat 20 --number 1000
```

## なぜ Go、なぜ HNO

**なぜ Go か:** コンパイル済み配布物、組み込みの並行処理、静的型、HTTP/JSON
標準ライブラリ、標準のテストとプロファイルツールが実装上の理由です。特定の
負荷での有利さは、その負荷で計測する必要があります。

**なぜ HNO か:** HNO は現在のプロジェクト名です。正式な略語の展開はリポジトリに
定義されていないため、ここで創作しません。Go module path は
`github.com/rexleimo/agno-go` のままで、HNO は標準モデルやプロトコルではなく
プロジェクトのブランド名です。

**証拠の方針:** 平均、中央値、範囲、環境、バージョン、コマンドを記録します。
Go の割り当てバイトを Python のメモリ値として扱いません。実 LLM や本番容量の
比較には同じ Provider と workload が必要です。
