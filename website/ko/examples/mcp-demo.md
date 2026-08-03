# MCP 데모 예제

## 개요

이 예제는 MCP (Model Context Protocol) 서버에 연결하고 HNO MCP 클라이언트를 통해 해당 도구를 사용하는 방법을 보여줍니다. 보안 검증 설정, 전송 생성, MCP 서버 연결, MCP 도구와 Agno 에이전트 통합 방법을 배우게 됩니다.

## 학습 내용

- MCP 명령에 대한 보안 검증 생성 및 구성 방법
- 하위 프로세스 통신을 위한 stdio 전송 설정 방법
- MCP 서버 연결 및 사용 가능한 도구 발견 방법
- Agno 에이전트에서 사용할 MCP 툴킷 생성 방법
- MCP 도구를 직접 호출하는 방법

## 전제 조건

- Go 1.21 이상
- 설치된 MCP 서버 (예: calculator 서버)

## 설정

### 1. MCP 서버 설치

```bash
# uvx 패키지 매니저 설치
pip install uvx

# calculator MCP 서버 설치
uvx mcp install @modelcontextprotocol/server-calculator

# 설치 확인
python -m mcp_server_calculator --help
```

### 2. 예제 실행

```bash
# 예제 디렉토리로 이동
cd cmd/examples/mcp_demo

# 직접 실행
go run main.go

# 또는 빌드 후 실행
go build -o mcp_demo
./mcp_demo
```

## 완전한 코드

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/client"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/security"
	mcptoolkit "github.com/rexleimo/agno-go/pkg/hno/mcp/toolkit"
)

func main() {
	fmt.Println("=== HNO MCP Demo ===")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 단계 1: 보안 검증 도구 생성
	fmt.Println("Step 1: Creating security validator...")
	validator := security.NewCommandValidator()

	command := "python"
	args := []string{"-m", "mcp_server_calculator"}

	if err := validator.Validate(command, args); err != nil {
		log.Fatalf("Command validation failed: %v", err)
	}
	fmt.Printf("✓ Command validated: %s %v\n", command, args)

	// 단계 2: 전송 생성
	fmt.Println("Step 2: Creating transport...")
	transport, err := client.NewStdioTransport(client.StdioConfig{
		Command: command,
		Args:    args,
	})
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}
	fmt.Println("✓ Stdio transport created")

	// 단계 3: MCP 클라이언트 생성
	fmt.Println("Step 3: Creating MCP client...")
	mcpClient, err := client.New(transport, client.Config{
		ClientName:    "agno-go-demo",
		ClientVersion: "0.1.0",
	})
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	fmt.Println("✓ MCP client created")

	// 단계 4: 서버에 연결
	fmt.Println("Step 4: Connecting to MCP server...")
	if err := mcpClient.Connect(ctx); err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer mcpClient.Disconnect()

	fmt.Println("✓ Connected to MCP server")
	if serverInfo := mcpClient.GetServerInfo(); serverInfo != nil {
		fmt.Printf("  Server: %s v%s\n", serverInfo.Name, serverInfo.Version)
	}

	// 단계 5: 도구 발견
	fmt.Println("Step 5: Discovering tools...")
	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("✓ Found %d tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// 단계 6: MCP 툴킷 생성
	fmt.Println("Step 6: Creating MCP toolkit...")
	toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
		Client: mcpClient,
		Name:   "calculator-tools",
	})
	if err != nil {
		log.Fatalf("Failed to create toolkit: %v", err)
	}
	defer toolkit.Close()

	fmt.Println("✓ MCP toolkit created")
	fmt.Printf("  Toolkit name: %s\n", toolkit.Name())
	fmt.Printf("  Available functions: %d\n", len(toolkit.Functions()))

	// 단계 7: 도구 직접 호출
	fmt.Println("Step 7: Calling a tool...")
	result, err := mcpClient.CallTool(ctx, "add", map[string]interface{}{
		"a": 5,
		"b": 3,
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}

	fmt.Println("✓ Tool call successful")
	fmt.Printf("  Result: %v\n", result.Content)

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("The MCP toolkit can now be passed to an agno Agent!")
}
```

## 코드 설명

### 1. 보안 검증

```go
validator := security.NewCommandValidator()
if err := validator.Validate(command, args); err != nil {
    log.Fatalf("Command validation failed: %v", err)
}
```

- 기본 화이트리스트를 사용하여 보안 검증 도구 생성
- 명령이 안전하게 실행될 수 있는지 검증
- 위험한 셸 메타문자 차단

### 2. Stdio 전송

```go
transport, err := client.NewStdioTransport(client.StdioConfig{
    Command: "python",
    Args:    []string{"-m", "mcp_server_calculator"},
})
```

- stdin/stdout를 통해 통신하는 전송 생성
- MCP 서버를 하위 프로세스로 시작
- 양방향 JSON-RPC 2.0 메시지 처리

### 3. MCP 클라이언트

```go
mcpClient, err := client.New(transport, client.Config{
    ClientName:    "agno-go-demo",
    ClientVersion: "0.1.0",
})
```

- 애플리케이션 식별자를 사용하여 MCP 클라이언트 생성
- 연결 수명 주기 관리
- 도구 발견 및 호출 메서드 제공

### 4. 도구 발견

```go
tools, err := mcpClient.ListTools(ctx)
for _, tool := range tools {
    fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
}
```

- MCP 서버에서 사용 가능한 도구 쿼리
- 도구 메타데이터 (이름, 설명, 매개변수) 반환
- 동적 도구 발견에 사용

### 5. MCP 툴킷 생성

```go
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client: mcpClient,
    Name:   "calculator-tools",
})
defer toolkit.Close()
```

- MCP 도구를 Agno 툴킷 함수로 변환
- MCP 스키마에서 함수 시그니처 자동 생성
- `agent.Config.Toolkits`와 호환

### 6. 직접 도구 호출

```go
result, err := mcpClient.CallTool(ctx, "add", map[string]interface{}{
    "a": 5,
    "b": 3,
})
fmt.Printf("Result: %v\n", result.Content)
```

- 에이전트 없이 MCP 도구를 직접 호출
- 매개변수를 맵으로 전달
- 결과 내용 반환

## 예상 출력

```
=== HNO MCP Demo ===

Step 1: Creating security validator...
✓ Command validated: python [-m mcp_server_calculator]

Step 2: Creating transport...
✓ Stdio transport created

Step 3: Creating MCP client...
✓ MCP client created

Step 4: Connecting to MCP server...
✓ Connected to MCP server
  Server: calculator v0.1.0

Step 5: Discovering tools...
✓ Found 4 tools:
  - add: Add two numbers
  - subtract: Subtract two numbers
  - multiply: Multiply two numbers
  - divide: Divide two numbers

Step 6: Creating MCP toolkit...
✓ MCP toolkit created
  Toolkit name: calculator-tools
  Available functions: 4

Step 7: Calling a tool...
✓ Tool call successful
  Result: 8

=== Demo Complete ===
The MCP toolkit can now be passed to an agno Agent!
```

## Agno 에이전트와 함께 사용

MCP 툴킷을 얻으면 모든 Agno 에이전트와 함께 사용할 수 있습니다:

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

// 모델 생성
model, _ := openai.New("gpt-4o-mini", openai.Config{
    APIKey: "your-api-key",
})

// MCP 툴킷을 사용하여 에이전트 생성
ag, _ := agent.New(agent.Config{
    Name:     "MCP Calculator Agent",
    Model:    model,
    Toolkits: []toolkit.Toolkit{toolkit},  // MCP toolkit here!
})

// 에이전트 실행
output, _ := ag.Run(context.Background(), "What is 25 * 4 + 15?")
fmt.Println(output.Content)
```

## 문제 해결

**오류: "command not allowed"**
- MCP 서버 명령이 보안 화이트리스트에 있는지 확인
- `validator.AddAllowedCommand("your-command")`로 추가

**오류: "failed to start process"**
- MCP 서버가 설치되어 있는지 확인: `python -m mcp_server_calculator --help`
- Python이 PATH에 있는지 확인

**오류: "connection timeout"**
- MCP 서버가 시작하는 데 시간이 오래 걸릴 수 있음
- 컨텍스트 타임아웃 늘리기: `context.WithTimeout(ctx, 60*time.Second)`

**도구 호출이 오류 반환**
- 도구가 존재하는지 확인: `mcpClient.ListTools(ctx)` 확인
- 매개변수 유형이 도구 스키마와 일치하는지 확인

## 다음 단계

- [MCP 통합 가이드](../guide/mcp.md) 읽기
- 다른 MCP 서버 (filesystem, git, sqlite) 연결 시도
- 사용 사례에 맞는 사용자 정의 MCP 서버 구축
- MCP 도구와 내장 Agno 도구 결합
