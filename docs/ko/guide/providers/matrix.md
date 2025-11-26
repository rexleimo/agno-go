
# 프로바이더 매트릭스

이 페이지는 Agno-Go에서 지원하는 각 모델 프로바이더의 주요 기능과 설정 포인트를 높은 수준에서 비교합니다. 다음을 빠르게 파악하는 것이 목적입니다.

- 어떤 프로바이더가 대화(chat)에 사용 가능한지  
- 어떤 프로바이더가 벡터 임베딩(embedding)을 지원하는지(지원 여부에 따라 다름)  
- 어떤 프로바이더가 스트리밍 출력을 지원하는지  
- 각 프로바이더에서 설정해야 할 환경 변수는 무엇인지  

> 실제로 사용 가능한 모델과 기능은 각 프로바이더에서의 계정, 리전, 할당량에 따라 달라집니다. 여기의 정보는 Go 어댑터와 `.env.example`를 기준으로 하며, 자세한 제한 사항은 각 프로바이더의 공식 문서를 참고해야 합니다.

## 요약 테이블

| 프로바이더  | 챗 지원                    | 임베딩 지원                      | 스트리밍 지원                     | 주요 환경 변수                                                                     |
|-------------|----------------------------|----------------------------------|-----------------------------------|-----------------------------------------------------------------------------------|
| OpenAI      | 지원(chat)                | 지원(embeddings)                | 지원(chat 스트리밍)              | `OPENAI_API_KEY`, `OPENAI_ENDPOINT`, `OPENAI_ORG`, `OPENAI_API_VERSION`           |
| Gemini      | 지원(chat)                | 지원(embeddings)                | 지원(chat 스트리밍)              | `GEMINI_API_KEY`, `GEMINI_ENDPOINT`, `VERTEX_PROJECT`, `VERTEX_LOCATION`          |
| GLM4        | 지원(chat)                | 제한적 / 계획 중*               | 모델에 따라 다름                  | `GLM4_API_KEY`, `GLM4_ENDPOINT`                                                   |
| OpenRouter  | 지원(chat/라우팅)         | 하위 모델이 지원하는 경우에만    | 하위 모델이 지원하는 경우에만     | `OPENROUTER_API_KEY`, `OPENROUTER_ENDPOINT`                                       |
| SiliconFlow | 지원(chat)                | 지원(embeddings)                | 지원(chat 스트리밍)              | `SILICONFLOW_API_KEY`, `SILICONFLOW_ENDPOINT`                                     |
| Cerebras    | 지원(chat)                | 공식 지원 여부에 따라 다름      | 공식 지원 여부에 따라 다름        | `CEREBRAS_API_KEY`, `CEREBRAS_ENDPOINT`                                           |
| ModelScope  | 지원(chat)                | 공식 지원 여부에 따라 다름      | 공식 지원 여부에 따라 다름        | `MODELSCOPE_API_KEY`, `MODELSCOPE_ENDPOINT`                                       |
| Groq        | 지원(chat)                | 제한적 / 계획 중*               | 지원(chat 스트리밍)              | `GROQ_API_KEY`, `GROQ_ENDPOINT`                                                   |
| Ollama      | 지원(로컬 chat)           | 로컬 모델 구현에 따라 다름      | 지원(로컬 chat 스트리밍)         | `OLLAMA_ENDPOINT`                                                                 |

`*` 일부 프로바이더의 embedding 지원은 현재도 계속 발전 중입니다. 아직 완전히 지원되지 않거나 일부 모델에만 제공되는 경우, Go 어댑터는 테스트 시 지원되지 않는 호출을 건너뛰거나 계약 문서에 차이를 명시합니다.

## 설정 메모

- 프로바이더 관련 환경 변수는 모두 `.env.example`에 나열되어 있습니다. 이를 `.env`로 복사한 뒤 실제로 사용할 프로바이더에 대해서만 값을 설정하세요.  
- 필수 키가 비어 있는 경우, 헬스 체크와 프로바이더 테스트는 해당 프로바이더를 건너뛰고 그 이유를 알려 줍니다. 런타임은 설정되지 않은 프로바이더를 자동으로 호출하지 않습니다.  
- `OPENAI_ENDPOINT`와 `GEMINI_ENDPOINT` 같은 변수는 기본적으로 공식 호스팅 API를 가리키지만, 필요에 따라 프라이빗 게이트웨이나 프록시로 변경할 수 있습니다.  
- `OLLAMA_ENDPOINT`는 일반적으로 로컬에서 실행 중인 Ollama/vLLM 인스턴스(예: `http://localhost:11434/v1`)를 가리키며, 로컬 모델을 명시적으로 활성화한 경우에만 사용됩니다.  

라우팅 로직과 에러 규약에 대한 자세한 내용은 **Core Features & API 개요** 페이지와 specs 디렉터리의 계약 문서를 함께 참고하세요.

## 다음 단계

- 이 페이지에서 나열한 환경 변수의 의미와 키 관리 모범 사례는 [구성 및 보안](../config-and-security) 문서에서 더 자세히 설명합니다.  
- Quickstart 예제를 다른 프로바이더 구성으로 확장해 보고 싶다면 [퀵스타트](../quickstart) 로 돌아가 이 매트릭스를 참고해 설정을 조정해 보세요.  
