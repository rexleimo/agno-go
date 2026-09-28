#!/usr/bin/env node
// 切片 21（票面 §4 B6 第 2 项：重复节点名构建期拒绝）的变异矩阵执行体。
//
// 延续 p1r19-join-barrier.mjs 的理由（S17-STD-5 / S18-STD-6）：判定脚本落在 /tmp 时，
// 别人无法复现「这行断言有牙齿」这个结论。
//
// 用法：
//   node scripts/mutation/p1r21-duplicate-node.mjs            # 跑全部变异
//   node scripts/mutation/p1r21-duplicate-node.mjs m2 m4      # 只跑指定的几条
//   node scripts/mutation/p1r21-duplicate-node.mjs --check    # 只预检锚唯一性
//
// 每条变异：用精确串锚替换当前字节（锚必须出现恰好一次，否则判为脚本失效而不是判红）
// → 跑两条命令 → 从 `-v` 输出里取 `--- FAIL:` 的 Test 名 → 立刻恢复原文 →
// 用 git hash-object 自检恢复后的字节与运行前逐字节相同。
//
// 判红只认 `-v` 里的 Test 名；退出码单独记录（构建失败同样 exit 1，不看名字就会误记为杀红）。
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const graphGo = "pkg/hno/graph/graph.go";

// 契约绑定的场景命令（D1–D5），以及承载既有构建期判据的邻近命令。
const CMD_A = { label: "bound TestP1R21_", args: ["test", "./pkg/hno/graph", "-run", "TestP1R21_", "-count=1", "-race", "-v"] };
const CMD_B = { label: "neighbor TestP1G_", args: ["test", "./pkg/hno/graph", "-run", "TestP1G_", "-count=1", "-race", "-v"] };

const MUTANTS = [
  {
    id: "m1",
    desc: "重复名检查整段失效（Validate 末位改回直接返回环检查）",
    edits: [{ file: graphGo, anchor: "\treturn g.rejectDuplicateNodeNames()", repl: "\treturn g.rejectUnconditionalCycles()" }],
  },
  {
    id: "m2",
    desc: "错误文案不再点名被重复的名字（名字槽位换成空串；可定位性判据失效）。不放成无指令的常量文案：go test 内置的 vet printf 检查会先拦下它，判红就不来自测试了",
    edits: [{ file: graphGo, anchor: 'more than once", g.dupNames[0]', repl: 'more than once", ""' }],
  },
  {
    id: "m3",
    desc: "记账退化成「每次注册都记」（合法图也被拒，D3/D4 的钉）",
    edits: [{ file: graphGo, anchor: "\tif _, exists := g.nodes[n.Name()]; exists {", repl: "\tif true {" }],
  },
  {
    id: "m4",
    desc: "次序变异：重复名检查挪到入口/端点检查之前（观察 D1–D5 是否钉次序）",
    edits: [
      { file: graphGo, anchor: "\treturn g.rejectDuplicateNodeNames()", repl: "\treturn g.rejectUnconditionalCycles()" },
      {
        file: graphGo,
        anchor: "\tif _, ok := g.nodes[g.entry]; !ok {",
        repl: "\tif err := g.rejectDuplicateNodeNames(); err != nil {\n\t\treturn err\n\t}\n\tif _, ok := g.nodes[g.entry]; !ok {",
      },
    ],
  },
];

function sha(buf) {
  // git blob 哈希：sha1("blob <len>\0" + content)，与 `git hash-object --no-filters` 同值。
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

  // 预检：每条变异的锚都要在当前字节上唯一命中，且一条变异只改一个文件（写回逻辑按此假设）。
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
  // 末行自检：全部恢复后仍要与基线逐字节相同。
  const finalDrift = files.filter((f) => sha(readFileSync(`${repo}/${f}`)) !== sha(baseline.get(f)));
  console.log(`finalCheck identical=${finalDrift.length === 0} files=${files.join(",")}`);
  if (finalDrift.length) process.exitCode = 2;
}

main();
