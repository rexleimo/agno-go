#!/usr/bin/env node
// 切片 30（R17 取消归一化加固）的变异矩阵执行体：票面 §13.3 的牙齿在这里逐条复跑。
//
// 延续 p1r21-duplicate-node.mjs / p1r19-join-barrier.mjs 的理由（S17-STD-5 / S18-STD-6）：
// 判定脚本落在 /tmp 时，别人无法复现「这行断言有牙齿」这个结论。本片的「红」不来自新行为
// （引擎语义未变，加固形在现行字节上全绿），而来自**回归注入**，所以矩阵就是交付物本身。
//
// 用法：
//   node scripts/mutation/p1r17-cancel-window.mjs --check        # 只预检锚唯一性 + 声明完整性
//   node scripts/mutation/p1r17-cancel-window.mjs                # 跑全部 10 条变异
//   node scripts/mutation/p1r17-cancel-window.mjs m1 m9          # 只跑指定的几条
//
// 每条变异：用精确串锚替换 scheduler.go 当前字节（锚必须出现恰好一次，否则判为脚本失效而不是判红）
// → 跑契约绑定的场景命令 → 从 `-v` 输出里取判红的行（含子例行，接手行按行归因）
// → 立刻恢复原文 → 用 git hash-object 复核恢复后的字节与运行前逐字节相同（基线 dfd6c1ba…）。
//
// 两条纪律写在代码里而不是写在注释里：
//   1) 杀红只认 `t.Error*` 的判据行；构建失败单独记录，绝不记成牙齿。
//   2) 声明为 solo 杀手的变异若没杀掉它声明的那一行 ⇒ 退出码 1（空转绿不可交付）；
//      登记为等价变异的 m11 若反而杀红了 ⇒ 同样退出码 1，因为它推翻了契约的等价登记。
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const schedulerGo = "pkg/hno/graph/scheduler.go";
// 契约 D8：每一红的原始输出必须有名有姓有落点，收口报告引用文件而不是引用记忆。
const RAW_DIR = process.env.R17_MATRIX_RAW || "/tmp/r17-matrix";

// 契约绑定的场景命令（本片只动测试文件，注入的是 scheduler.go 的临时态）。
const BOUND = { label: "bound TestP1R17_", args: ["test", "./pkg/hno/graph", "-run", "TestP1R17_", "-count=1", "-race", "-v"] };

// 行标识 → `-v` 里的判红名。子例单独建条目，D5 的两段与 D6 的三条才有可归因的牙齿。
const ROWS = {
  D1hard: "TestP1R17_CancellationWinsOverStepLimitDuringDispatch",
  D2hard: "TestP1R17_CancellationIsNotDisguisedAsNodeBusinessErrorAcrossRounds",
  D3: "TestP1R17_AlreadyCancelledContextIsReportedAsCancellation",
  "D3/普通图": "TestP1R17_AlreadyCancelledContextIsReportedAsCancellation/普通图",
  "D3/预算一条的宽扇出": "TestP1R17_AlreadyCancelledContextIsReportedAsCancellation/预算一条的宽扇出",
  "D3/上限一个槽": "TestP1R17_AlreadyCancelledContextIsReportedAsCancellation/上限一个槽",
  D4: "TestP1R17_BlockedNodeExitsAllNormalizeToCancellation",
  "D4/交回 ctx.Err()": "TestP1R17_BlockedNodeExitsAllNormalizeToCancellation/交回_ctx.Err()",
  "D4/交回业务哨兵": "TestP1R17_BlockedNodeExitsAllNormalizeToCancellation/交回业务哨兵",
  "D4/交回包装错误": "TestP1R17_BlockedNodeExitsAllNormalizeToCancellation/交回包装了别的哨兵的错误",
  D5a: "TestP1R17_CancellationWhileQueueHasBacklog/一个槽位且有积压",
  D5b: "TestP1R17_CancellationWhileQueueHasBacklog/上限让派发停住时也是取消语义",
  D6biz: "TestP1R17_UncancelledBusinessErrorStillSurfacesVerbatim/未取消时业务错误原样交出",
  D6conv: "TestP1R17_UncancelledBusinessErrorStillSurfacesVerbatim/未取消时正常收敛不受影响",
  D6valve: "TestP1R17_UncancelledBusinessErrorStillSurfacesVerbatim/未取消的步数超限仍报超限",
};

// 三行加固形（本片的加固对象）与三行反向牙齿（D6 族）：solo 杀手归属按契约 redProtocol 声明。
const HARDENED = ["D1hard", "D2hard", "D5b"];

// 注入用的精确串锚（每条都经 --check 验证唯一命中）。
const ANCHOR = {
  guardItemAfterPop: "\t\t\tif err := s.ctx.Err(); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}",
  selectCancelBranch: "\t\tcase <-s.ctx.Done():\n\t\t\treturn nil, s.ctx.Err()",
  dispatchLoopHeadWithGuard:
    "\tfor len(s.pending) > 0 {\n\t\tif err := s.ctx.Err(); err != nil {\n\t\t\treturn err\n\t\t}",
  dispatchLoopHead: "\tfor len(s.pending) > 0 {",
  stepLimitError:
    '\t\t\treturn fmt.Errorf("graph: step limit %d reached before the graph converged: %w", s.stepLimit, ErrStepLimitExceeded)',
  businessErrorReturn: "\t\t\t\treturn nil, item.err",
  consumeHead: "func (s *scheduler) consume() (*Result, error) {\n\tdefer close(s.done)",
};

const CANCEL_AFTER_BUDGET =
  ANCHOR.stepLimitError + "\n\t\tif err := s.ctx.Err(); err != nil {\n\t\t\treturn err\n\t\t}";

const MUTANTS = [
  {
    id: "m1",
    name: "τ 删掉派发侧取消判据（③）",
    edits: [{ anchor: ANCHOR.dispatchLoopHeadWithGuard, repl: ANCHOR.dispatchLoopHead }],
    expectKill: ["D1hard"],
    expectGreen: HARDENED.filter((r) => r !== "D1hard"),
    note: "契约 D1 的 solo 杀手；停靠点放行后消费者只能撞上安全阀 ⇒ 取消被报成超限",
  },
  {
    id: "m2",
    name: "ο 把预算判据抢在取消判据之前",
    edits: [
      { anchor: ANCHOR.dispatchLoopHeadWithGuard, repl: ANCHOR.dispatchLoopHead },
      { anchor: ANCHOR.stepLimitError, repl: CANCEL_AFTER_BUDGET },
    ],
    expectKill: ["D1hard"],
    expectGreen: ["D2hard", "D5b"],
    note: "加固前这一条整族判绿（票面把它登记为等价变异）；加固形给出位置证明后它成为 D1 的杀手，属升级登记",
  },
  {
    id: "m3",
    name: "λ/L_snap 让 ② 读 consume 入口快照",
    edits: [
      { anchor: ANCHOR.consumeHead, repl: ANCHOR.consumeHead + "\n\tsnap := s.ctx.Err()" },
      {
        anchor: ANCHOR.guardItemAfterPop,
        repl: "\t\t\tif err := snap; err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}",
      },
    ],
    expectKill: ["D2hard"],
    expectGreen: ["D1hard", "D5b"],
    note: "契约 D2 的 solo 杀手：快照恒为 nil ⇒ 手上那条业务错误被原样交出，取消被伪装成业务错误",
  },
  {
    id: "m4",
    name: "κ 两条守卫全删（②+③）",
    edits: [
      { anchor: ANCHOR.dispatchLoopHeadWithGuard, repl: ANCHOR.dispatchLoopHead },
      { anchor: ANCHOR.guardItemAfterPop, repl: "" },
    ],
    expectKill: ["D1hard", "D2hard"],
    expectGreen: ["D5b"],
    note: "复合对照行；① 还在，所以 D5 第二段仍由 select 的取消分支回答",
  },
  {
    id: "m5",
    name: "删掉 select 的取消分支（①）",
    edits: [{ anchor: ANCHOR.selectCancelBranch, repl: "" }],
    expectKill: ["D5b"],
    expectGreen: ["D1hard", "D2hard"],
    note: "契约 D5 第二段的 solo 杀手：全部后继闸住（零投递）⇒ 取消后只剩挂死，由「未在有界阀内返回」接住",
  },
  {
    id: "m6",
    name: "μ 一切结论先交 ctx.Err()",
    edits: [{ anchor: ANCHOR.businessErrorReturn, repl: "\t\t\t\treturn nil, s.ctx.Err()" }],
    expectKill: ["D6biz"],
    expectGreen: HARDENED,
    note: "反向牙齿：只应由原始 D6 族接手，加固行必须判绿（否则「任何时候都先返回取消」的实现能蒙过去）",
  },
  {
    id: "m7",
    name: "ξ 未取消时把安全阀报成取消",
    edits: [{ anchor: ANCHOR.stepLimitError, repl: "\t\t\treturn context.Canceled" }],
    expectKill: ["D6valve"],
    expectGreen: HARDENED,
    note: "反向牙齿：同 m6，接手行是 D6 的「未取消的步数超限仍报超限」",
  },
  {
    id: "m8",
    name: "取消报成功（① 交出 Result 而不是错误）",
    edits: [
      { anchor: ANCHOR.selectCancelBranch, repl: "\t\tcase <-s.ctx.Done():\n\t\t\treturn s.result, nil" },
    ],
    // 契约归属如实照抄：`HardD2 ← {…, 取消报成功(m8)}`、`HardD5b ← {…, 取消报成功(m8)}`，
    // 而 `HardD1 ← {τ(m1), ο(m2), κ(m4)}` 里**没有** m8 —— 加固形 D1 的取消由派发侧守卫 ③ 回答，
    // 消费者根本走不到 select，所以 m8 对 D1 不可达。矩阵把这一点写成 expectGreen，
    // 是为了让「D1 的牙齿落在 ③ 上」这句归属可复跑，而不是靠收口报告叙述。
    expectKill: ["D2hard", "D5b"],
    expectGreen: ["D1hard"],
    note: "同时杀到 D4 三个出口与 D5 第一段属预期额外红：那几行本来就在断言「取消必须交出错误」",
  },
  {
    id: "m9",
    name: "整条删除 ②（票面 §13.3 第 1 条的不可达形态）",
    edits: [{ anchor: ANCHOR.guardItemAfterPop, repl: "" }],
    expectKill: ["D2hard"],
    expectGreen: ["D1hard", "D5b"],
    note: "本片闭合 R17-GAP-1 的证据：加固形杀得动、原始观测式夹具在同一执行里走不到差别点",
  },
  {
    id: "m11",
    name: "σ 把取消判据提到循环之外（取消之后继续放出积压）",
    edits: [
      {
        anchor: ANCHOR.dispatchLoopHeadWithGuard,
        repl:
          "\tif err := s.ctx.Err(); err != nil {\n\t\t\treturn err\n\t\t}\n\t" + ANCHOR.dispatchLoopHead.slice(1),
      },
    ],
    expectKill: [],
    expectGreen: Object.keys(ROWS),
    equivalent: true,
    note: "契约登记的等价变异（M2 实测 4/4 全绿）：不许拿它充当牙齿；若这里反而杀红 ⇒ 等价登记需重写",
  },
];

function sha(buf) {
  const header = Buffer.from(`blob ${buf.length}\0`);
  return createHash("sha1").update(Buffer.concat([header, buf])).digest("hex");
}

function apply(text, edits) {
  let out = text;
  for (const e of edits) {
    const parts = out.split(e.anchor);
    if (parts.length !== 2) {
      throw new Error(`锚出现 ${parts.length - 1} 次，需恰好 1 次：${JSON.stringify(e.anchor.slice(0, 56))}`);
    }
    out = parts[0] + e.repl + parts[1];
  }
  return out;
}

function preflight(picked) {
  const text = readFileSync(`${repo}/${schedulerGo}`, "utf8");
  for (const m of picked) {
    for (const e of m.edits) {
      const hits = text.split(e.anchor).length - 1;
      if (hits !== 1) throw new Error(`${m.id} 的锚出现 ${hits} 次，需恰好 1 次：${JSON.stringify(e.anchor.slice(0, 56))}`);
    }
    for (const r of m.expectKill) if (!ROWS[r]) throw new Error(`${m.id} 声明了未知行 ${r}`);
    for (const r of m.expectGreen) if (!ROWS[r]) throw new Error(`${m.id} 声明了未知行 ${r}`);
  }
}

function run(args) {
  let out = "";
  let code = 0;
  try {
    out = execFileSync("go", args, { cwd: repo, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"], timeout: 300000 });
  } catch (err) {
    code = typeof err.status === "number" ? err.status : -1;
    out = `${err.stdout ?? ""}${err.stderr ?? ""}`;
  }
  const fails = [...out.matchAll(/^\s*--- FAIL: (\S+)/gm)].map((m) => m[1]);
  const passes = [...out.matchAll(/^\s*--- PASS: (\S+)/gm)].map((m) => m[1]);
  const buildFailed = /\[setup failed\]|\[build failed\]|^# /m.test(out) && fails.length === 0 && passes.length === 0;
  // 契约 redProtocol：杀红必须是判据行，前提守卫红一律不计入牙齿。
  // 守卫文案是夹具里成文的分型（「本行前提不成立」「先修夹具」「停靠点没建立」…），按它分诊。
  const GUARD_WORDS =
    /本行前提不成立|前提没建立|停靠点没建立|先修夹具|被别的节点占了|槽位没打满|积压没建立|没有后继节点开始运行|没有节点开始运行/;
  const guardRed = out.split("\n").filter((line) => GUARD_WORDS.test(line));
  return { code, fails: [...new Set(fails)].sort(), buildFailed, guardRed, out };
}

// 判红名 → 行 id。Go 在子例失败时同时打印子例行与父例行，所以父例名只有在「该父例下
// 没有任何子例行被点名」时才回退到它的全部子例行——否则会把没失败的那一段也算杀红。
function attribute(fails) {
  const hit = new Set();
  const unmapped = [];
  const named = new Set(fails);
  for (const f of fails) {
    const row = Object.keys(ROWS).find((k) => ROWS[k] === f);
    if (row) hit.add(row);
  }
  for (const f of fails) {
    if (Object.keys(ROWS).some((k) => ROWS[k] === f)) continue;
    const children = Object.keys(ROWS).filter((k) => ROWS[k].startsWith(f + "/"));
    if (children.length) {
      if (children.some((k) => named.has(ROWS[k]))) continue; // 子例已点名，父例只是汇总
      children.forEach((k) => hit.add(k));
    } else unmapped.push(f);
  }
  return { hit, unmapped };
}

function main() {
  const checkOnly = process.argv.includes("--check");
  const wanted = process.argv.slice(2).filter((a) => !a.startsWith("-"));
  const picked = wanted.length ? MUTANTS.filter((m) => wanted.includes(m.id)) : MUTANTS;
  if (wanted.length && wanted.length !== picked.length) {
    throw new Error(`未知变异 id：${wanted.filter((w) => !MUTANTS.some((m) => m.id === w)).join(",")}`);
  }

  preflight(picked);
  if (checkOnly) {
    console.log(`preflight ok: ${picked.length} 条变异，锚唯一命中，接手行声明完整（${picked.map((m) => m.id).join(",")}）`);
    return;
  }

  const baseline = readFileSync(`${repo}/${schedulerGo}`);
  const baselineHash = sha(baseline);
  const stamp = new Date().toISOString().replace(/[-:]/g, "").replace(/\..+$/, "");
  console.log(`scheduler.go baseline git-blob-hash ${baselineHash} (契约锚 dfd6c1ba1c2a9c35bf4ccb613ee986dd80a7c736)`);
  if (baselineHash !== "dfd6c1ba1c2a9c35bf4ccb613ee986dd80a7c736") {
    console.error("REFUSE 现行 scheduler.go 与契约登记字节不同：先回契约核对，不在漂移的基线上谈牙齿");
    process.exit(1);
  }

  let violations = 0;
  for (const m of picked) {
    let r = null;
    let scriptError = null;
    try {
      writeFileSync(`${repo}/${schedulerGo}`, apply(baseline.toString("utf8"), m.edits));
      r = run(BOUND.args);
    } catch (err) {
      scriptError = String(err.message ?? err);
    } finally {
      writeFileSync(`${repo}/${schedulerGo}`, baseline);
    }
    const restored = sha(readFileSync(`${repo}/${schedulerGo}`)) === baselineHash;
    if (!restored) {
      console.error(`${m.id} RESTORE DRIFT 恢复后字节与基线不同 —— 立刻停止，人工核对工作树`);
      process.exit(2);
    }
    if (scriptError) {
      console.error(`${m.id} SCRIPT_ERROR ${scriptError}`);
      violations++;
      continue;
    }
    mkdirSync(RAW_DIR, { recursive: true });
    const rawFile = `${RAW_DIR}/${m.id}.${stamp}.txt`;
    writeFileSync(rawFile, r.out);
    if (r.buildFailed) {
      console.error(`${m.id} BUILD_FAILED 变异没编译过，本次没有牙齿可言（不计杀红） raw=${rawFile}`);
      violations++;
      continue;
    }
    if (r.code !== 0 && r.fails.length === 0) {
      console.error(`${m.id} NO_ASSERTION_RED 场景命令非零退出但没有任何判红行 ⇒ 崩溃/构建/环境红，不计牙齿 raw=${rawFile}`);
      violations++;
      continue;
    }

    const { hit, unmapped } = attribute(r.fails);
    const killed = m.expectKill.filter((row) => hit.has(row));
    const missed = m.expectKill.filter((row) => !hit.has(row));
    const brokeGreen = m.expectGreen.filter((row) => hit.has(row));
    const rows = [...hit].sort().join(",") || "-";
    let verdict = "ok";
    if (r.guardRed.length) verdict = `PREMISE_RED(守卫红，不计牙齿) ${r.guardRed.length} 处`;
    else if (missed.length) verdict = `NO_TEETH(空转) missing=${missed.join(",")}`;
    else if (brokeGreen.length) verdict = `ATTRIBUTION_LEAK leaked=${brokeGreen.join(",")}`;
    else if (m.equivalent && hit.size) verdict = "EQUIVALENCE_REGISTRATION_DRIFT";
    if (verdict !== "ok") {
      violations++;
      if (r.guardRed.length) r.guardRed.slice(0, 3).forEach((line) => console.error(`      guard: ${line.trim()}`));
    }

    console.log(
      [
        m.id,
        `exit=${r.code}`,
        `接手行=${rows}`,
        `声明杀=${m.expectKill.join(",") || "(等价变异，不充当牙齿)"}`,
        `杀到=${killed.join(",") || "-"}`,
        verdict,
        `restored=${restored}`,
        `raw=${rawFile}`,
        unmapped.length ? `UNMAPPED=${unmapped.join(",")}` : "",
      ].join(" | "),
    );
    console.log(`      ${m.name} — ${m.note}`);
  }

  const finalSame = sha(readFileSync(`${repo}/${schedulerGo}`)) === baselineHash;
  console.log(`finalCheck scheduler.go identical=${finalSame}`);
  console.log(`矩阵条数=${picked.length} 违规=${violations}`);
  if (!finalSame) process.exit(2);
  if (violations) process.exit(1);
}

main();
