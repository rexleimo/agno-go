#!/usr/bin/env node
// 切片 26（母约 §6 G5：动态扇出 Send + AddJoinSend）的变异矩阵执行体。
// 延续 p3g6-durability.mjs 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `-v` Test 名）。
//
// 用法：
//   node scripts/mutation/p3g5-send.mjs            # 跑全部变异
//   node scripts/mutation/p3g5-send.mjs m1 m5      # 只跑指定的几条
//   node scripts/mutation/p3g5-send.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const schedGo = "pkg/hno/graph/scheduler.go";
const graphGo = "pkg/hno/graph/graph.go";

// 命令 A = 契约绑定场景（TestP3G5_，-race -v）；命令 B = 既有先序族（P1G/P1R13–19，-race -v）。
const CMD_A = { label: "bound TestP3G5_", args: ["test", "./pkg/hno/graph", "-run", "TestP3G5_", "-count=1", "-race", "-v"] };
const CMD_B = { label: "prior TestP1G_|TestP1R1[3-9]", args: ["test", "./pkg/hno/graph", "-run", "TestP1G_|TestP1R1[3-9]", "-count=1", "-race", "-v"] };

const MUTANTS = [
  {
    id: "m1",
    desc: "生产者 Sender 断言摘掉：每次激活都走 Node.Run（SenderFunc 的 Run 视图丢弃 sends）——扇出根本不发生",
    edits: [
      {
        file: schedGo,
        anchor: "\tif snd, ok := s.plan.nodes[act.name].(Sender); ok {\n\t\treturn snd.SendRun(ctx, act.in)\n\t}\n",
        repl: "",
      },
    ],
  },
  {
    id: "m2",
    desc: "屏障凑齐判据摘掉：在途计数归零不再触发 joinSendReady（递减照旧、永不激活 target）",
    edits: [
      {
        file: schedGo,
        anchor: "\t\tif s.sendPending[item.via] == 0 {\n\t\t\ts.pending = append(s.pending, s.joinSendReady()...)\n\t\t}",
        repl: "",
      },
    ],
  },
  {
    id: "m3",
    desc: "AddJoinSend 的可达性贡献摘掉：edgeJoinSend 的 source 不再被视为可达并传播——map-reduce 形状整体被判不可达",
    edits: [
      {
        file: graphGo,
        anchor: "\t// AddJoinSend 的 source 只被运行期 Send 指名，构建期无法反驳「会有 Sender 派发给它」\n\t//（与「可达不看谓词」的同一乐观先例），因此视为可达并从它继续传播。\n\treached := map[string]bool{g.entry: true}\n\tqueue := []string{g.entry}\n\tfor _, e := range g.edges {\n\t\tif e.kind == edgeJoinSend && !reached[e.from] {\n\t\t\treached[e.from] = true\n\t\t\tqueue = append(queue, e.from)\n\t\t}\n\t}\n",
        repl: "\treached := map[string]bool{g.entry: true}\n\tqueue := []string{g.entry}\n",
      },
    ],
  },
  {
    id: "m4",
    desc: "Send 未注册检查摘掉并静默丢弃（契约 forbiddenShortcuts 的形状）：整体校验循环移除、派发循环跳过未注册目标——不报错、不部分派发，悄悄少派",
    edits: [
      {
        file: schedGo,
        anchor: "\t\tfor _, snd := range item.sends {\n\t\t\tif _, ok := s.plan.nodes[snd.Node]; !ok {\n\t\t\t\treturn fmt.Errorf(\"graph: node %q sent to unregistered node %q\", item.name, snd.Node)\n\t\t\t}\n\t\t}\n",
        repl: "",
      },
      {
        file: schedGo,
        anchor: "\t\tfor _, snd := range item.sends {\n\t\t\tidx := len(s.sendCollected[snd.Node])",
        repl: "\t\tfor _, snd := range item.sends {\n\t\t\tif _, ok := s.plan.nodes[snd.Node]; !ok {\n\t\t\t\tcontinue\n\t\t\t}\n\t\t\tidx := len(s.sendCollected[snd.Node])",
      },
    ],
  },
  {
    id: "m5",
    desc: "零派发改判激活（契约 forbiddenShortcuts 的形状）：凑齐判据从「计数归零」放宽为「非 Send 完成或计数归零都激活」——零派发/普通完成也以空聚合激活 target 一次",
    edits: [
      {
        file: schedGo,
        anchor: "\tif item.via != \"\" {\n\t\ts.sendCollected[item.via][item.idx] = item.out\n\t\ts.sendPending[item.via]--\n\t\tif s.sendPending[item.via] == 0 {\n\t\t\ts.pending = append(s.pending, s.joinSendReady()...)\n\t\t}\n\t}",
        repl: "\tif item.via != \"\" {\n\t\ts.sendCollected[item.via][item.idx] = item.out\n\t\ts.sendPending[item.via]--\n\t}\n\tif item.via == \"\" || s.sendPending[item.via] == 0 {\n\t\ts.pending = append(s.pending, s.joinSendReady()...)\n\t}",
      },
    ],
  },
  {
    id: "m6",
    desc: "多波守卫复位摘掉：计数由 0 转正时不再复位 sendFired——第二波凑齐后 target 因「本波已激活」被跳过（跨波丢弃）",
    edits: [
      {
        file: schedGo,
        anchor: "\t\t\tif s.sendPending[snd.Node] == 1 {\n\t\t\t\tfor _, t := range s.sendTargetsOf[snd.Node] {\n\t\t\t\t\ts.sendFired[t] = false\n\t\t\t\t}\n\t\t\t}\n",
        repl: "",
      },
    ],
  },
  // 预期「全绿」的等价变异：登记观察上限，不算杀红。
  {
    id: "e1",
    desc: "（预期全绿）joinSendReady 的延迟复位改为就地复位——只有「多个 target 共享同一 source 且同波就绪」时可观察，D1–D9 无该形状（观察上限）",
    edits: [
      { file: schedGo, anchor: "\tvar drained []string\n", repl: "" },
      { file: schedGo, anchor: "\t\tready = append(ready, activation{name: target, in: agg})\n\t\tdrained = append(drained, target)\n", repl: "\t\tready = append(ready, activation{name: target, in: agg})\n\t\tfor src := range s.sendSourcesOf[target] {\n\t\t\ts.sendCollected[src] = nil\n\t\t}\n" },
      { file: schedGo, anchor: "\t// 复位放在全部就绪 target 的聚合都取走之后：同一次归零触发的多个 target\n\t// 看到的是同一波输出，不能被先处理的 target 清空。\n\tfor _, target := range drained {\n\t\tfor src := range s.sendSourcesOf[target] {\n\t\t\ts.sendCollected[src] = nil\n\t\t}\n\t}\n", repl: "" },
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
