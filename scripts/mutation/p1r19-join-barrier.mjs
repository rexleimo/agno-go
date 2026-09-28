#!/usr/bin/env node
// 切片 19（票面 §4 B2：并行扇出 + Join 汇聚屏障）的变异矩阵执行体。
//
// 为什么放在仓库里：前两轮审查（S17-STD-5、S18-STD-6）记下同一件事 —— 判定脚本落在 /tmp
// 时，别人无法复现「这行断言有牙齿」这个结论。
//
// 用法：
//   node scripts/mutation/p1r19-join-barrier.mjs            # 跑全部变异
//   node scripts/mutation/p1r19-join-barrier.mjs m2 m4      # 只跑指定的几条
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
const schedGo = "pkg/hno/graph/scheduler.go";

// 契约绑定的场景命令（D1–D7），以及承载被退役的 R1 格与端点次序的邻近命令。
const CMD_A = { label: "bound TestP1R19_", args: ["test", "./pkg/hno/graph", "-run", "TestP1R19_", "-count=1", "-race", "-v"] };
const CMD_B = { label: "neighbor TestP1G_", args: ["test", "./pkg/hno/graph", "-run", "TestP1G_", "-count=1", "-race", "-v"] };

const MUTANTS = [
  {
    id: "m1",
    desc: "屏障凑齐判据整段失效（每个前驱完成都激活一次）",
    edits: [{ file: schedGo, anchor: "if len(collected) < len(s.waits[target]) {", repl: "if false {" }],
  },
  {
    id: "m2",
    desc: "激活时交出「最后一个前驱的输出」而不是聚合 map",
    edits: [{ file: schedGo, anchor: "ready = append(ready, activation{name: target, in: collected})", repl: "ready = append(ready, activation{name: target, in: out})" }],
  },
  {
    id: "m3",
    desc: "聚合输入按汇聚目标名键控，而不是按前驱名键控",
    edits: [{ file: schedGo, anchor: "collected[name] = out", repl: "collected[target] = out" }],
  },
  {
    id: "m4",
    desc: "前驱数检查读了一份空声明表（等于该检查整段不存在）",
    edits: [{ file: graphGo, anchor: "waits, _ := joinBarriers(g.edges)", repl: "waits, _ := joinBarriers(nil)" }],
  },
  {
    id: "m5",
    desc: "阈值放宽为「至少 1 个前驱」",
    edits: [{ file: graphGo, anchor: "if len(preds) < 2 {", repl: "if len(preds) < 1 {" }],
  },
  {
    id: "m6",
    desc: "票面 :77 反例的另一半：把 Join 当成无条件边在 successors 里直接激活（屏障逻辑仍在）",
    edits: [
      {
        file: schedGo,
        anchor: "\t\tcase edgeDefault:\n\t\t\tfallback = append(fallback, activation{name: e.to, in: out})",
        repl: "\t\tcase edgeJoin:\n\t\t\tconcrete = append(concrete, activation{name: e.to, in: out})\n\t\tcase edgeDefault:\n\t\t\tfallback = append(fallback, activation{name: e.to, in: out})",
      },
    ],
  },
  {
    id: "m7",
    desc: "屏障算完了但从不交给调度器（joinReady 结果被丢弃）",
    edits: [
      {
        file: schedGo,
        anchor: "\ts.pending = append(s.pending, s.joinReady(item.name, item.out)...)",
        repl: "\t_ = s.joinReady(item.name, item.out)",
      },
    ],
  },
  {
    id: "m8",
    desc: "汇聚目标自己的出边被忽略（join 的输出不再往下游走）",
    edits: [
      {
        file: schedGo,
        anchor: "\t\tif e.from != name {\n\t\t\tcontinue\n\t\t}",
        repl: "\t\tif e.from != name || s.waits[name] != nil {\n\t\t\tcontinue\n\t\t}",
      },
    ],
  },
  {
    id: "m9",
    desc: "次序变异：前驱数检查挪到逐条边的端点检查之前",
    edits: [
      {
        file: graphGo,
        anchor: "\tif err := g.rejectUnderfedJoinTargets(); err != nil {\n\t\treturn err\n\t}\n",
        repl: "",
      },
      {
        file: graphGo,
        anchor: "\tfor _, e := range g.edges {\n\t\tif _, ok := g.nodes[e.from]; !ok {",
        repl: "\tif err := g.rejectUnderfedJoinTargets(); err != nil {\n\t\treturn err\n\t}\n\tfor _, e := range g.edges {\n\t\tif _, ok := g.nodes[e.from]; !ok {",
      },
    ],
  },
  {
    id: "m10",
    desc: "把 Join 收回「不可路由」白名单（等于本片的路由没落地）",
    edits: [
      {
        file: graphGo,
        anchor: "\tcase edgeUnconditional, edgeConditional, edgeDefault, edgeJoin:",
        repl: "\tcase edgeUnconditional, edgeConditional, edgeDefault:",
      },
    ],
  },
  {
    id: "m11",
    desc: "步数安全阀的错误不再包装 ErrStepLimitExceeded 哨兵（%w 改成 %v）",
    edits: [
      {
        file: schedGo,
        anchor: "step limit %d reached before the graph converged: %w",
        repl: "step limit %d reached before the graph converged: %v",
      },
    ],
  },
  // 以下三条预期「全绿」：它们不是漏网的缺陷形态，而是本片判据观察不到的性质，
  // 用于把观察上限钉成实测而不是口头声明。
  {
    id: "e1",
    desc: "（预期全绿）不足 2 前驱的多个目标同时存在时不再按字典序点名",
    edits: [{ file: graphGo, anchor: "\tsort.Strings(thin)", repl: "\t_ = thin[0]" }],
  },
  {
    id: "e2",
    desc: "（预期全绿）前驱名不去重，按声明条数算屏障",
    edits: [
      {
        file: graphGo,
        anchor: "\t\tif waits[e.to][e.from] {\n\t\t\tcontinue\n\t\t}\n",
        repl: "",
      },
    ],
  },
  {
    id: "e3",
    desc: "（预期全绿）反向索引按 map 迭代顺序累加，放弃声明顺序",
    edits: [
      {
        file: graphGo,
        anchor: "\t\tfeeds[e.from] = append(feeds[e.from], e.to)",
        repl: "\t\tfeeds[e.from] = append([]string{e.to}, feeds[e.from]...)",
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
