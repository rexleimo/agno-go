## 設定とセキュリティのベストプラクティス

このページでは、Agno-Go の設定方法と、各プロバイダの認証情報を安全に扱うためのガイドラインを説明します。ここでは、リポジトリのルートから Go 1.25.1 でサーバーを起動し、デフォルトの設定ファイルを利用することを前提とします。

### 1. 設定ファイルと環境

- `config/default.yaml` – サーバー、プロバイダ、メモリ、ランタイムのデフォルト設定。  
- `.env` – モデルプロバイダの API キーやカスタムエンドポイント。  
- `.env.example` – 安全に共有できるテンプレート。ローカルでコピーして編集します。  

推奨フロー:

1. サンプル環境ファイルをコピー:

   ```bash
   cp .env.example .env
   ```

2. 利用したいプロバイダ（例: OpenAI や Groq）のキーを `.env` に設定します。  
3. デフォルト設定でサーバーを起動します:

   ```bash
   cd go
   go run ./cmd/agno --config ../config/default.yaml
   ```

### 2. 主要な環境変数

`.env.example` にはサポートされているすべてのプロバイダが列挙されています。主な変数は次のとおりです。

- **OpenAI**
  - `OPENAI_API_KEY` – OpenAI を有効にするために必須。  
  - `OPENAI_ENDPOINT` – 任意。プロキシや Azure 互換エンドポイントを使う場合に上書き。  
  - `OPENAI_ORG`, `OPENAI_API_VERSION` – 任意。組織スコープやプレビュー API 用。  

- **Gemini / Vertex**
  - `GEMINI_API_KEY` – Gemini API を直接利用する場合に必要。  
  - `GEMINI_ENDPOINT` – 任意。デフォルトは公開 Generative Language API。  
  - `VERTEX_PROJECT`, `VERTEX_LOCATION` – 任意。Vertex AI を利用する場合に設定。  

- **GLM4**
  - `GLM4_API_KEY` – GLM4 を有効にするために必須。  
  - `GLM4_ENDPOINT` – デフォルトの公開エンドポイント。必要に応じてプロキシに変更可能。  

- **OpenRouter**
  - `OPENROUTER_API_KEY` – OpenRouter を有効にするために必須。  
  - `OPENROUTER_ENDPOINT` – 任意。カスタムルーティングに利用。  

- **SiliconFlow / Cerebras / ModelScope / Groq**
  - `SILICONFLOW_API_KEY`, `CEREBRAS_API_KEY`, `MODELSCOPE_API_KEY`, `GROQ_API_KEY` – 各プロバイダの必須キー。  
  - `SILICONFLOW_ENDPOINT`, `CEREBRAS_ENDPOINT`, `MODELSCOPE_ENDPOINT`, `GROQ_ENDPOINT` – 任意のエンドポイント上書き。  

- **Ollama / ローカルモデル**
  - `OLLAMA_ENDPOINT` – ローカルモデルサーバーの HTTP エンドポイント。空のままにすると無効扱い。  

ルール:

- 必須キーが空の場合、そのプロバイダは「未設定」とみなされます。  
- ヘルスチェックやプロバイダテストは、そのプロバイダをスキップし、理由を明示することが期待されています。  

### 3. `config/default.yaml` の概要

デフォルト設定ファイルはサーバーの動作を制御します。

- **server**
  - `server.host` – 監視アドレス（デフォルト `0.0.0.0`）。  
  - `server.port` – HTTP API ポート（デフォルト `8080`）。  

- **providers**
  - `providers.<name>.endpoint` – 対応する環境変数から読み込みます（例: `${OPENAI_ENDPOINT}`, `${GROQ_ENDPOINT}`）。  
  - プロバイダの有効/無効は env のキーに基づいて判断されます。  

- **memory**
  - `memory.storeType` – `memory` / `bolt` / `badger` のいずれか。  
  - `memory.tokenWindow` – 会話コンテキストに保持するトークン数。  

- **runtime / bench**
  - `runtime.maxConcurrentRequests`, `runtime.requestTimeout`, `runtime.router.*` などは同時実行数やタイムアウトを制御します。  
  - `bench` セクションは内部ベンチマーク用のデフォルト値であり、通常利用では変更不要です。  

`config/default.yaml` 自体はバージョン管理下に置き、環境依存の値や秘密情報は `.env` やデプロイ環境の環境変数で上書きする運用を推奨します。

### 4. セキュリティのベストプラクティス

- **実際のシークレットをコミットしない**
  - `.env` や、実際の API キーを含むファイルはリポジトリにコミットしないでください。  
  - このリポジトリの `.gitignore` には、`.env` やローカル設定ファイルが既に含まれています。  

- **ドキュメントやサンプルではプレースホルダーを使う**
  - シェルの例では `OPENAI_API_KEY=...` のようなプレースホルダーを使い、実際のキーを履歴や共有スニペットに残さないようにします。  
  - コードサンプルでは、`./config/default.yaml` のような相対パスを使い、マシン固有の絶対パスを避けます。  

- **環境ごとに分離する**
  - 可能であれば、開発・ステージング・本番で異なるキーやプロジェクトを利用してください。  
  - 長期利用するシークレットは、プラットフォームのシークレットマネージャーや CI のシークレットストアで管理することを検討してください。  

- **監査とテスト**
  - 変更をコミットする前に、次のコマンドを実行することを推奨します:

    ```bash
    make test providers-test coverage bench constitution-check
    ```

  - これにより、プロバイダ統合が期待どおりに動作していることや、安全でない設定変更が入っていないことを確認できます。  
