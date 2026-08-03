# MCP 통합

## MCP란?

**모델 컨텍스트 프로토콜 (Model Context Protocol, MCP)** 은 LLM 애플리케이션과 외부 데이터 소스 및 도구 간의 원활한 통합을 가능하게 하는 개방형 표준입니다. Anthropic에서 개발한 MCP는 표준화된 인터페이스를 통해 AI 모델을 다양한 서비스에 연결하는 범용 프로토콜을 제공합니다.

**Model Context Protocol (MCP)** is an open standard that enables seamless integration between LLM applications and external data sources and tools.

## Agno-Go에서 MCP를 사용하는 이유

- **🔌 확장성** - 에이전트를 모든 MCP 호환 서버에 연결
  - **Extensibility** - Connect your agents to any MCP-compatible server
- **🔒 보안** - 내장 명령 검증 및 셸 인젝션 보호
  - **Security** - Built-in command validation and shell injection protection
- **🚀 성능** - 빠른 초기화 (<100μs) 및 낮은 메모리 사용량 (<10KB)
  - **Performance** - Fast initialization (<100μs) and low memory footprint (<10KB)
- **📦 재사용성** - 기존 MCP 서버 활용, 바퀴를 재발명하지 않음
  - **Reusability** - Leverage existing MCP servers

## 아키텍처 | Architecture

```
pkg/hno/mcp/
├── protocol/       # JSON-RPC 2.0 및 MCP 메시지 타입
├── client/         # MCP 클라이언트 코어 및 전송
├── security/       # 명령 검증 및 보안
├── content/        # 콘텐츠 타입 처리
└── toolkit/        # agno 툴킷 시스템과 통합
```

## 빠른 시작

### 전제 조건 | Prerequisites

- Go 1.21 이상 | Go 1.21 or later
- MCP 서버 (예: calculator, filesystem, git)

### 설치 | Installation

```bash
# MCP 서버 관리를 위한 uvx 설치
pip install uvx

# 샘플 MCP 서버 설치
uvx mcp install @modelcontextprotocol/server-calculator
```

### 기본 사용법 | Basic Usage

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/mcp/client"
    "github.com/rexleimo/agno-go/pkg/hno/mcp/security"
    mcptoolkit "github.com/rexleimo/agno-go/pkg/hno/mcp/toolkit"
)

// 보안 검증기 생성
// Create security validator
validator := security.NewCommandValidator()

// 전송 설정
// Setup transport
transport, err := client.NewStdioTransport(client.StdioConfig{
    Command: "python",
    Args:    []string{"-m", "mcp_server_calculator"},
})

// MCP 클라이언트 생성
// Create MCP client
mcpClient, err := client.New(transport, client.Config{
    ClientName:    "my-agent",
    ClientVersion: "1.0.0",
})

ctx := context.Background()
mcpClient.Connect(ctx)
defer mcpClient.Disconnect()

// 에이전트용 MCP 툴킷 생성
// Create MCP toolkit for agents
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client: mcpClient,
    Name:   "calculator-tools",
})
defer toolkit.Close()
```

## 보안 기능 | Security Features

### 명령 화이트리스트 | Command Whitelist

기본적으로 허용되는 명령:
- `python`, `python3`
- `node`, `npm`, `npx`
- `uvx`
- `docker`

### 셸 인젝션 보호 | Shell Injection Protection

차단되는 문자 | Blocked characters:
- `;` (명령 구분자)
- `|` (파이프)
- `&` (백그라운드 실행)
- `` ` `` (명령 치환)
- `$` (변수 확장)
- `>`, `<` (리다이렉션)

## 도구 필터링 | Tool Filtering

```go
// 특정 도구만 포함
// Include only specific tools
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client:       mcpClient,
    IncludeTools: []string{"add", "subtract", "multiply"},
})

// 특정 도구 제외
// Exclude certain tools
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client:       mcpClient,
    ExcludeTools: []string{"divide"},
})
```

## 알려진 MCP 서버 | Known MCP Servers

| 서버 | 설명 | 설치 |
|-----|------|------|
| **server-calculator** | 수학 연산 | `uvx mcp install @modelcontextprotocol/server-calculator` |
| **server-filesystem** | 파일 작업 | `uvx mcp install @modelcontextprotocol/server-filesystem` |
| **server-git** | Git 작업 | `uvx mcp install @modelcontextprotocol/server-git` |
| **server-sqlite** | SQLite 데이터베이스 | `uvx mcp install @modelcontextprotocol/server-sqlite` |

## 성능 | Performance

- **MCP 클라이언트 초기화**: <100μs
- **도구 검색**: 서버당 <50μs
- **메모리**: 연결당 <10KB
- **테스트 커버리지**: >80%

## 모범 사례 | Best Practices

1. **항상 보안 검증 사용** - 명령 검증을 우회하지 마세요
2. **도구를 적절히 필터링** - 에이전트에 필요한 도구만 노출
3. **오류를 우아하게 처리** - MCP 서버가 실패하거나 시간 초과될 수 있음
4. **연결 닫기** - 리소스 정리를 위해 항상 `defer toolkit.Close()`
5. **모의 서버로 테스트** - `pkg/hno/mcp/client/testing.go`의 테스트 유틸리티 사용

## 다음 단계 | Next Steps

- [MCP 데모](../examples/mcp-demo.md)를 시도해보세요
- [MCP 구현 가이드](../../pkg/hno/mcp/IMPLEMENTATION.md)를 읽어보세요
- [MCP 프로토콜 사양](https://spec.modelcontextprotocol.io/)을 탐색하세요
- [GitHub](https://github.com/rexleimo/agno-Go/discussions)에서 토론에 참여하세요

## 문제 해결 | Troubleshooting

**오류: "command not allowed"**
- 명령이 화이트리스트에 있는지 확인
- `validator.AddAllowedCommand()`를 사용하여 사용자 정의 명령 추가

**오류: "shell metacharacters detected"**
- 명령 인수에 위험한 문자가 포함되어 있음
- 인수에 `;`, `|`, `&` 등이 포함되지 않았는지 확인

**오류: "failed to start MCP server"**
- MCP 서버가 설치되었는지 확인
- 명령 경로가 올바른지 확인
- 필요한 권한이 있는지 확인
