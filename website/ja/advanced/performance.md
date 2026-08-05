# パフォーマンス

このページは、固定のローカル dummy model を使った framework overhead の測定を公開します。
LLM の品質、ネットワーク遅延、本番スループットの保証ではありません。原データは
`benchmarks/framework_comparison/results/latest.json` にあります。

## クロスフレームワーク測定

記録: 2026-08-04 04:19:54 UTC

- Target: `windows/amd64`
- CPU: `12th Gen Intel(R) Core(TM) i5-12400F`
- Go: `go1.26.4`
- Python: `3.11.15`
- Packages: `agno==2.8.6`, `langgraph==1.2.10`
- Python 20 回、Go 10 回。外部 Provider、ネットワーク、DB、Token は使用しません。

### HNO と Agno: Agent 作成

| Operation | Framework | Mean ns/op | Median ns/op | Range ns/op |
| --- | --- | ---: | ---: | ---: |
| Minimal Agent | HNO | 255.8 | 252.4 | 249.2-266.8 |
| Minimal Agent | Agno | 7,105.5 | 6,869.1 | 5,308.8-10,042.1 |
| Agent + Tool 1個 | HNO | 298.4 | 296.8 | 265.9-355.1 |
| Agent + Tool 1個 | Agno | 6,603.7 | 6,394.4 | 5,812.1-8,591.5 |

この構成だけの Agno/HNO 平均比は 27.8 倍と 22.1 倍です。これはこのマシン、
Version、設定、操作に限定され、本番やエンドツーエンドの速度比ではありません。

### HNO と Agno: fresh local dummy run

新しい Agent を作成し、固定応答で `ping` を一回実行します。

| Framework | Mean ns/op | Median ns/op | Range ns/op |
| --- | ---: | ---: | ---: |
| HNO | 6,431.0 | 6,528.0 | 5,360.0-7,090.0 |
| Agno | 187,208.6 | 180,537.0 | 162,652.0-241,754.0 |

この操作の平均比は 29.1 倍です。実 LLM、ネットワーク、DB、Token 生成は含みません。

### LangGraph: 別の指標

LangGraph は Agent 作成ではなく graph compile を行うため別に報告します。最小
`StateGraph` は平均 `356,598.2 ns/op`、中央値 `352,408.5 ns/op`、範囲
`332,839.0-394,720.0 ns/op` でした。これは完全な Agent システムの速度順位ではありません。

## 再現

```bash
uv run --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  python benchmarks/framework_comparison/compare.py --repeat 20 --number 1000
```

プロトコル、raw JSON、source hash、制限は
`benchmarks/framework_comparison/README.md` を参照してください。Go の `B/op` と
Python の `timeit` は別の測定体系であり、メモリ比較には使いません。

## 未測定

実 LLM 遅延、Token throughput、resident memory、Team/Workflow、実サービス Tool、
固定並行度の RPS、本番容量とコストは未測定です。これらには同じ Provider、model、
prompt、Tool、output、timeout、concurrency、hardware、Version が必要です。

## なぜ Go、なぜ HNO

Go はコンパイル済み配布物、組み込み並行処理、静的型、HTTP/JSON 標準ライブラリ、
テスト/profile/race ツールが理由の実装選択です。HNO は現在のプロジェクト名で、
正式な略語の展開はリポジトリに定義されていません。Go module path は
`github.com/rexleimo/agno-go` のままです。
