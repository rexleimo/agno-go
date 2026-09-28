#!/usr/bin/env node
// 切片 22（母约 §4 G3：节点策略四合一）的变异矩阵执行体。
// 延续 p1r19/p1r21 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `-v` Test 名）。
//
// 用法：
//   node scripts/mutation/p2g3-policies.mjs            # 跑全部变异
//   node scripts/mutation/p2g3-policies.mjs m2 m4      # 只跑指定的几条
//   node scripts/mutation/p2g3-policies.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const graphGo = "pkg/hno/graph/graph.go";
const schedGo = "pkg/hno/graph/scheduler.go";
const policyGo = "pkg/hno/graph/policy.go";

// 命令 A = 契约绑定场景（TestP2G3_，-race -v）；命令 B = 切片 1–21 的全部既有判据。
const CMD_A = { label: "bound TestP2G3_", args: ["test", "./pkg/hno/graph", "-run", "TestP2G3_", "-count=1", "-race", "-v"] };
const CMD_B = { label: "prior TestP1*", args: ["test", "./pkg/hno/graph", "-run", "TestP1", "-count=1", "-race", "-v"] };

const MUTANTS = [
  {
    id: "m1",
    desc: "重试循环整段失效（首次失败即 break；用 true || 保持 max/should 被引用，避免未使用变量掩盖判红）",
    edits: [{ file: schedGo, anchor: "if err == nil || attempt >= max || !pol.retry.should(err) {", repl: "if true || attempt >= max || !pol.retry.should(err) {" }],
  },
  {
    id: "m2",
    desc: "ShouldRetry 判据取反（说不重试的反而重试）",
    edits: [{ file: schedGo, anchor: "!pol.retry.should(err)", repl: "pol.retry.should(err)" }],
  },
  {
    id: "m3",
    desc: "期限钳制整段失效（Timeout 永远不挂）",
    edits: [{ file: policyGo, anchor: "\tif c == nil || c.Timeout <= 0 {\n\t\treturn 0, false\n\t}", repl: "\treturn 0, false" }],
  },
  {
    id: "m4",
    desc: "PerAttempt 退化为单次期限（每次尝试不再新建）",
    edits: [
      {
        file: schedGo,
        anchor: "\tif pol.timeout != nil && pol.timeout.PerAttempt {\n\t\tif d, ok := pol.timeout.deadline(); ok {\n\t\t\tvar cancel context.CancelFunc\n\t\t\tctx, cancel = context.WithTimeout(base, d)\n\t\t\tdefer cancel()\n\t\t}\n\t}",
        repl: "",
      },
    ],
  },
  {
    id: "m5",
    desc: "缓存查询整段失效（永远真跑）",
    edits: [{ file: schedGo, anchor: "\tif pol.cache != nil && pol.cache.Store != nil {\n\t\tif v, ok, gerr := pol.cache.Store.GetAny(s.ctx, pol.cache.keyFor(act.in)); gerr == nil && ok {\n\t\t\treturn v, nil, 1\n\t\t}\n\t}", repl: "" }],
  },
  {
    id: "m6",
    desc: "失败也写缓存（违反 fail-closed）",
    edits: [{ file: schedGo, anchor: "if err == nil && pol.cache != nil && pol.cache.Store != nil {", repl: "if pol.cache != nil && pol.cache.Store != nil {" }],
  },
  {
    id: "m7",
    desc: "完成事件的发射点摘掉（trace 失聪）",
    edits: [
      {
        file: schedGo,
        anchor: "\ts.plan.policies[item.name].trace.emit(NodeEvent{\n\t\tNode: item.name, Attempt: item.attempt, Out: item.out, In: item.in,\n\t})\n",
        repl: "",
      },
    ],
  },
  {
    id: "m8",
    desc: "脱敏分支摘掉（载荷裸奔进事件）",
    edits: [
      {
        file: policyGo,
        anchor: "\tif c.RedactIn {\n\t\tev.In = nil\n\t}\n\tif c.RedactOut {\n\t\tev.Out = nil\n\t}",
        repl: "",
      },
    ],
  },
  // 预期「全绿」的等价变异：登记观察上限，不算杀红。
  {
    id: "e1",
    desc: "（预期全绿）失败路径的 trace 发射摘掉 —— D1–D14 没有一行观察失败事件（D12 只钉完成事件），该发射是 fail-closed 方向的未钉语义，登记为观察上限",
    edits: [
      {
        file: schedGo,
        anchor: "\t\t\t\ts.plan.policies[item.name].trace.emit(NodeEvent{\n\t\t\t\t\tNode: item.name, Attempt: item.attempt, Err: item.err, In: item.in,\n\t\t\t\t})\n",
        repl: "",
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
