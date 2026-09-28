#!/usr/bin/env node
// 切片 32（P7 第 2 片，母约 G9：run/agent 级 invoke_agent span）的变异矩阵执行体。
// 延续 p5s28-sidecar.mjs 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `--- FAIL:` 名），
// 缝位三选一的实测结论见 docs/design/v3-test-scope-p7-g9-agent-span.json M3：
// 入口 defer（选项 A）与 newKernel 内开（选项 C）均被否证，唯一合法形状是
// kernel.go 的私有 runKernel（驱动点之前开、spanCtx 交给 r.Run、End 归驱动）。
//
// 用法：
//   node scripts/mutation/p7g9s2-agent-span.mjs            # 跑全部变异
//   node scripts/mutation/p7g9s2-agent-span.mjs m-no-end   # 只跑指定的几条
//   node scripts/mutation/p7g9s2-agent-span.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const kernelGo = "pkg/hno/agent/kernel.go";
const runGo = "pkg/hno/agent/run.go";
const streamGo = "pkg/hno/agent/stream.go";

// 命令 A = 契约绑定场景（TestP7G9S2_，-race -v）；命令 B = 邻居面（切片 25 的
// span 家族：runner 的 chat span + observability 的零件判据）；命令 C = agent 包
// 整包基线（既有 15 份测试文件，P7G9S2_ 判红以 A 为准，其余判红即归属泄漏）。
const CMD_A = {
  label: "bound TestP7G9S2_",
  args: ["test", "./pkg/hno/agent", "-run", "TestP7G9S2_", "-count=1", "-race", "-v"],
};
const CMD_B = {
  label: "neighbors span-family",
  args: ["test", "./pkg/hno/runner", "./pkg/hno/observability", "-count=1"],
};
const CMD_C = {
  label: "agent prior family",
  args: ["test", "./pkg/hno/agent", "-count=1"],
};

// runKernel 的发射体（三行语义体 + 其间的空行），全部 kernel.go 变异的共同锚。
const open =
  "\tspanCtx, span := observability.StartAgentSpan(ctx, a.Name, runID)\n\tdefer span.End()\n\n\treturn r.Run(spanCtx, messages)";

const MUTANTS = [
  {
    id: "m-parent-lost",
    desc: "父链断裂（D2/D3/D4/D7 破）：span 照开、身份照对，但交给内核的是调用方 ctx 而非 spanCtx——子 span 全部失父。最小可编译形态必须写作 `_, span :=`，否则 Go 报 declared and not used",
    edits: [
      {
        file: kernelGo,
        anchor: open,
        repl: "\t_, span := observability.StartAgentSpan(ctx, a.Name, runID)\n\tdefer span.End()\n\n\treturn r.Run(ctx, messages)",
      },
    ],
  },
  {
    id: "m-no-end",
    desc: "开了从不 End（D1–D8 破，D7/abandoned 除外）：exporter 永远收不到 span，全部计数读 0",
    edits: [
      {
        file: kernelGo,
        anchor: "\tdefer span.End()\n",
        repl: "\t_ = span\n",
      },
    ],
  },
  {
    id: "m-identity-id",
    desc: "身份取错（D1 破）：agent.name 填成 a.ID 而非 a.Name——trace 看上去有名字，但夹具的标签与自动 id 可分辨",
    edits: [
      {
        file: kernelGo,
        anchor: "observability.StartAgentSpan(ctx, a.Name, runID)",
        repl: "observability.StartAgentSpan(ctx, a.ID, runID)",
      },
    ],
  },
  {
    id: "m-runid-blank",
    desc: "run_id 留空（D1/D3 破）：占位串发射，跨协程行（:342）与身份行（:261）同时失配",
    edits: [
      {
        file: kernelGo,
        anchor: "observability.StartAgentSpan(ctx, a.Name, runID)",
        repl: 'observability.StartAgentSpan(ctx, a.Name, "")',
      },
    ],
  },
  {
    id: "m-double-open",
    desc: "一次驱动开两条 span（D1–D8 破，D7/abandoned 除外）：「恰 1 条」的绝对计数全部翻倍",
    edits: [
      {
        file: kernelGo,
        anchor: open,
        repl: "\tspanCtx, span := observability.StartAgentSpan(ctx, a.Name, runID)\n\tdefer span.End()\n\t_, second := observability.StartAgentSpan(ctx, a.Name, runID)\n\tdefer second.End()\n\n\treturn r.Run(spanCtx, messages)",
      },
    ],
  },
  {
    id: "m-stream-unwired",
    desc: "流式调用点退回直调 r.Run（D3/D4/D5/D6/D7 破）：流式运行整族无主干，正控（D6）拿不到 1 条",
    edits: [
      {
        file: streamGo,
        anchor: "\t\tfinalResponse, _, stopReason, err := a.runKernel(ctx, runID, r, a.Memory.GetMessages(a.UserID))",
        repl: "\t\tfinalResponse, _, stopReason, err := r.Run(ctx, a.Memory.GetMessages(a.UserID))",
      },
    ],
  },
  {
    id: "m-sync-unwired",
    desc: "同步调用点退回直调 r.Run（D1/D2/D5/D8 破）：同步运行整族无主干，D8 的 SDK 侧正控随之死亡",
    edits: [
      {
        file: runGo,
        anchor: "\tfinalResponse, _, stopReason, err := a.runKernel(ctx, runID, r, messages)",
        repl: "\tfinalResponse, _, stopReason, err := r.Run(ctx, messages)",
      },
    ],
  },
  {
    id: "m-tracer-guard",
    desc: "被禁捷径（D1–D8 破，D7/abandoned 除外）：以 observability.IsRecording(ctx) 为发射 guard——该谓词对装了真 SDK 的裸 ctx 实测仍为 false，guard 恒真即恒零 span（S32-STD-1）",
    edits: [
      {
        file: kernelGo,
        anchor: open,
        repl: "\tif !observability.IsRecording(ctx) {\n\t\treturn r.Run(ctx, messages)\n\t}\n" + open,
      },
    ],
  },
  // 预期「全绿」的登记项：一条是等价变异（观察上限），一条是已证明的覆盖缺口（S32-STATUS-1）。
  {
    id: "m-explicit-end",
    desc: "（预期全绿，登记等价）defer End 改成单返回点后的显式 End——span 仍在驱动点结束，D1–D8 无行能分辨两者",
    edits: [
      {
        file: kernelGo,
        anchor: open,
        repl: "\tspanCtx, span := observability.StartAgentSpan(ctx, a.Name, runID)\n\n\tresp, msgs, reason, err := r.Run(spanCtx, messages)\n\tspan.End()\n\treturn resp, msgs, reason, err",
      },
    ],
  },
  {
    id: "m-status",
    desc: "（预期全绿，登记为 S32-STATUS-1 覆盖缺口）失败驱动在父 span 上 RecordError——本契约对 span 的 status/exception 无任何判据，该变异逃杀是已证明的缺口而非未做",
    edits: [
      {
        file: kernelGo,
        anchor: open,
        repl: "\tspanCtx, span := observability.StartAgentSpan(ctx, a.Name, runID)\n\tdefer span.End()\n\n\tresp, msgs, reason, err := r.Run(spanCtx, messages)\n\tif err != nil {\n\t\tspan.RecordError(err)\n\t}\n\treturn resp, msgs, reason, err",
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

function apply(original, edits) {
  let text = original;
  for (const e of edits) {
    const parts = text.split(e.anchor);
    if (parts.length !== 2) throw new Error(`anchor 出现 ${parts.length - 1} 次，需恰好 1 次：${e.file} :: ${e.anchor.slice(0, 48)}`);
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
  const passes = [...out.matchAll(/^--- PASS: (\S+)/gm)].map((m) => m[1]);
  const buildFailed = /\[build failed\]|^# /.test(out) && fails.length === 0 && passes.length === 0;
  return { code, fails: [...new Set(fails)].sort(), passes: [...new Set(passes)], buildFailed, out };
}

function main() {
  const checkOnly = process.argv.includes("--check");
  const wanted = process.argv.slice(2).filter((a) => !a.startsWith("-"));
  const picked = wanted.length ? MUTANTS.filter((m) => wanted.includes(m.id)) : MUTANTS;
  const files = [...new Set(MUTANTS.flatMap((m) => m.edits.map((e) => e.file)))];

  for (const m of picked) {
    if (new Set(m.edits.map((e) => e.file)).size !== 1) throw new Error(`${m.id} 的 edits 跨文件，脚本不支持`);
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
    let c = null;
    let scriptError = null;
    try {
      writeFileSync(`${repo}/${m.edits[0].file}`, apply(readFileSync(`${repo}/${m.edits[0].file}`, "utf8"), m.edits));
      a = run(CMD_A);
      b = run(CMD_B);
      c = run(CMD_C);
    } catch (err) {
      scriptError = String(err.message ?? err);
    } finally {
      restore(baseline);
    }
    const drift = files.filter((f) => sha(readFileSync(`${repo}/${f}`)) !== sha(baseline.get(f)));
    // C 面里 P7G9S2_ 判红以 A 面为准；其余判红即对既有家族的归属泄漏。
    const collateral = (c?.fails ?? []).filter((f) => !f.startsWith("TestP7G9S2_"));
    rows.push({ id: m.id, desc: m.desc, aExit: a?.code, bExit: b?.code, cExit: c?.code, aFails: a?.fails, aPasses: a?.passes, bFails: b?.fails, collateral, aBuild: a?.buildFailed, bBuild: b?.buildFailed, cBuild: c?.buildFailed, restored: drift.length === 0 && !scriptError, scriptError });
    const r = rows.at(-1);
    console.log(
      [
        r.id,
        `exitA=${r.aExit}`,
        `exitB=${r.bExit}`,
        `exitC=${r.cExit}`,
        `restored=${r.restored}`,
        r.aBuild || r.bBuild || r.cBuild ? "BUILD_FAILED(不计杀红)" : "",
        r.scriptError ? `SCRIPT_ERROR=${r.scriptError}` : "",
        `A_FAIL(${r.aFails?.length ?? 0})=${(r.aFails ?? []).join(",") || "-"}`,
        `B_FAIL=${(r.bFails ?? []).join(",") || "-"}`,
        `COLLATERAL=${collateral.join(",") || "-"}`,
      ].join(" | "),
    );
  }
  const finalDrift = files.filter((f) => sha(readFileSync(`${repo}/${f}`)) !== sha(baseline.get(f)));
  console.log(`finalCheck identical=${finalDrift.length === 0} files=${files.join(",")}`);
  if (finalDrift.length) process.exitCode = 2;
}

main();
