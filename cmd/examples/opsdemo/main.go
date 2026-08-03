// opsdemo AgentOS 运维平台演示
// 启动后访问 http://localhost:18081/ops/ui
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/agentos"
	"github.com/rexleimo/agno-go/pkg/hno/skills"
)

func main() {
	// 加载示例技能（项目 skills/ 目录）
	loader := skills.NewLoader(os.DirFS("skills"))
	reg, err := skills.NewRegistry(loader, ".")
	if err != nil {
		log.Fatalf("skills registry: %v", err)
	}
	catalog := reg.Catalog()
	fmt.Printf("已加载 %d 个技能\n", len(catalog))

	// 创建 server（带技能注册表）
	server, err := agentos.NewServer(&agentos.Config{
		Address:        ":18081",
		Debug:          true,
		SkillsRegistry: reg,
	})
	if err != nil {
		log.Fatalf("server: %v", err)
	}

	fmt.Println("启动 AgentOS 运维平台: http://localhost:18081/ops/ui")
	fmt.Println("API 端点: /api/v1/ops/{skills,observability,eval-runs}")

	// 服务就绪后写入演示评估数据
	go func() {
		time.Sleep(1 * time.Second)
		seed := func(model string, runs, successes int, dur string) {
			payload := fmt.Sprintf(`{"model":%q,"runs":%d,"successes":%d,"duration":%q}`,
				model, runs, successes, dur)
			resp, err := http.Post("http://127.0.0.1:18081/api/v1/ops/eval-runs",
				"application/json", strings.NewReader(payload))
			if err != nil {
				log.Printf("seed %s: %v", model, err)
				return
			}
			resp.Body.Close()
			fmt.Printf("已写入评估记录: %s (%d/%d 通过)\n", model, successes, runs)
		}
		seed("gpt-4o-mini", 12, 11, "3.2s")
		seed("claude-3-5-haiku", 8, 8, "2.1s")
	}()

	if err := server.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
