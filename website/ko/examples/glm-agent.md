# GLM Agent 예제

이 예제는 중국을 대표하는 국산 LLM 플랫폼인 GLM (智谱AI)을 HNO에서 사용하는 방법을 보여줍니다.

## 개요

GLM (Zhipu AI)는 칭화대학교 지식공학그룹에서 개발한 첨단 언어 모델입니다. 다음과 같은 기능을 제공합니다:

- **중국어 최적화**: 중국어 작업에서 뛰어난 성능
- **GLM-4**: 128K 컨텍스트를 갖춘 주요 대화 모델
- **GLM-4V**: 비전을 지원하는 멀티모달 기능
- **GLM-3-Turbo**: 빠르고 비용 효율적인 변형

## 전제 조건

1. **Go 1.21+** 설치
2. https://open.bigmodel.cn/ 에서 **GLM API 키** 획득

## API 키 발급

1. https://open.bigmodel.cn/ 방문
2. 가입 또는 로그인
3. API Keys 섹션으로 이동
4. 새 API 키 생성

API 키 형식: `{key_id}.{key_secret}`

## 설치

```bash
go get github.com/rexleimo/agno-go
```

## 환경 설정

`.env` 파일을 생성하거나 환경 변수를 내보냅니다:

```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

## 기본 예제

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
)

func main() {
    // GLM 모델 생성
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatalf("GLM 모델 생성 실패: %v", err)
    }

    // 에이전트 생성
    agent, err := agent.New(agent.Config{
        Name:         "GLM 도우미",
        Model:        model,
        Instructions: "당신은 유용한 AI 도우미입니다.",
    })
    if err != nil {
        log.Fatalf("에이전트 생성 실패: %v", err)
    }

    // 에이전트 실행
    output, err := agent.Run(context.Background(), "안녕하세요! 자기소개를 해주세요.")
    if err != nil {
        log.Fatalf("에이전트 실행 실패: %v", err)
    }

    fmt.Println(output.Content)
}
```

## 도구를 사용한 예제

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    // GLM 모델 생성
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 계산기 도구를 가진 에이전트 생성
    agent, err := agent.New(agent.Config{
        Name:         "GLM 계산 도우미",
        Model:        model,
        Toolkits:     []toolkit.Toolkit{calculator.New()},
        Instructions: "당신은 계산을 수행할 수 있는 유용한 AI 도우미입니다.",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 계산 테스트
    output, err := agent.Run(context.Background(), "123 곱하기 456은 얼마입니까?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("결과: %s\n", output.Content)
}
```

## 중국어 예제

GLM은 중국어 작업에서 탁월합니다:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
)

func main() {
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
    })
    if err != nil {
        log.Fatal(err)
    }

    agent, err := agent.New(agent.Config{
        Name:         "中文助手",
        Model:        model,
        Instructions: "你是一个有用的中文AI助手。",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 중국어로 질문
    output, err := agent.Run(context.Background(), "请用中文介绍一下人工智能的发展历史。")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

## 예제 실행

1. 저장소 복제:
```bash
git clone https://github.com/rexleimo/agno-go.git
cd HNO
```

2. API 키 설정:
```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

3. 예제 실행:
```bash
go run cmd/examples/glm_agent/main.go
```

## 구성 옵션

```go
glm.Config{
    APIKey:      string  // 필수: {key_id}.{key_secret} 형식
    BaseURL:     string  // 선택: 사용자 정의 API 엔드포인트
    Temperature: float64 // 선택: 0.0-1.0 (기본값: 0.7)
    MaxTokens:   int     // 선택: 최대 응답 토큰 수
    TopP:        float64 // 선택: Top-p 샘플링 매개변수
    DoSample:    bool    // 선택: 샘플링 활성화
}
```

## 인증

GLM은 JWT (JSON Web Token) 인증을 사용합니다:

- API 키는 `key_id`와 `key_secret`으로 분할됩니다
- HMAC-SHA256 서명을 사용하여 JWT 토큰을 생성합니다
- 토큰의 유효 기간은 7일입니다
- SDK에 의해 자동으로 처리됩니다

## 지원되는 모델

| 모델 | 컨텍스트 | 최적 용도 |
|-------|---------|----------|
| `glm-4` | 128K | 일반 대화, 중국어 |
| `glm-4v` | 128K | 비전 작업, 멀티모달 |
| `glm-3-turbo` | 128K | 빠른 응답, 비용 효율적 |

## 일반적인 문제

### 잘못된 API 키 형식

**문제**: `API key must be in format {key_id}.{key_secret}`

**해결책**: API 키에 key_id와 key_secret 사이에 점(.) 구분자가 포함되어 있는지 확인하세요.

### 인증 실패

**문제**: `GLM API error: Invalid API key`

**해결책**:
- API 키가 올바른지 확인
- https://open.bigmodel.cn/ 에서 API 키가 활성화되어 있는지 확인
- 환경 변수에 여분의 공백이 없는지 확인

### 속도 제한

**문제**: `GLM API error: Rate limit exceeded`

**해결책**:
- 지수 백오프로 재시도 로직 구현
- 요청 빈도 줄이기
- 필요시 API 플랜 업그레이드

## 다음 단계

- 다른 LLM 옵션에 대해서는 [Models](/ko/guide/models) 참조
- 기능 향상을 위해 [Tools](/ko/guide/tools) 추가
- 여러 에이전트로 [Teams](/ko/guide/team) 구축
- 복잡한 프로세스를 위해 [Workflows](/ko/guide/workflow) 탐색

## 관련 예제

- [Simple Agent](/ko/examples/simple-agent) - OpenAI 예제
- [Claude Agent](/ko/examples/claude-agent) - Anthropic 예제
- [Team Demo](/ko/examples/team-demo) - 다중 에이전트 협업

## 리소스

- [GLM 공식 웹사이트](https://www.bigmodel.cn/)
- [GLM API 문서](https://open.bigmodel.cn/dev/api)
- [HNO 저장소](https://github.com/rexleimo/agno-go)
