#!/usr/bin/env node
// 切片 28（母约 §9 G7 第 2 片：会话事件侧车 + 派生视图 + 跨进程恢复）的变异矩阵执行体。
// 延续 p3g6-durability.mjs 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `--- FAIL:` 名）。
//
// 用法：
//   node scripts/mutation/p5s28-sidecar.mjs            # 跑全部变异
//   node scripts/mutation/p5s28-sidecar.mjs m1 m5      # 只跑指定的几条
//   node scripts/mutation/p5s28-sidecar.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const sidecarGo = "pkg/hno/session/sidecar/sidecar.go";
const bridgeGo = "internal/hitlbridge/bridge.go";
const restoreGo = "pkg/hno/graph/restore.go";

// 命令 A = 契约绑定场景（TestP5S28_，-race -v）；命令 B = 邻近包 + 锚保护面
//（internal/session 对齐层、pkg/agentos 投影层、pkg/hno/session 本体、pkg/hno/graph 引擎全判据）。
const CMD_A = {
  label: "bound TestP5S28_",
  args: ["test", "./pkg/hno/session/sidecar/...", "./internal/hitlbridge/...", "-run", "TestP5S28_", "-count=1", "-race", "-v"],
};
const CMD_B = {
  label: "neighbors+anchors",
  args: ["test", "./pkg/hno/session/...", "./internal/session/...", "./pkg/agentos/...", "./pkg/hno/graph/...", "-count=1"],
};

const MUTANTS = [
  {
    id: "m1",
    desc: "事件写失败升级为阻塞（D9 fail-open 破 / fail-open→fail-closed 反转）：事件追加失败让 Capture 上交错误并中断会话恢复路径",
    edits: [
      {
        file: bridgeGo,
        anchor:
          '\tfor _, w := range susp.Interrupts {\n\t\t_, _ = sc.AppendEvent(ctx, sessionID, sidecar.Event{\n\t\t\tRunKey: runKey,\n\t\t\tKind:   "hitl.interrupt",\n\t\t\tPayload: map[string]any{\n\t\t\t\t"interrupt_id": w.Interrupt.InterruptID,\n\t\t\t\t"node":         w.Node,\n\t\t\t\t"mode":         int(w.Interrupt.Mode),\n\t\t\t},\n\t\t})\n\t}',
        repl:
          '\tfor _, w := range susp.Interrupts {\n\t\tif _, err := sc.AppendEvent(ctx, sessionID, sidecar.Event{\n\t\t\tRunKey: runKey,\n\t\t\tKind:   "hitl.interrupt",\n\t\t\tPayload: map[string]any{\n\t\t\t\t"interrupt_id": w.Interrupt.InterruptID,\n\t\t\t\t"node":         w.Node,\n\t\t\t\t"mode":         int(w.Interrupt.Mode),\n\t\t\t},\n\t\t}); err != nil {\n\t\t\treturn fmt.Errorf("hitlbridge: append event: %w", err)\n\t\t}\n\t}',
      },
    ],
  },
  {
    id: "m2",
    desc: "挂起记录写失败静默吞掉（D9 fail-closed 破）：保存失败被丢弃，恢复关键数据丢失不报警",
    edits: [
      {
        file: bridgeGo,
        anchor: '\tif err := sc.SaveSuspendedRun(ctx, rec); err != nil {\n\t\treturn fmt.Errorf("hitlbridge: save suspended run (fail-closed): %w", err)\n\t}',
        repl: "\t_ = sc.SaveSuspendedRun(ctx, rec)",
      },
    ],
  },
  {
    id: "m3",
    desc: "半截挂起记录放行（D2 破）：缺 session/run/等待项的记录不再被 ErrInvalidRecord 拒收",
    edits: [
      {
        file: sidecarGo,
        anchor: '\tif rec.SessionID == "" || rec.RunKey == "" || len(rec.Waiting) == 0 {\n\t\treturn ErrInvalidRecord\n\t}',
        repl: "\t_ = rec.SessionID + rec.RunKey",
      },
    ],
  },
  {
    id: "m4",
    desc: "恢复成功后不清除挂起记录（D8 生命周期破）：下次重启会对已完成的运行二次恢复",
    edits: [{ file: bridgeGo, anchor: "\t\t_ = sc.ClearSuspendedRun(ctx, sessionID, runKey)", repl: "\t\t_ = ctx" }],
  },
  {
    id: "m5",
    desc: "接线层对 ErrPendingInstalled 的良性处理摘掉（D8 安装守卫的接线半边破）：校验失败后的重试恢复在二次安装处直接失败",
    edits: [
      {
        file: bridgeGo,
        anchor: "\tif err := Install(g, rec); err != nil && !errors.Is(err, graph.ErrPendingInstalled) {",
        repl: "\tif err := Install(g, rec); err != nil {",
      },
      { file: bridgeGo, anchor: '\t"errors"\n', repl: "" },
    ],
  },
  {
    id: "m6",
    desc: "安装覆盖活挂起（D8 守卫破，恢复入口）：RestorePending 的不覆盖守卫摘掉，重悬链的新挂起可被旧记录冲掉",
    edits: [{ file: restoreGo, anchor: "\tif g.pending != nil {\n\t\treturn ErrPendingInstalled\n\t}\n", repl: "" }],
  },
  {
    id: "m7",
    desc: "账目清零回装（D7 破）：Install 不装回记录携带的 seq/steps，检查点 Seq 跨重启跳号、预算重置",
    edits: [{ file: bridgeGo, anchor: "\tsusp.SetResumeAccount(rec.Seq, rec.Steps)\n", repl: "" }],
  },
  {
    id: "m8",
    desc: "快照形状丢字段（D2 破）：Snapshot 不携带 NextSeq，水合后事件 Seq 从 1 重跳（序号续号断）",
    edits: [
      {
        file: sidecarGo,
        anchor: "return json.Marshal(snapshot{Events: m.events, Suspended: m.suspended, NextSeq: m.nextSeq})",
        repl: "return json.Marshal(snapshot{Events: m.events, Suspended: m.suspended})",
      },
    ],
  },
  {
    id: "m9",
    desc: "派生视图失联（D3 破）：PendingInterrupts 不再聚合挂起记录（视图与存储脱钩的同族可达形态——侧车不 import 主会话存储，「改读主存储」在包依赖结构上不可达，见 refactor 文档 §2 等价登记）",
    edits: [
      {
        file: sidecarGo,
        anchor: "\tvar out []InterruptRecord\n\tfor _, rec := range recs {\n\t\tfor _, w := range rec.Waiting {\n\t\t\tout = append(out, w.Interrupt)\n\t\t}\n\t}\n\treturn out, nil",
        repl: "\treturn nil, nil",
      },
    ],
  },
  {
    id: "m10",
    desc: "记录缺失静默继续（D9 点名上交破）：ResumeSaved 读不到挂起记录时不再上交 ErrNotFound，拿零值记录冒充恢复",
    edits: [
      {
        file: bridgeGo,
        anchor: '\tif !ok {\n\t\treturn nil, fmt.Errorf("hitlbridge: %w: %s/%s", sidecar.ErrNotFound, sessionID, runKey)\n\t}',
        repl: "\t_ = ok",
      },
    ],
  },
  // 预期「全绿」的等价变异：登记观察上限，不算杀红。
  {
    id: "e1",
    desc: "（预期全绿）PendingInterrupts 的聚合排序摘掉——契约 forbiddenObservations 明言聚合次序只断「稳定可复现」不断言具体算法，且全部断言只涉及单条记录，无行观察排序",
    edits: [
      {
        file: sidecarGo,
        anchor: "\tsort.Slice(recs, func(i, j int) bool { return recs[i].CreatedAt.Before(recs[j].CreatedAt) })\n",
        repl: "",
      },
      { file: sidecarGo, anchor: '\t"sort"\n', repl: "" },
    ],
  },
  {
    id: "e2",
    desc: "（预期全绿）AppendEvent 无条件重打事件墙钟 At——契约 forbiddenObservations 明言事件 At 时间戳不断言（墙钟），无行观察",
    edits: [
      {
        file: sidecarGo,
        anchor: "\tif e.At.IsZero() {\n\t\te.At = time.Now().UTC()\n\t}",
        repl: "\te.At = time.Now().UTC()",
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
