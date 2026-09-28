#!/usr/bin/env node
// 切片 23（母约 §5 G4：StreamMode 七模式统一流式协议的协议层）的变异矩阵执行体。
// 延续 p2g3-policies.mjs 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `-v` 的 `--- FAIL:` Test 名）。
//
// 用法：
//   node scripts/mutation/p4g4-streammode.mjs            # 跑全部变异
//   node scripts/mutation/p4g4-streammode.mjs m2 m4      # 只跑指定的几条
//   node scripts/mutation/p4g4-streammode.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const eventsJsonGo = "pkg/hno/run/events_json.go";
const streamEventsGo = "pkg/hno/run/stream_events.go";
const agentGo = "pkg/hno/agent/agent.go";
const streamGo = "pkg/hno/agent/stream.go";

// decodeEvent 的六个精确匹配 case（整块），切片 23 插入在 contains 归一化之前。
const EXACT_BLOCK = [
  "\tcase kind == EventTypeNodeStarted:",
  "\t\tvar evt NodeStartedEvent",
  "\t\tif err := json.Unmarshal(raw, &evt); err != nil {",
  "\t\t\treturn nil, err",
  "\t\t}",
  "\t\treturn &evt, nil",
  "\tcase kind == EventTypeNodeCompleted:",
  "\t\tvar evt NodeCompletedEvent",
  "\t\tif err := json.Unmarshal(raw, &evt); err != nil {",
  "\t\t\treturn nil, err",
  "\t\t}",
  "\t\treturn &evt, nil",
  "\tcase kind == EventTypeTaskError:",
  "\t\tvar evt TaskErrorEvent",
  "\t\tif err := json.Unmarshal(raw, &evt); err != nil {",
  "\t\t\treturn nil, err",
  "\t\t}",
  "\t\treturn &evt, nil",
  "\tcase kind == EventTypeCheckpoint:",
  "\t\tvar evt CheckpointEvent",
  "\t\tif err := json.Unmarshal(raw, &evt); err != nil {",
  "\t\t\treturn nil, err",
  "\t\t}",
  "\t\treturn &evt, nil",
  "\tcase kind == EventTypeStateUpdate:",
  "\t\tvar evt StateUpdateEvent",
  "\t\tif err := json.Unmarshal(raw, &evt); err != nil {",
  "\t\t\treturn nil, err",
  "\t\t}",
  "\t\treturn &evt, nil",
  "\tcase kind == EventTypeCustom:",
  "\t\tvar evt CustomEvent",
  "\t\tif err := json.Unmarshal(raw, &evt); err != nil {",
  "\t\t\treturn nil, err",
  "\t\t}",
  "\t\treturn &evt, nil",
  "",
].join("\n");

// 命令 A = 契约绑定场景（TestP4G4_，-race -v）；命令 B = 切片 1–22 的既有判据（P0/P1 流式与运行判据）。
const CMD_A = { label: "bound TestP4G4_", args: ["test", "./pkg/hno/run", "./pkg/hno/agent", "-run", "TestP4G4_", "-count=1", "-race", "-v"] };
const CMD_B = { label: "prior TestP1|TestP0", args: ["test", "./pkg/hno/run", "./pkg/hno/agent", "-run", "TestP1|TestP0", "-count=1", "-race", "-v"] };

const MUTANTS = [
  {
    id: "m1",
    desc: "精确匹配块整体摘掉（退回 contains + GenericRunEvent 兜底；node_started 等落 Generic、node_completed 被吃进 RunCompleted）",
    edits: [{ file: eventsJsonGo, anchor: EXACT_BLOCK, repl: "" }],
  },
  {
    id: "m2",
    desc: "次序回退（精确块挪回 contains 归一化之后；node_completed 被 Contains(completed) 抢先吃掉）",
    edits: [
      { file: eventsJsonGo, anchor: EXACT_BLOCK, repl: "" },
      { file: eventsJsonGo, anchor: "\tdefault:\n\t\tvar evt GenericRunEvent", repl: `${EXACT_BLOCK}\tdefault:\n\t\tvar evt GenericRunEvent` },
    ],
  },
  {
    id: "m3",
    desc: "MarshalJSON 空 kind 兜底摘掉（六类全部改为原样吐 eventType；字面量构造的事件 wire 上是哑的）",
    edits: [
      { file: streamEventsGo, anchor: "Event:     canonicalEventType(e.eventBase.eventType, EventTypeNodeStarted),", repl: "Event:     e.eventBase.eventType," },
      { file: streamEventsGo, anchor: "Event:     canonicalEventType(e.eventBase.eventType, EventTypeNodeCompleted),", repl: "Event:     e.eventBase.eventType," },
      { file: streamEventsGo, anchor: "Event:     canonicalEventType(e.eventBase.eventType, EventTypeTaskError),", repl: "Event:     e.eventBase.eventType," },
      { file: streamEventsGo, anchor: "Event:     canonicalEventType(e.eventBase.eventType, EventTypeCheckpoint),", repl: "Event:     e.eventBase.eventType," },
      { file: streamEventsGo, anchor: "Event:     canonicalEventType(e.eventBase.eventType, EventTypeStateUpdate),", repl: "Event:     e.eventBase.eventType," },
      { file: streamEventsGo, anchor: "Event:     canonicalEventType(e.eventBase.eventType, EventTypeCustom),", repl: "Event:     e.eventBase.eventType," },
    ],
  },
  {
    id: "m4",
    desc: "RunStreamMode 的模式校验整段失效（永不触发 → 其余模式静默降级为 messages 流；不 fail-closed）",
    edits: [{ file: agentGo, anchor: "\t\tif mode != run.StreamMessages {", repl: "\t\tif mode != run.StreamMessages && len(fmt.Sprint(mode)) == 0 {" }],
  },
  {
    id: "m5",
    desc: "RunStream 不再等价于 messages 模式（包装改传 StreamCustom → fail-closed 错误）",
    edits: [{ file: streamGo, anchor: "return a.RunStreamMode(ctx, input, run.StreamMessages)", repl: "return a.RunStreamMode(ctx, input, run.StreamCustom)" }],
  },
  {
    id: "m6",
    desc: "构造函数漏盖种类（六个 New* 不再写 eventType；wire 形状靠序列化侧兜底救回，D2 判红）",
    edits: [
      { file: streamEventsGo, anchor: "eventBase: eventBase{eventType: EventTypeNodeStarted, timestamp: time.Now().UTC()}", repl: "eventBase: eventBase{timestamp: time.Now().UTC()}" },
      { file: streamEventsGo, anchor: "eventBase: eventBase{eventType: EventTypeNodeCompleted, timestamp: time.Now().UTC()}", repl: "eventBase: eventBase{timestamp: time.Now().UTC()}" },
      { file: streamEventsGo, anchor: "eventBase: eventBase{eventType: EventTypeTaskError, timestamp: time.Now().UTC()}", repl: "eventBase: eventBase{timestamp: time.Now().UTC()}" },
      { file: streamEventsGo, anchor: "eventBase: eventBase{eventType: EventTypeCheckpoint, timestamp: time.Now().UTC()}", repl: "eventBase: eventBase{timestamp: time.Now().UTC()}" },
      { file: streamEventsGo, anchor: "eventBase: eventBase{eventType: EventTypeStateUpdate, timestamp: time.Now().UTC()}", repl: "eventBase: eventBase{timestamp: time.Now().UTC()}" },
      { file: streamEventsGo, anchor: "eventBase: eventBase{eventType: EventTypeCustom, timestamp: time.Now().UTC()}", repl: "eventBase: eventBase{timestamp: time.Now().UTC()}" },
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
