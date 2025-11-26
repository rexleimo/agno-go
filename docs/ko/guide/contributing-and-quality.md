## 기여 및 품질 게이트

이 페이지에서는 Agno-Go 프로젝트에 기여하는 방법과, 변경 사항이 머지되기 전에 통과해야 하는 품질 게이트를 설명합니다.

### 1. 먼저 읽어야 할 문서

- **리포지터리 루트의 `AGENTS.md`**  
  - 프로젝트 구조(`go/`, `docs/`, `specs/`, `scripts/` 등).  
  - 런타임 제약(순수 Go 런타임, Python/cgo 브리지는 허용되지 않음).  
  - 스펙과 픽스처가 어떻게 동작을 제약하는지.  
- **구현 전에 관련 스펙 확인**  
  - AgentOS 계약과 픽스처: `specs/001-go-agno-rewrite/`.  
  - VitePress 문서 계획: `specs/001-vitepress-docs/`.  

비교적 큰 변경의 경우, 런타임 코드를 수정하기 전에 해당 spec / plan / tasks 를 먼저 갱신하는 것을 권장합니다.

### 2. 핵심 `make` 타깃

모든 품질 검사는 리포지터리 루트의 `Makefile` 을 통해 실행됩니다. 주요 타깃은 다음과 같습니다.

- `make fmt` – `gofumpt` 를 사용해 Go 코드를 포맷.  
- `make lint` – `golangci-lint` 를 사용한 정적 분석.  
- `make test` – Go 단위/패키지 테스트(`go test ./...`).  
- `make providers-test` – 프로바이더 통합 테스트(환경 변수로 게이트됨).  
- `make coverage` – 커버리지 프로파일과 요약 보고서를 생성.  
- `make bench` – 벤치마크를 실행하고 `benchstat` 으로 요약.  
- `make constitution-check` – 전체 게이트 실행: fmt / lint / test / providers-test / coverage / bench 에 더해, cgo 및 Python 서브프로세스 사용 여부를 검사하는 감사 포함.  
- `make docs-build` – `docs/` 에 대한 의존성을 설치하고 VitePress 문서 사이트를 빌드.  
- `make docs-check` – `docs/` 에 대한 경로 안전성 검사(로컬 사용자 디렉터리 등 유지보수자 환경에 특화된 절대 경로를 금지) 후, 전체 문서 빌드를 수행.  

PR 을 만들기 전에, 최소한 다음 명령을 실행하는 것이 좋습니다.

```bash
make fmt lint test docs-check
```

프로바이더 동작, 계약, 성능에 영향을 주는 변경인 경우에는 다음도 함께 실행하는 것을 권장합니다.

```bash
make providers-test coverage bench constitution-check
```

### 3. Go 코드 기여 가이드

- **스타일과 구조**
  - 포맷은 `gofumpt` (`make fmt`) 에 맡깁니다.  
  - 패키지 구조는 기존 `go/internal`, `go/pkg` 의 패턴에 맞추는 것을 권장합니다.  
- **테스트**
  - 모든 패키지에 `_test.go` 파일이 존재하는 것이 이상적입니다.  
  - 동작에 영향을 주는 변경은 단위 테스트로 커버하고, 외부 API 나 프로바이더 동작과 관련된 변경은 `go/tests/contract` 나 `go/tests/providers` 에 계약/통합 테스트를 추가합니다.  
- **런타임 브리지 금지**
  - Go 런타임에서 Python 을 서브프로세스로 호출하거나 cgo 브리지를 사용하는 것은 허용되지 않습니다.  
  - `make constitution-check` 타깃의 감사가 이러한 제약을 검사합니다.  

### 4. 문서와 스펙에 대한 기대치

- **스펙을 단일 진실 소스로 사용**
  - 새로운 기능이나 동작 변경 시, 먼저 `specs/` 아래의 관련 스펙을 업데이트하고 필요한 경우 작업 목록을 재생성합니다.  
  - 스펙을 기준으로 Go 구현과 VitePress 문서를 변경하는 것을 권장합니다.  

- **문서 정합성 유지**
  - `docs/` 아래의 VitePress 문서는 다음과 일관되게 유지해야 합니다.  
    - HTTP 계약: `specs/001-go-agno-rewrite/contracts/`.  
    - 프로바이더 픽스처 및 동작: `specs/001-go-agno-rewrite/contracts/fixtures/`.  
  - 문서 예제에서는 `./config/default.yaml` 과 같은 상대 경로 또는 범용 플레이스홀더를 사용하고, 유지보수자 로컬 환경에 특화된 절대 경로는 피합니다.  

- **다국어 일관성**
  - 핵심 페이지(개요, Quickstart, Core Features & API, Provider Matrix, Advanced Guides, Configuration & Security, Contributing & Quality)에 대해서는:  
    - en/zh/ja/ko 모든 언어에 대응되는 페이지가 있어야 하며,  
    - 코드 예제는 동작이 동일하도록 유지하고, 텍스트만 현지화해야 합니다.  

### 5. PR 에 포함하면 좋은 내용

PR 을 만들 때는 가능하면 다음을 포함해 주세요.

- 변경 내용의 요약 및 관련되는 spec / task 번호.  
- 로컬에서 실행한 `make` 타깃(필요시 핵심 명령의 출력 포함).  
- 문서 중심 변경인 경우:  
  - 새로 추가되거나 수정된 페이지에 대한 간단한 설명이나 스크린샷.  
  - 후속 PR 에서 처리할 예정인 부분(예: 번역 보완)이 있다면 명시.  

이러한 원칙을 따르면 Go 런타임, 계약, VitePress 문서 간의 정합성을 유지하면서도, 새로운 기여가 프로젝트의 품질 게이트를 충족하도록 도울 수 있습니다.
