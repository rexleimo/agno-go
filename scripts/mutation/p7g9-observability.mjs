#!/usr/bin/env node
// 切片 25（母约 G9：观测接线）的变异矩阵执行体。
// 延续 p2g3-policies 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `-v` Test 名）。
// 与 p2g3 的唯一差别：单个变异允许跨 runner.go / resilience.go 两个实现文件
//（测试文件永远不在变异面内），恢复按全文件快照逐字节核对。
//
// 用法：
//   node scripts/mutation/p7g9-observability.mjs            # 跑全部变异
//   node scripts/mutation/p7g9-observability.mjs m1 m4      # 只跑指定的几条
//   node scripts/mutation/p7g9-observability.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const runnerGo = "pkg/hno/runner/runner.go";
const seamGo = "pkg/hno/runner/resilience.go";

// 命令 A = 契约绑定场景（TestP7G9_，-race -v）；命令 B = 全部既有判据
//（runner 包既有 16 个测试，-skip 排除本片新行）。
const CMD_A = { label: "bound TestP7G9_", args: ["test", "./pkg/hno/runner", "-run", "TestP7G9_", "-count=1", "-race", "-v"] };
const CMD_B = { label: "prior runner suite", args: ["test", "./pkg/hno/runner", "-count=1", "-race", "-v", "-skip", "TestP7G9_"] };

const MUTANTS = [
  {
    id: "m1",
    desc: "钳制摘掉（只保留 nil 守卫）—— MaxAttempts=0 透传给 observability.Retry，M1 SHAPE-2 的零迭代静默成功复现",
    kills: "D4",
    edits: [{ file: seamGo, anchor: "\tif cfg == nil || cfg.MaxAttempts < 1 {\n\t\treturn 1\n\t}", repl: "\tif cfg == nil {\n\t\treturn 1\n\t}" }],
  },
  {
    id: "m2",
    desc: "熔断记账挪进尝试闭包（每尝试记一次）—— 一回合烧光阈值，提前开断",
    kills: "D8",
    edits: [
      {
        file: seamGo,
        anchor: "\t\tspan.End()\n\t\treturn attemptErr\n\t}",
        repl: "\t\tspan.End()\n\t\tif r.breaker != nil {\n\t\t\tif attemptErr != nil {\n\t\t\t\tr.breaker.Failure()\n\t\t\t} else {\n\t\t\t\tr.breaker.Success()\n\t\t\t}\n\t\t}\n\t\treturn attemptErr\n\t}",
      },
      {
        file: seamGo,
        anchor: "\t// Breaker records ONE logical outcome per turn (post-retry), so retry\n\t// hides transients from the breaker instead of burning credits per attempt.\n\tif r.breaker != nil {\n\t\tif attemptErr != nil {\n\t\t\tr.breaker.Failure()\n\t\t} else {\n\t\t\tr.breaker.Success()\n\t\t}\n\t}\n",
        repl: "",
      },
    ],
  },
  {
    id: "m3",
    desc: "门禁摘掉 —— open 后照打模型，D5 调用数不再冻结、D7 烧尝试",
    kills: "D5, D7",
    edits: [
      {
        file: seamGo,
        anchor: "\tif r.breaker != nil && !r.breaker.Allow() {\n\t\t// Fail closed, never retried: an open breaker is a rejection, not a\n\t\t// model failure to burn attempts on. The bare sentinel keeps Run's\n\t\t// single wrap point (\"runner: model invoke: %w\") as the only prefix.\n\t\treturn nil, observability.ErrOpen\n\t}\n\n",
        repl: "",
      },
    ],
  },
  {
    id: "m4",
    desc: "成功/失败记账取反 —— 失败喂 Success、成功喂 Failure，熔断永不按失败开断",
    kills: "D5, D7",
    edits: [
      {
        file: seamGo,
        anchor: "\t\tif attemptErr != nil {\n\t\t\tr.breaker.Failure()\n\t\t} else {\n\t\t\tr.breaker.Success()\n\t\t}",
        repl: "\t\tif attemptErr != nil {\n\t\t\tr.breaker.Success()\n\t\t} else {\n\t\t\tr.breaker.Failure()\n\t\t}",
      },
    ],
  },
  {
    id: "m5",
    desc: "chat span 挪出尝试闭包（每回合一条，且首 End 后的记录丢失）—— span 粒度退化",
    kills: "D10",
    edits: [
      {
        file: seamGo,
        anchor: "\tfn := func(ctx context.Context) error {\n\t\tspanCtx, span := observability.StartChatSpan(ctx, r.provider, r.modelName)\n\t\tresp, attemptErr = r.invoker.InvokeTurn(spanCtx, req)",
        repl: "\tspanCtx, span := observability.StartChatSpan(ctx, r.provider, r.modelName)\n\tfn := func(ctx context.Context) error {\n\t\tresp, attemptErr = r.invoker.InvokeTurn(spanCtx, req)",
      },
    ],
  },
  {
    id: "m6",
    desc: "重试循环摘掉（attempts 恒 1）—— 首败即停回到今天的形状",
    kills: "D1, D3, D10, D13",
    edits: [{ file: seamGo, anchor: "\tattempts := clampAttempts(r.retry)", repl: "\tattempts := 1" }],
  },
  {
    id: "m7",
    desc: "wrap 点重复（缝内预包 runner: model invoke: 前缀，M2 双包缺陷的 D3 形态）—— 错误链前缀计数变 2",
    kills: "D2, D3, D11",
    edits: [
      {
        file: seamGo,
        anchor: "import (\n\t\"context\"\n\n\t\"github.com/rexleimo/agno-go/pkg/hno/models\"",
        repl: "import (\n\t\"context\"\n\t\"fmt\"\n\n\t\"github.com/rexleimo/agno-go/pkg/hno/models\"",
      },
      {
        file: seamGo,
        anchor: "\treturn resp, attemptErr\n}",
        repl: "\tif attemptErr != nil {\n\t\treturn resp, fmt.Errorf(\"runner: model invoke: %w\", attemptErr)\n\t}\n\treturn resp, attemptErr\n}",
      },
    ],
  },
  {
    id: "m8",
    desc: "只给 New 默认 Invoker 回退接线（调用点直调）—— 自定义 Invoker（流式聚合形态）无韧性，即「流式路径绕过缝」",
    kills: "D12",
    edits: [
      {
        file: runnerGo,
        anchor: "\t\t\tresp, err := r.invokeTurn(ctx, req)",
        repl: "\t\t\tresp, err := r.invoker.InvokeTurn(ctx, req)",
      },
      {
        file: runnerGo,
        anchor: "\tinvoker := cfg.Invoker\n\tif invoker == nil {\n\t\tmodel := cfg.Model\n\t\tinvoker = TurnInvokerFunc(func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {\n\t\t\treturn model.Invoke(ctx, req)\n\t\t})\n\t}",
        repl: "\tinvoker := cfg.Invoker\n\tif invoker == nil {\n\t\tmodel := cfg.Model\n\t\tinner := TurnInvokerFunc(func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {\n\t\t\treturn model.Invoke(ctx, req)\n\t\t})\n\t\tretry, breaker := cfg.Retry, cfg.Breaker\n\t\tprovider, modelName := model.GetProvider(), model.GetName()\n\t\tinvoker = TurnInvokerFunc(func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {\n\t\t\tif breaker != nil && !breaker.Allow() {\n\t\t\t\treturn nil, observability.ErrOpen\n\t\t\t}\n\t\t\tattempts := 1\n\t\t\tif retry != nil {\n\t\t\t\tattempts = retry.MaxAttempts\n\t\t\t\tif attempts < 1 {\n\t\t\t\t\tattempts = 1\n\t\t\t\t}\n\t\t\t}\n\t\t\tvar resp *types.ModelResponse\n\t\t\tvar attemptErr error\n\t\t\tspanFn := func(ctx context.Context) error {\n\t\t\t\tspanCtx, span := observability.StartChatSpan(ctx, provider, modelName)\n\t\t\t\tresp, attemptErr = inner.InvokeTurn(spanCtx, req)\n\t\t\t\tif attemptErr != nil {\n\t\t\t\t\tspan.RecordError(attemptErr)\n\t\t\t\t} else if resp != nil {\n\t\t\t\t\tobservability.SetUsage(span, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)\n\t\t\t\t}\n\t\t\t\tspan.End()\n\t\t\t\treturn attemptErr\n\t\t\t}\n\t\t\tif attempts == 1 {\n\t\t\t\tattemptErr = spanFn(ctx)\n\t\t\t} else {\n\t\t\t\tc := *retry\n\t\t\t\tc.MaxAttempts = attempts\n\t\t\t\tattemptErr = observability.Retry(ctx, c, spanFn)\n\t\t\t}\n\t\t\tif breaker != nil {\n\t\t\t\tif attemptErr != nil {\n\t\t\t\t\tbreaker.Failure()\n\t\t\t\t} else {\n\t\t\t\t\tbreaker.Success()\n\t\t\t\t}\n\t\t\t}\n\t\t\treturn resp, attemptErr\n\t\t})\n\t}",
      },
    ],
  },
  {
    id: "m9",
    desc: "回合预算被缝内尝试消耗（尝试数被 MaxTurns 钳制）—— MaxTurns=1 时第 2 次尝试撞预算，即 turn++ 挪进缝内的可观察形态",
    kills: "D13",
    edits: [
      {
        file: seamGo,
        anchor: "\tif attempts == 1 {\n\t\tattemptErr = fn(ctx)\n\t} else {\n\t\tcfg := *r.retry\n\t\tcfg.MaxAttempts = attempts\n\t\tattemptErr = observability.Retry(ctx, cfg, fn)\n\t}",
        repl: "\tif attempts == 1 {\n\t\tattemptErr = fn(ctx)\n\t} else {\n\t\tbudget := attempts\n\t\tif r.maxTurns > 0 && budget > r.maxTurns {\n\t\t\tbudget = r.maxTurns\n\t\t}\n\t\tcfg := *r.retry\n\t\tcfg.MaxAttempts = budget\n\t\tattemptErr = observability.Retry(ctx, cfg, fn)\n\t}",
      },
    ],
  },
  // 预期「全绿」的等价变异：登记观察上限，不算杀红。
  {
    id: "e1",
    desc: "（预期全绿）单次快路径摘掉（恒走 observability.Retry）—— Retry 对 MaxAttempts=1 恰执行 fn 一次并原样交错、无退避等待，与快路径逐语义同形；该快路径是防意外退避的保守写法而非语义承载，登记为等价变异",
    edits: [
      {
        file: seamGo,
        anchor: "\tif attempts == 1 {\n\t\tattemptErr = fn(ctx)\n\t} else {\n\t\tcfg := *r.retry\n\t\tcfg.MaxAttempts = attempts\n\t\tattemptErr = observability.Retry(ctx, cfg, fn)\n\t}",
        repl: "\tcfg := observability.RetryConfig{MaxAttempts: attempts}\n\tif r.retry != nil {\n\t\tcfg = *r.retry\n\t\tcfg.MaxAttempts = attempts\n\t}\n\tattemptErr = observability.Retry(ctx, cfg, fn)",
      },
    ],
  },
  {
    id: "e2",
    desc: "（预期全绿）仅 breaker-open 返回处预包 model invoke: 前缀（M2 缺陷的历史形态）—— errors.Is 穿透 %w 链仍可达 ErrOpen，D5/D7 只钉错误身份与调用冻结，不钉前缀形状；该缺陷靠 designConsequence 第 5 条的设计裁决修正而非测试行钉住，实测登记为观察上限",
    edits: [
      {
        file: seamGo,
        anchor: "import (\n\t\"context\"\n\n\t\"github.com/rexleimo/agno-go/pkg/hno/models\"",
        repl: "import (\n\t\"context\"\n\t\"fmt\"\n\n\t\"github.com/rexleimo/agno-go/pkg/hno/models\"",
      },
      {
        file: seamGo,
        anchor: "\t\treturn nil, observability.ErrOpen\n\t}",
        repl: "\t\treturn nil, fmt.Errorf(\"model invoke: %w\", observability.ErrOpen)\n\t}",
      },
    ],
  },
];

function sha(buf) {
  const header = Buffer.from(`blob ${buf.length}\0`);
  return createHash("sha1").update(Buffer.concat([header, buf])).digest("hex");
}

function snapshot(files) {
  const m = new Map();
  for (const f of files) m.set(f, readFileSync(`${repo}/${f}`));
  return m;
}

function apply(original, edits, file) {
  let text = original;
  for (const e of edits) {
    const parts = text.split(e.anchor);
    if (parts.length !== 2) throw new Error(`anchor 出现 ${parts.length - 1} 次，需恰好 1 次：${file} :: ${e.anchor.slice(0, 48)}`);
    text = parts[0] + e.repl + parts[1];
  }
  return text;
}

function restore(originals) {
  for (const [f, buf] of originals) writeFileSync(`${repo}/${f}`, buf);
}

function run(cmd) {
  let out = "";
  let code = 0;
  try {
    out = execFileSync("go", cmd.args, { cwd: repo, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"], timeout: 240000 });
  } catch (err) {
    code = typeof err.status === "number" ? err.status : -1;
    out = `${err.stdout ?? ""}${err.stderr ?? ""}`;
  }
  const fails = [...out.matchAll(/^--- FAIL: (\S+)/gm)].map((m) => m[1]);
  const buildFailed = /\[build failed\]|^# /.test(out) && fails.length === 0;
  return { code, fails: [...new Set(fails)].sort(), buildFailed, out };
}

function main() {
  const checkOnly = process.argv.includes("--check");
  const wanted = process.argv.slice(2).filter((a) => !a.startsWith("-"));
  const picked = wanted.length ? MUTANTS.filter((m) => wanted.includes(m.id)) : MUTANTS;
  const files = [...new Set(MUTANTS.flatMap((m) => m.edits.map((e) => e.file)))];
  if (files.some((f) => f.includes("_test.go"))) throw new Error("测试文件不得进入变异面");

  for (const m of picked) {
    for (const e of m.edits) {
      const text = readFileSync(`${repo}/${e.file}`, "utf8");
      const hits = text.split(e.anchor).length - 1;
      if (hits !== 1) throw new Error(`${m.id} 的锚在 ${e.file} 出现 ${hits} 次，需恰好 1 次：${e.anchor.slice(0, 48)}`);
    }
  }
  if (checkOnly) {
    console.log(`preflight ok: ${picked.map((m) => m.id).join(",")}`);
    return;
  }

  const baseline = snapshot(files);
  const baselineHash = Object.fromEntries([...baseline].map(([f, buf]) => [f, sha(buf)]));
  console.log(`baseline git-blob-hash ${JSON.stringify(baselineHash, null, 0)}`);
  const rows = [];
  for (const m of picked) {
    let a = null;
    let b = null;
    let scriptError = null;
    try {
      const byFile = new Map();
      for (const e of m.edits) {
        if (!byFile.has(e.file)) byFile.set(e.file, []);
        byFile.get(e.file).push(e);
      }
      for (const [f, edits] of byFile) {
        writeFileSync(`${repo}/${f}`, apply(readFileSync(`${repo}/${f}`, "utf8"), edits, f));
      }
      a = run(CMD_A);
      b = run(CMD_B);
    } catch (err) {
      scriptError = String(err.message ?? err);
    } finally {
      restore(baseline);
    }
    const drift = files.filter((f) => sha(readFileSync(`${repo}/${f}`)) !== sha(baseline.get(f)));
    rows.push({ id: m.id, desc: m.desc, aExit: a?.code, bExit: b?.code, aFails: a?.fails, bFails: b?.fails, aBuild: a?.buildFailed, bBuild: b?.buildFailed, restored: drift.length === 0 && !scriptError, scriptError });
    const r = rows.at(-1);
    console.log(
      [
        r.id,
        `exitA=${r.aExit}`,
        `exitB=${r.bExit}`,
        `restored=${r.restored}`,
        r.aBuild || r.bBuild ? "BUILD_FAILED(不计杀红)" : "",
        r.scriptError ? `SCRIPT_ERROR=${r.scriptError}` : "",
        `A_FAIL=${(r.aFails ?? []).join(",") || "-"}`,
        `B_FAIL=${(r.bFails ?? []).join(",") || "-"}`,
      ].join(" | "),
    );
  }
  const finalDrift = files.filter((f) => sha(readFileSync(`${repo}/${f}`)) !== sha(baseline.get(f)));
  console.log(`finalCheck identical=${finalDrift.length === 0} files=${files.join(",")}`);
  if (finalDrift.length) process.exitCode = 2;
}

main();
