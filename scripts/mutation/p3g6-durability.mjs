#!/usr/bin/env node
// 切片 24（母约 §7 G6：Durability 持久化档位）的变异矩阵执行体。
// 延续 p2g3-policies.mjs 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `-v` Test 名）。
//
// 用法：
//   node scripts/mutation/p3g6-durability.mjs            # 跑全部变异
//   node scripts/mutation/p3g6-durability.mjs m1 m5      # 只跑指定的几条
//   node scripts/mutation/p3g6-durability.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const durGo = "pkg/hno/graph/durability.go";
const schedGo = "pkg/hno/graph/scheduler.go";

// 命令 A = 契约绑定场景（TestP3G6_，-race -v）；命令 B = 切片 1–22 的全部既有判据。
const CMD_A = { label: "bound TestP3G6_", args: ["test", "./pkg/hno/graph", "-run", "TestP3G6_", "-count=1", "-race", "-v"] };
const CMD_B = { label: "prior TestP1|TestP2G3", args: ["test", "./pkg/hno/graph", "-run", "TestP1|TestP2G3", "-count=1", "-race", "-v"] };

const MUTANTS = [
  {
    id: "m1",
    desc: "Sync 提交点挪到下一轮完成之后（时序破坏）：条目滞后一笔提交，后继节点体采样必为 0，且最后一笔永不落",
    edits: [
      {
        file: schedGo,
        anchor: "\t\t\tif err := s.record(item); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}",
        repl: "\t\t\tif len(s.exitQueue) > 0 {\n\t\t\t\tlagged := s.exitQueue[len(s.exitQueue)-1]\n\t\t\t\ts.exitQueue = s.exitQueue[:len(s.exitQueue)-1]\n\t\t\t\tif err := s.record(queueItem{name: lagged.Node, out: lagged.Output}); err != nil {\n\t\t\t\t\treturn nil, err\n\t\t\t\t}\n\t\t\t}\n\t\t\ts.exitQueue = append(s.exitQueue, Checkpoint{Node: item.name, Output: item.out})",
      },
    ],
  },
  {
    id: "m2",
    desc: "默认档与 Sync 档改判 Exit 积累（默认语义破 + 时序破坏）：Sync 的落位分支失配后落入 default，退出又不冲",
    edits: [{ file: durGo, anchor: "\tswitch s.durability {\n\tcase DurabilitySync:", repl: "\tswitch s.durability {\n\tcase -1:" }],
  },
  {
    id: "m3",
    desc: "Exit 的退出落齐整段摘掉（丢条目）",
    edits: [
      {
        file: durGo,
        anchor: "\tcase DurabilityExit:\n\t\tfor _, cp := range s.exitQueue {\n\t\t\tif err := s.checkpointer.Append(s.ctx, cp); err != nil && runErr == nil {\n\t\t\t\treturn nil, fmt.Errorf(\"graph: checkpoint sink: %w\", err)\n\t\t\t}\n\t\t}",
        repl: "",
      },
    ],
  },
  {
    id: "m4",
    desc: "Async 的退出冲刷摘掉（丢条目）：close 与收敛等待整段移除、冲刷体空转（复合变异，保证丢条目判定确定性）",
    edits: [
      {
        file: durGo,
        anchor: "\tcase DurabilityAsync:\n\t\tclose(s.commits)\n\t\tif err := <-s.flushDone; err != nil && runErr == nil {\n\t\t\treturn nil, fmt.Errorf(\"graph: checkpoint sink: %w\", err)\n\t\t}",
        repl: "",
      },
      { file: durGo, anchor: "\t\tfor cp := range s.commits {", repl: "\t\tfor range s.commits {" },
      { file: durGo, anchor: "\t\t\tif err := s.checkpointer.Append(s.ctx, cp); err != nil && first == nil {", repl: "\t\t\tif err := error(nil); err != nil && first == nil {" },
    ],
  },
  {
    id: "m5",
    desc: "sink 错误吞成 nil（fail-closed 破）：Sync 即时失败改成交出 nil，Async/Exit 的退出上交判据整段钳死",
    edits: [
      { file: durGo, anchor: "\t\t\treturn fmt.Errorf(\"graph: checkpoint sink: %w\", err)", repl: "\t\t\treturn nil" },
      { file: durGo, anchor: "\t\tif err := <-s.flushDone; err != nil && runErr == nil {", repl: "\t\tif err := <-s.flushDone; err != nil && runErr == nil && false {" },
      { file: durGo, anchor: "\t\t\tif err := s.checkpointer.Append(s.ctx, cp); err != nil && runErr == nil {", repl: "\t\t\tif err := s.checkpointer.Append(s.ctx, cp); err != nil && runErr == nil && false {" },
    ],
  },
  {
    id: "m6",
    desc: "安全阀路径丢提交流（D9 同族：设了非默认步数预算就整体跳过提交——撞阀清空条目的同形后果）",
    edits: [
      {
        file: durGo,
        anchor: "\tif s.checkpointer == nil {\n\t\treturn nil\n\t}",
        repl: "\tif s.checkpointer == nil || s.stepLimit != defaultStepLimit {\n\t\treturn nil\n\t}",
      },
    ],
  },
  {
    id: "m7",
    desc: "Seq 盖章摘掉（对账破）：序号恒 0，条目流与 Result 的 Seq 连续性对账全族判红",
    edits: [{ file: durGo, anchor: "\ts.seq++", repl: "\t_ = s.seq" }],
  },
  {
    id: "m8",
    desc: "常量表零值错位（D1 结构前提破）：Sync 不再是 iota 零值，默认档随之落到不冲不落的死区",
    edits: [{ file: durGo, anchor: "\tDurabilitySync Durability = iota", repl: "\tDurabilitySync Durability = iota + 1" }],
  },
  // 预期「全绿」的等价变异：登记观察上限，不算杀红。
  {
    id: "e1",
    desc: "（预期全绿）失败路径上 Exit 档已完成条目的交付摘掉 —— D 行没有一行观察「runErr!=nil 时 Exit 落齐」（D9 只钉 Sync+撞阀），designConsequence 第 4 条的该半边登记为观察上限",
    edits: [
      {
        file: durGo,
        anchor: "\tcase DurabilityExit:\n\t\tfor _, cp := range s.exitQueue {",
        repl: "\tcase DurabilityExit:\n\t\tif runErr != nil {\n\t\t\treturn res, runErr\n\t\t}\n\t\tfor _, cp := range s.exitQueue {",
      },
    ],
  },
  {
    id: "e2",
    desc: "（预期全绿）Async 档 sink 失败的 runErr 优先钳制摘掉 —— D7 只钉 Sync/Exit 两档，Async 的 sink 失败语义无行观察（契约 observabilityLimit 第 2 条同源）",
    edits: [{ file: durGo, anchor: "\t\tif err := <-s.flushDone; err != nil && runErr == nil {", repl: "\t\tif err := <-s.flushDone; err != nil {" }],
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
