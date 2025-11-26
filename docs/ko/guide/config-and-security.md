## 구성 및 보안 모범 사례

이 페이지에서는 Agno-Go 서버를 어떻게 설정하고, 각 모델 프로바이더의 자격 증명을 안전하게 관리할지 설명합니다. 예제는 리포지터리 루트에서 Go 1.25.1로 서버를 실행하고, 기본 설정 파일을 사용하는 것을 전제로 합니다.

### 1. 설정 파일과 환경

- `config/default.yaml` – 서버, 프로바이더, 메모리, 런타임에 대한 기본 설정.  
- `.env` – 모델 프로바이더의 API 키와 사용자 정의 엔드포인트.  
- `.env.example` – 안전하게 공유 가능한 템플릿. 로컬에서 복사하여 사용합니다.  

권장 흐름:

1. 샘플 환경 파일 복사:

   ```bash
   cp .env.example .env
   ```

2. 사용하려는 프로바이더(예: OpenAI 또는 Groq)의 키를 `.env` 에 채웁니다.  
3. 기본 설정으로 서버를 시작합니다:

   ```bash
   cd go
   go run ./cmd/agno --config ../config/default.yaml
   ```

### 2. 핵심 환경 변수

`.env.example` 에는 지원되는 모든 프로바이더가 나열되어 있습니다. 주요 변수는 다음과 같습니다.

- **OpenAI**
  - `OPENAI_API_KEY` – OpenAI를 활성화하기 위해 필수.  
  - `OPENAI_ENDPOINT` – 선택 사항. 프록시나 Azure 스타일 엔드포인트를 사용할 때 재정의.  
  - `OPENAI_ORG`, `OPENAI_API_VERSION` – 선택 사항. 조직 범위 또는 프리뷰 API용.  

- **Gemini / Vertex**
  - `GEMINI_API_KEY` – Gemini API를 직접 사용할 때 필요.  
  - `GEMINI_ENDPOINT` – 선택 사항. 기본값은 공개 Generative Language API.  
  - `VERTEX_PROJECT`, `VERTEX_LOCATION` – 선택 사항. Vertex AI 사용 시 설정.  

- **GLM4**
  - `GLM4_API_KEY` – GLM4를 활성화하기 위해 필수.  
  - `GLM4_ENDPOINT` – 기본 공개 엔드포인트. 필요에 따라 프록시로 변경 가능.  

- **OpenRouter**
  - `OPENROUTER_API_KEY` – OpenRouter를 활성화하기 위해 필수.  
  - `OPENROUTER_ENDPOINT` – 선택 사항. 커스텀 라우팅에 사용.  

- **SiliconFlow / Cerebras / ModelScope / Groq**
  - `SILICONFLOW_API_KEY`, `CEREBRAS_API_KEY`, `MODELSCOPE_API_KEY`, `GROQ_API_KEY` – 각 프로바이더에 대한 필수 키.  
  - `SILICONFLOW_ENDPOINT`, `CEREBRAS_ENDPOINT`, `MODELSCOPE_ENDPOINT`, `GROQ_ENDPOINT` – 선택적인 엔드포인트 재정의.  

- **Ollama / 로컬 모델**
  - `OLLAMA_ENDPOINT` – 로컬 모델 서버의 HTTP 엔드포인트. 비워 두면 비활성으로 간주됩니다.  

규칙:

- 필수 키가 비어 있으면 해당 프로바이더는 “구성되지 않음” 상태로 간주됩니다.  
- 헬스 체크와 프로바이더 테스트는 해당 프로바이더를 건너뛰고, 이유를 명확하게 표시해야 합니다.  

### 3. `config/default.yaml` 개요

기본 설정 파일은 서버 동작을 제어합니다.

- **server**
  - `server.host` – 리슨 주소(기본값 `0.0.0.0`).  
  - `server.port` – HTTP API 포트(기본값 `8080`).  

- **providers**
  - `providers.<name>.endpoint` – 해당 환경 변수에서 값을 읽습니다(예: `${OPENAI_ENDPOINT}`, `${GROQ_ENDPOINT}`).  
  - 프로바이더 활성화 여부는 env의 키 존재 여부에 따라 결정됩니다.  

- **memory**
  - `memory.storeType` – `memory` / `bolt` / `badger` 중 하나.  
  - `memory.tokenWindow` – 대화 컨텍스트에 유지할 토큰 수.  

- **runtime / bench**
  - `runtime.maxConcurrentRequests`, `runtime.requestTimeout`, `runtime.router.*` 필드는 동시성 및 타임아웃 전략을 제어합니다.  
  - `bench` 섹션은 내부 벤치마크용 기본 값이며, 일반적인 사용에서는 수정할 필요가 없습니다.  

`config/default.yaml` 은 버전 관리 하에 두고, 환경별 차이나 비밀 값은 `.env` 또는 배포 환경의 환경 변수로 오버라이드하는 방식을 권장합니다.

### 4. 보안 모범 사례

- **실제 시크릿을 커밋하지 않기**
  - `.env` 나 실제 API 키가 포함된 파일은 리포지터리에 커밋하지 마십시오.  
  - 이 리포지터리의 `.gitignore` 에는 이미 `.env` 와 일반적인 로컬 설정 파일이 포함되어 있습니다.  

- **문서와 샘플에서는 플레이스홀더 사용**
  - 쉘 예제에서는 `OPENAI_API_KEY=...` 와 같은 플레이스홀더를 사용하여, 실제 키가 히스토리나 공유 스니펫에 남지 않도록 합니다.  
  - 코드 샘플에서는 `./config/default.yaml` 과 같은 상대 경로를 사용하고, 개발자 머신에 종속된 절대 경로는 피합니다.  

- **환경 분리**
  - 가능하다면 개발/스테이징/프로덕션 환경에 서로 다른 키 또는 프로젝트를 사용하십시오.  
  - 장기간 사용하는 시크릿은 플랫폼의 시크릿 매니저나 CI 시크릿 저장소에서 관리하는 것을 고려하십시오.  

- **감사 및 테스트**
  - 변경 사항을 커밋하기 전에 다음 명령을 실행하는 것을 권장합니다:

    ```bash
    make test providers-test coverage bench constitution-check
    ```

  - 이를 통해 프로바이더 통합이 예상대로 작동하는지, 위험한 설정 변경이 없는지 확인할 수 있습니다.  
