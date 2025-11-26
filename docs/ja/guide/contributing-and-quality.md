## 貢献と品質ゲート

このページでは、Agno-Go プロジェクトへの貢献方法と、変更をマージする前に通過すべき品質ゲートについて説明します。

### 1. まず読むべきもの

- **リポジトリルートの `AGENTS.md`**  
  - プロジェクト構成（`go/`, `docs/`, `specs/`, `scripts/` など）。  
  - ランタイムの制約（純粋な Go 実装であり、ランタイムでの Python/cgo ブリッジは禁止）。  
  - 仕様とフィクスチャがどのように挙動を制約しているか。  
- **実装前に該当する spec を確認**  
  - AgentOS 契約とフィクスチャ: `specs/001-go-agno-rewrite/`。  
  - VitePress ドキュメント計画: `specs/001-vitepress-docs/`。  

大きめの変更では、ランタイムコードを触る前に spec / plan / tasks を更新することを推奨します。

### 2. 主要な `make` ターゲット

すべての品質チェックは、リポジトリルートの `Makefile` から実行できます。主なターゲットは次のとおりです。

- `make fmt` – `gofumpt` による Go コードのフォーマット。  
- `make lint` – `golangci-lint` による静的解析。  
- `make test` – Go の単体テスト/パッケージテスト（`go test ./...`）。  
- `make providers-test` – プロバイダ統合テスト（環境変数によりゲートされる）。  
- `make coverage` – カバレッジプロファイルとそのサマリを生成。  
- `make bench` – ベンチマークを実行し、`benchstat` で集計。  
- `make constitution-check` – 完全なゲート: fmt / lint / test / providers-test / coverage / bench に加え、cgo や Python 子プロセスを禁止する監査を実行。  
- `make docs-build` – `docs/` の依存をインストールし、VitePress ドキュメントサイトをビルド。  
- `make docs-check` – `docs/` に対してパスの安全性チェック（メンテナのローカルユーザーディレクトリなど、開発者固有の絶対パスを禁止）を行い、その後フルビルドを実行。  

PR を送る前に、最低限次のコマンドを実行することを推奨します。

```bash
make fmt lint test docs-check
```

プロバイダの挙動、契約、性能に影響する変更の場合は、さらに:

```bash
make providers-test coverage bench constitution-check
```

を実行してください。

### 3. Go コードへの貢献方針

- **スタイルと構造**
  - フォーマットは `gofumpt`（`make fmt`）に任せます。  
  - パッケージ構成は既存の `go/internal` や `go/pkg` のパターンに合わせます。  
- **テスト**
  - すべてのパッケージに `_test.go` が存在することが望ましいです。  
  - 挙動に影響する変更には単体テストを追加し、対外 API やプロバイダ挙動に関わる場合は `go/tests/contract` や `go/tests/providers` に契約・統合テストを追加します。  
- **ランタイムブリッジ禁止**
  - Go ランタイムから Python をサブプロセスとして呼び出したり、cgo を利用したりしないでください。  
  - `make constitution-check` の監査により、この制約が守られているか確認されます。  

### 4. ドキュメントと仕様に関する期待値

- **仕様を単一のソース・オブ・トゥルースとする**
  - 新機能や挙動の変更では、まず `specs/` 配下の仕様を更新し、必要に応じてタスク一覧を再生成します。  
  - 仕様を起点として Go 実装と VitePress ドキュメントを更新することを推奨します。  

- **ドキュメント整合性**
  - `docs/` 以下の VitePress ドキュメントは、次の内容と整合するよう維持してください。  
    - HTTP 契約: `specs/001-go-agno-rewrite/contracts/`。  
    - プロバイダフィクスチャと挙動: `specs/001-go-agno-rewrite/contracts/fixtures/`。  
  - ドキュメントの例では、メンテナのローカルパスではなく、`./config/default.yaml` のような相対パスや汎用プレースホルダーを利用します。  

- **多言語の一貫性**
  - コアページ（概要、Quickstart、Core Features & API、Provider Matrix、Advanced Guides、Configuration & Security、Contributing & Quality）に関しては、  
    - en/zh/ja/ko の各言語で対応するページを用意し、  
    - コードサンプルは挙動が等価になるよう維持し、テキストのみローカライズしてください。  

### 5. PR に含めるとよい情報

PR を作成する際には、可能であれば以下を含めてください。

- 変更内容の概要と、対応する spec / task の番号。  
- ローカルで実行した `make` ターゲット（必要に応じて主要なコマンドの出力）。  
- ドキュメント中心の変更の場合:
  - 追加・変更されたページの簡単な説明やスクリーンショット。  
  - 後続 PR で対応する予定の項目（例: 翻訳の追従）があれば、その旨を明記。  

これらの方針に従うことで、Go ランタイム、契約、VitePress ドキュメントの整合性を保ちつつ、新しい貢献がプロジェクトの品質ゲートに適合していることを確認できます。
