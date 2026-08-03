package agentos

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed ops_ui.html
var opsUITemplate embed.FS

// opsUI is the rendered operations dashboard (single page).
// opsUI 是运维平台仪表盘（单页）。
type opsUI struct {
	tmpl *template.Template
}

// newOpsUI loads the embedded dashboard template.
// newOpsUI 加载内嵌的仪表盘模板。
func newOpsUI() (*opsUI, error) {
	raw, err := opsUITemplate.ReadFile("ops_ui.html")
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("ops_ui").Parse(string(raw))
	if err != nil {
		return nil, err
	}
	return &opsUI{tmpl: tmpl}, nil
}

// register mounts the dashboard at /ops/ui (HTML) on the given router.
// register 将仪表盘挂载到 /ops/ui（HTML）。
func (u *opsUI) register(router *gin.Engine) {
	router.GET("/ops/ui", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		_ = u.tmpl.Execute(c.Writer, map[string]interface{}{
			"APIBase": "/api/v1/ops",
		})
	})
}

var _ = http.StatusOK // keep net/http imported for documentation builds
