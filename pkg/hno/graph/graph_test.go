package graph_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// TestP1G_LinearDagRunsChain 覆盖契约 §4 B1 与票面判据 1、5 的 DAG 类。
// 观察面只用 graph 的导出 API：New/AddNode/AddEdge/SetEntry/SetOutput/Run
// 与 Result 的 Output/Value/Completed。
func TestP1G_LinearDagRunsChain(t *testing.T) {
	var trace []string

	entry := graph.NodeFunc("entry", func(_ context.Context, in any) (any, error) {
		trace = append(trace, "entry")
		return "seed:" + in.(string), nil
	})
	out := graph.NodeFunc("out", func(_ context.Context, in any) (any, error) {
		trace = append(trace, "out")
		return "final<-" + in.(string), nil
	})

	g := graph.New().
		AddNode(entry).
		AddNode(out).
		AddEdge("entry", "out").
		SetEntry("entry").
		SetOutput("out")

	res, err := g.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run 未产出结果: %v", err)
	}

	if got, want := res.Output(), "final<-seed:x"; got != want {
		t.Errorf("Result.Output() = %v, want %v", got, want)
	}
	if got, ok := res.Value("entry"); !ok || got != "seed:x" {
		t.Errorf("Result.Value(\"entry\") = %v, %v; want \"seed:x\", true", got, ok)
	}
	if got, want := res.Completed(), []string{"entry", "out"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Result.Completed() = %v, want %v", got, want)
	}
	if got, want := trace, []string{"entry", "out"}; !reflect.DeepEqual(got, want) {
		t.Errorf("节点执行轨迹 = %v, want %v（线性链为因果关系，可作全序断言）", got, want)
	}
}

// TestP1G_ValidateRejectsDanglingEdge 覆盖契约 §11.1（切片 2）与票面判据 3、5。
// 观察面：Validate 的返回值与错误可定位性、Run 对非法图的拒绝（不得崩溃）。
func TestP1G_ValidateRejectsDanglingEdge(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }

	t.Run("悬空边被拒且可定位", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddEdge("a", "ghost").
			SetEntry("a").
			SetOutput("a")

		err := g.Validate()
		if err == nil {
			t.Fatal("出边指向不存在的节点，Validate 却判定合法")
		}
		if !strings.Contains(err.Error(), "ghost") {
			t.Errorf("Validate 错误未指出问题节点 %q: %v", "ghost", err)
		}

		res, runErr := g.Run(context.Background(), "x")
		if runErr == nil {
			t.Fatalf("非法图仍可执行，且返回了结果 %v", res)
		}
	})

	t.Run("合法图通过校验并可执行", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("b", identity)).
			AddEdge("a", "b").
			SetEntry("a").
			SetOutput("b")

		if err := g.Validate(); err != nil {
			t.Fatalf("合法图被拒: %v", err)
		}
		res, err := g.Run(context.Background(), "seed")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if got, want := res.Output(), "seed"; got != want {
			t.Errorf("Output() = %v, want %v", got, want)
		}
	})
}
