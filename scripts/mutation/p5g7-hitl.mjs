#!/usr/bin/env node
// 切片 27（母约 §9 G7：图引擎 HITL Interrupt/Resume 核心）的变异矩阵执行体。
// 延续 p3g6-durability.mjs 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `-v` Test 名）。
//
// 用法：
//   node scripts/mutation/p5g7-hitl.mjs            # 跑全部变异
//   node scripts/mutation/p5g7-hitl.mjs m1 m5      # 只跑指定的几条
//   node scripts/mutation/p5g7-hitl.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const hitlGo = "pkg/hno/graph/hitl.go";
const schedGo = "pkg/hno/graph/scheduler.go";

// 命令 A = 契约绑定场景（TestP5G7_，-race -v）；命令 B = 切片 1–26 的既有判据。
const CMD_A = { label: "bound TestP5G7_", args: ["test", "./pkg/hno/graph", "-run", "TestP5G7_", "-count=1", "-race", "-v"] };
const CMD_B = { label: "prior TestP1G|R13-R19|P2G3|P3G5/6", args: ["test", "./pkg/hno/graph", "-run", "TestP1G_|TestP1R1[3-9]|TestP2G3_|TestP3G[56]_", "-count=1", "-race", "-v"] };

const MUTANTS = [
  {
    id: "m1",
    desc: "挂起捕获摘掉：consume 不再辨认中断载体，中断退化为普通错误整单失败（D1/D10 判红）",
    edits: [
      {
        file: schedGo,
        anchor: "\t\t\t\tif sig, ok := interruptFrom(item.err); ok {\n\t\t\t\t\tif err := s.suspend(item, sig); err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\t\t\t\t\tcontinue\n\t\t\t\t}",
        repl: "",
      },
    ],
  },
  {
    id: "m2",
    desc: "schema 校验摘掉：validateResponse 恒放行（D2 判红——坏响应被当成好响应）",
    edits: [
      {
        file: hitlGo,
        anchor: "\tif len(schema) == 0 {\n\t\treturn nil\n\t}",
        repl: "\t_ = schema\n\treturn nil",
      },
    ],
  },
  {
    id: "m3",
    desc: "幂等判据摘掉：Resume 不消费挂起账，重复 Resume 重跑图而非 ErrNothingToResume（D5 判红）",
    edits: [{ file: hitlGo, anchor: "\tg.pending = nil", repl: "\t_ = g.pending" }],
  },
  {
    id: "m4",
    desc: "Handoff 档错走 Rerun：交接分支失配后落入 default，节点体被偷跑（D4 callsB 判红）",
    edits: [{ file: hitlGo, anchor: "\t\tcase ResumeHandoff:", repl: "\t\tcase InterruptMode(99):" }],
  },
  {
    id: "m5",
    desc: "EntryInterrupt 写入摘掉：挂起条目不落流（D7 条目数判红；D13 的 sink 失败半边随之失据）",
    edits: [{ file: schedGo, anchor: "\tif s.checkpointer != nil {\n\t\ts.seq++", repl: "\tif false {\n\t\ts.seq++" }],
  },
  {
    id: "m6",
    desc: "重试不吞中断的判据反转：中断反而可重试、普通错误却短路（D8 calls==1 判红）",
    edits: [
      {
        file: schedGo,
        anchor: "\t\tif _, isInterrupt := interruptFrom(err); isInterrupt {\n\t\t\tbreak\n\t\t}",
        repl: "\t\tif _, isInterrupt := interruptFrom(err); !isInterrupt {\n\t\t\tbreak\n\t\t}",
      },
    ],
  },
  {
    id: "m7",
    desc: "预算累计改每次重置：Resume 从 0 计步，跨恢复共享阀失据（D12 的 calls==2 半边判红）",
    edits: [{ file: hitlGo, anchor: "s := g.newScheduler(ctx, susp.seq, susp.steps)", repl: "s := g.newScheduler(ctx, susp.seq, 0)" }],
  },
  // 预期「全绿」的等价变异：登记观察上限，不算杀红。
  {
    id: "e1",
    desc: "（预期全绿）unknown-ID 错误也包 ErrInvalidResponse —— D6 只钉「文案点名 ID + 挂起保留」，未钉该方向的哨兵归属（契约 observabilityLimit 第 1 条同族的观察上限）",
    edits: [
      {
        file: hitlGo,
        anchor: 'return nil, fmt.Errorf("graph: response for unknown interrupt %q", id)',
        repl: 'return nil, fmt.Errorf("graph: response for unknown interrupt %q: %w", id, ErrInvalidResponse)',
      },
    ],
  },
  {
    id: "e2",
    desc: "（预期全绿）Suspension.Error 文案不再列出 ID 清单 —— D1 用 errors.As 取结构化待答清单，没有一行钉挂起错误的文案形状（forbiddenObservations：文案不是证据面）",
    edits: [
      {
        file: hitlGo,
        anchor: 'return fmt.Sprintf("graph: run suspended awaiting human response: interrupt(s) %v", ids)',
        repl: 'return fmt.Sprintf("graph: run suspended awaiting human response (%d waiting)", len(ids))',
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
  const buildFailed = /\[build failed\]|^# /.test(out) && fails.length === 0;
  return { code, fails: [...new Set(fails)].sort(), buildFailed, out };
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
    let scriptError = null;
    try {
      writeFileSync(`${repo}/${m.edits[0].file}`, apply(readFileSync(`${repo}/${m.edits[0].file}`, "utf8"), m.edits));
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
