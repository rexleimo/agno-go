#!/usr/bin/env node
// 切片 34（P4 G4 剩余五模式生产者）的变异矩阵执行体：契约 completionCriteria 3 的牙齿在这里复跑。
//
// 为什么矩阵是交付物：本片的新族判据（逐回合发射、Debug 并集门、Custom 写入口 fail-closed、
// 载荷键形判别）都是「不做什么/只做什么」型事实——绿灯本身不证明门存在，只有把门摘掉仍判红
// 才算牙齿。m-ledger-order（emit 内记账与上通道换序）是契约登记的等价变异：全部夹具都在
// 排空通道之后才读账本，换序不可分辨——如实登记，不充当牙齿。
//
// 用法：
//   node scripts/mutation/p4g4-producers.mjs --check        # 只预检锚唯一性 + 接手行声明完整
//   node scripts/mutation/p4g4-producers.mjs                # 对照跑 + 全部 12 条变异
//   node scripts/mutation/p4g4-producers.mjs m-table m-degrade   # 只跑指定几条
//   P4G4P_MUT_WORK=/tmp/p4g4pmut/work node scripts/mutation/p4g4-producers.mjs
//
// 纪律（与切片 31 执行体相同）：注入不落在仓库工作树——每条变异先把仓库源码整份复制到
// P4G4P_MUT_WORK，再在副本里替换精确串锚，用 `go -C <副本>` 跑契约绑定的同一条场景命令，
// 跑完整份重拷 ⇒ 工作树字节全程不可被本脚本改动。杀红只认 `--- FAIL` 判据行；构建失败/
// 崩溃单独记录，绝不记成牙齿；声明为杀手的变异没杀掉 ⇒ 退出码 1；等价变异反而杀红 ⇒ 同样退出码 1。
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync, mkdirSync, cpSync, rmSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = process.env.P4G4P_MUT_REPO || "/Users/rex/codes/agno-go";
const work = process.env.P4G4P_MUT_WORK || "/tmp/p4g4pmut/work";
const RAW_DIR = process.env.P4G4P_MUT_RAW || "/tmp/p4g4p-matrix";

const PRODUCERS = "pkg/hno/agent/stream_producers.go";
const STREAM = "pkg/hno/agent/stream.go";

// 契约绑定的场景命令（逐字同一串，只是在副本里跑）。
const BOUND = ["test", "./pkg/hno/agent", "-run", "TestP4G4P_", "-count=1", "-race", "-v"];

const ROWS = {
  D1: "TestP4G4P_FiveModesAreWiredAndStartStreams",
  D2: "TestP4G4P_UnknownModesStillFailClosed",
  D3: "TestP4G4P_UpdatesEmitPerTurnDelta",
  D4: "TestP4G4P_ValuesEmitRunningStatePerTurn",
  D5: "TestP4G4P_CheckpointsEmitPerTurnSnapshot",
  D6: "TestP4G4P_DebugIsTheCheckpointsTasksUnion",
  D7: "TestP4G4P_CustomWriterWritesFromToolHandlers",
  D8: "TestP4G4P_MessagesAndTasksBehaviorUnchanged",
};

const A = {
  tableRow: "\trun.StreamValues:      true,\n",
  degrade:
    '\t\tif !wiredStreamModes[mode] {\n\t\t\treturn nil, fmt.Errorf("agent: stream mode %d has no wired producer: %w", int(mode), run.ErrUnsupportedStreamMode)\n\t\t}',
  updatesGate:
    '\tif e.enabled(run.StreamUpdates) {\n\t\tpatch := map[string]any{"turn": e.turn, "delta": resp.Content}',
  turnWire:
    "\t\tmessageCount := func() int { return len(a.Memory.GetMessages(a.UserID)) }\n\t\tr, err := a.newKernel(ctx, tools, currentInstructions, state, invoker, emitter.turnObserver(ctx, messageCount), emitter.streamExecutor(a))",
  contentEmit:
    "\tevt := run.NewRunContentEvent(e.runID, e.agentID, string(types.RoleAssistant), content, e.sequence)\n\te.sequence++\n\treturn e.emit(ctx, evt)",
  debugUnion:
    "\tif e.modes[run.StreamDebug] {\n\t\treturn mode == run.StreamCheckpoints || mode == run.StreamTasks\n\t}\n\treturn false",
  debugBranchOnly:
    "\tif e.modes[run.StreamDebug] {\n\t\treturn mode == run.StreamCheckpoints || mode == run.StreamTasks\n\t}",
  customInstall:
    "\tif e.emitter.enabled(run.StreamCustom) {\n\t\tctx = withCustomWriter(ctx, e.emitter)\n\t}",
  executorGate:
    "\tif !e.enabled(run.StreamTasks) && !e.enabled(run.StreamCustom) {\n\t\treturn nil\n\t}\n\treturn &taskToolExecutor{agent: a, emitter: e}",
  customEmit: "\treturn e.emit(ctx, run.NewCustomEvent(e.runID, subtype, data))",
  valuesEmit:
    '\tif e.enabled(run.StreamValues) {\n\t\tif err := e.emit(ctx, run.NewStateUpdateEvent(e.runID, "", e.snapshot(messageCount))); err != nil {\n\t\t\treturn err\n\t\t}\n\t}',
  counters: "\te.turn++\n\te.contentSoFar += resp.Content",
  label: '\t\tlabel := fmt.Sprintf("turn-%d", e.turn)',
  emitBody:
    "\te.output.appendEvent(evt)\n\n\tselect {\n\tcase e.eventsCh <- evt:\n\t\treturn nil\n\tcase <-ctx.Done():\n\t\treturn ctx.Err()\n\t}",
};

const MUTANTS = [
  {
    id: "m-table",
    name: "模式表漏加 StreamValues（解禁迁移倒退）",
    edits: [{ file: PRODUCERS, anchor: A.tableRow, repl: "" }],
    expectKill: ["D1", "D4", "D5"],
    expectGreen: ["D2", "D3", "D6", "D7", "D8"],
    note: "表漏一行 ⇒ 该模式回到 fail-closed：D1 的 Values 子用例、D4/D5 的 Values 依赖行判红；Updates/Debug/Custom 各行不沾光",
  },
  {
    id: "m-degrade",
    name: "归一化静默跳过未知模式（fail-closed 机制拆除）",
    edits: [{ file: PRODUCERS, anchor: A.degrade, repl: "\t\tif !wiredStreamModes[mode] {\n\t\t\tcontinue\n\t\t}" }],
    expectKill: ["D2"],
    expectGreen: ["D1", "D3", "D4", "D5", "D6", "D7", "D8"],
    note: "未知序数被静默吞掉 ⇒ D2 的三判据（errors.Is/result==nil/零调用）与「stream mode 99」文案形状一起落空；fmt 仍被 turnCompleted 使用，无需动 import",
  },
  {
    id: "m-turn-gate",
    name: "逐回合发射摘掉 Updates 族门（if true）",
    edits: [
      { file: PRODUCERS, anchor: A.updatesGate, repl: '\tif true {\n\t\tpatch := map[string]any{"turn": e.turn, "delta": resp.Content}' },
    ],
    expectKill: ["D4", "D5", "D6", "D7", "D8"],
    expectGreen: ["D1", "D2", "D3"],
    note: "增量泄漏进一切运行：Values/Checkpoints-only 的绝对序列（D4/D5）、Debug-only（D6）、Custom-only 与 Messages 负向（D7）、旧两族零漂移（D8）逐个点名；Updates-only（D3）本来就该有这两条，如实登记为不杀手",
  },
  {
    id: "m-per-chunk",
    name: "逐回合发射挪进分块循环（每分块一条，turnObserver 摘除）",
    edits: [
      { file: STREAM, anchor: A.turnWire, repl: "\t\tr, err := a.newKernel(ctx, tools, currentInstructions, state, invoker, nil, emitter.streamExecutor(a))" },
      {
        file: PRODUCERS,
        anchor: A.contentEmit,
        repl: "\tevt := run.NewRunContentEvent(e.runID, e.agentID, string(types.RoleAssistant), content, e.sequence)\n\te.sequence++\n\tif err := e.emit(ctx, evt); err != nil {\n\t\treturn err\n\t}\n\treturn e.turnCompleted(ctx, &types.ModelResponse{Content: content}, 0)",
      },
    ],
    expectKill: ["D3", "D4", "D5", "D6"],
    expectGreen: ["D1", "D2", "D7", "D8"],
    note: "发射点挪进 streamOnce 的分块循环：回合 1（无分块）从此没有事件、回合 2 每分块一条 ⇒ D3 的 delta/turn 序、D4 的 messages 计数、D5 的计数与 label、D6 的 checkpoint 缺席全部判红；Custom-only 与 Tasks-only 的 content 门关着，D7/D8 如实登记为不杀手",
  },
  {
    id: "m-debug-union",
    name: "摘掉 Debug 并集分支（Debug 退化为空选择）",
    edits: [{ file: PRODUCERS, anchor: A.debugUnion, repl: "\treturn false" }],
    expectKill: ["D6"],
    expectGreen: ["D1", "D2", "D3", "D4", "D5", "D7", "D8"],
    note: "Debug-only 的通道既无 checkpoint 也无任务事件 ⇒ D6 的并集序列判红；D1 的 Debug 子用例只断「流启动了」，如实登记为不杀手（族内容由 D6 管）",
  },
  {
    id: "m-debug-messages",
    name: "Debug 暗含 Messages（并集被扩写）",
    edits: [{ file: PRODUCERS, anchor: A.debugBranchOnly, repl: "\tif e.modes[run.StreamDebug] {\n\t\treturn true\n\t}" }],
    expectKill: ["D6"],
    expectGreen: ["D1", "D2", "D3", "D4", "D5", "D7", "D8"],
    note: "Debug-only 泄漏 run_content ⇒ D6 的泄漏检查与绝对序列判红；其余行不选 Debug，如实登记为不杀手",
  },
  {
    id: "m-custom-gate",
    name: "writer 恒安装 + 包装器恒挂载（未选 Custom 也有写入口）",
    edits: [
      { file: PRODUCERS, anchor: A.customInstall, repl: "\tctx = withCustomWriter(ctx, e.emitter)" },
      { file: PRODUCERS, anchor: A.executorGate, repl: "\treturn &taskToolExecutor{agent: a, emitter: e}" },
    ],
    expectKill: ["D7"],
    expectGreen: ["D1", "D2", "D3", "D4", "D5", "D6", "D8"],
    note: "写入口缺席是两道构造（executor 选择 + 安装门）合起来的性质：Messages-only 本来不挂包装器，只摘安装门杀不到它——必须两处同摘，Messages-only 运行里 handler 的写入才从「报错」退成「writer 在、族门关 ⇒ 静默吞」，D7 的 fail-closed 负向判红。旧两族在族门内移下不漏事件（D8 如实登记为不杀手）",
  },
  {
    id: "m-custom-payload",
    name: "subtype/data 两字段互调",
    edits: [{ file: PRODUCERS, anchor: A.customEmit, repl: '\treturn e.emit(ctx, run.NewCustomEvent(e.runID, fmt.Sprintf("%v", data), subtype))' }],
    expectKill: ["D7"],
    expectGreen: ["D1", "D2", "D3", "D4", "D5", "D6", "D8"],
    note: "线上的 Subtype 变成了数据文本 ⇒ D7 的逐字段断言判红",
  },
  {
    id: "m-values-keys",
    name: "Values 载荷退化成 Updates 键形（判别键混同）",
    edits: [
      {
        file: PRODUCERS,
        anchor: A.valuesEmit,
        repl: '\tif e.enabled(run.StreamValues) {\n\t\tif err := e.emit(ctx, run.NewStateUpdateEvent(e.runID, "", map[string]any{"turn": e.turn, "delta": resp.Content})); err != nil {\n\t\t\treturn err\n\t\t}\n\t}',
      },
    ],
    expectKill: ["D4", "D5"],
    expectGreen: ["D1", "D2", "D3", "D6", "D7", "D8"],
    note: "values-only 的载荷 JSON 变 {delta,…} ⇒ D4 判红；三族并选里 state 补丁与 checkpoint 快照不再同值 ⇒ D5 连带判红；D6 只看 kinds 与 label，如实登记为不杀手",
  },
  {
    id: "m-turn-count",
    name: "回合序数不递增（两条都叫 turn-1）",
    edits: [{ file: PRODUCERS, anchor: A.counters, repl: "\te.contentSoFar += resp.Content" }],
    expectKill: ["D3", "D4", "D5", "D6"],
    expectGreen: ["D1", "D2", "D7", "D8"],
    note: "两个回合的 patch/label 全部停在 turn-1 ⇒ D3/D4 的逐回合载荷、D5 的 label、D6 的 checkpoint label 逐个判红",
  },
  {
    id: "m-label",
    name: "checkpoint label 改成裸序数",
    edits: [{ file: PRODUCERS, anchor: A.label, repl: "\t\tlabel := fmt.Sprint(e.turn)" }],
    expectKill: ["D5", "D6"],
    expectGreen: ["D1", "D2", "D3", "D4", "D7", "D8"],
    note: "label 不再是 turn-N 形状 ⇒ D5 的 label 断言与 D6 的 checkpoint label 断言判红；D3/D4 不看 checkpoint，如实登记为不杀手",
  },
  {
    id: "m-ledger-order",
    name: "emit 内记账与上通道换序（契约登记的等价变异）",
    equivalent: true,
    edits: [
      {
        file: PRODUCERS,
        anchor: A.emitBody,
        repl: "\tselect {\n\tcase e.eventsCh <- evt:\n\t\te.output.appendEvent(evt)\n\t\treturn nil\n\tcase <-ctx.Done():\n\t\treturn ctx.Err()\n\t}",
      },
    ],
    expectKill: [],
    expectGreen: Object.keys(ROWS),
    note: "全部夹具在排空通道之后才读账本，账面序与通道序的差异在观测上不可分辨 ⇒ 不充当牙齿；若这里反而杀红，等价登记需重写（并发消费方能分辨的形状不在本片夹具内）",
  },
];

function sha(buf) {
  const header = Buffer.from(`blob ${buf.length}\0`);
  return createHash("sha1").update(Buffer.concat([header, buf])).digest("hex");
}

function syncCopy() {
  rmSync(work, { recursive: true, force: true });
  mkdirSync(work, { recursive: true });
  cpSync(repo, work, {
    recursive: true,
    filter: (src) => !/(^|\/)(\.git|website|node_modules|bin|dist|\.rex-harness)(\/|$)/.test(src.replace(`${repo}/`, "")),
  });
}

function verifyBaseline() {
  syncCopy();
  for (const file of [PRODUCERS, STREAM]) {
    const a = readFileSync(`${repo}/${file}`);
    const b = readFileSync(`${work}/${file}`);
    if (sha(a) !== sha(b)) throw new Error(`副本 ${file} 与仓库字节不同`);
  }
}

function apply(mutant) {
  for (const edit of mutant.edits) {
    const path = `${work}/${edit.file}`;
    const text = readFileSync(path, "utf8");
    const hits = text.split(edit.anchor).length - 1;
    if (hits !== 1) throw new Error(`${mutant.id} 在 ${edit.file} 的锚出现 ${hits} 次，需恰好 1 次：${JSON.stringify(edit.anchor.slice(0, 60))}`);
    writeFileSync(path, text.replace(edit.anchor, edit.repl));
  }
}

function run() {
  let out = "";
  let code = 0;
  try {
    out = execFileSync("go", ["-C", work, ...BOUND], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"], timeout: 420000 });
  } catch (err) {
    code = typeof err.status === "number" ? err.status : -1;
    out = `${err.stdout ?? ""}${err.stderr ?? ""}`;
  }
  const fails = [...new Set([...out.matchAll(/^\s*--- FAIL: (\S+)/gm)].map((m) => m[1]))];
  const passes = [...out.matchAll(/^\s*--- PASS: (\S+)/gm)].map((m) => m[1]);
  const buildFailed = /\[setup failed\]|\[build failed\]|^# /m.test(out) && fails.length === 0 && passes.length === 0;
  const panic = /panic: |DATA RACE/.test(out);
  return { code, fails, buildFailed, panic, out };
}

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
      if (children.some((k) => named.has(ROWS[k]))) continue;
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
  for (const m of picked) {
    for (const r of [...m.expectKill, ...m.expectGreen]) if (!ROWS[r]) throw new Error(`${m.id} 声明了未知接手行 ${r}`);
    const declared = new Set([...m.expectKill, ...m.expectGreen]);
    const uncovered = Object.keys(ROWS).filter((r) => !declared.has(r));
    if (uncovered.length) throw new Error(`${m.id} 的接手行没覆盖全部判据：${uncovered.join(",")}`);
    if (m.equivalent && m.expectKill.length) throw new Error(`${m.id} 是等价变异却声明了杀手行`);
  }
  if (checkOnly) {
    for (const m of picked) {
      verifyBaseline();
      apply(m);
    }
    console.log(`preflight ok: ${picked.length} 条变异，锚在仓库当前字节上唯一命中，接手行声明完整`);
    return;
  }

  mkdirSync(RAW_DIR, { recursive: true });
  const stamp = new Date().toISOString().replace(/[-:]/g, "").replace(/\..+$/, "");
  verifyBaseline();
  console.log(`baseline ${PRODUCERS}=${sha(readFileSync(`${repo}/${PRODUCERS}`))} ${STREAM}=${sha(readFileSync(`${repo}/${STREAM}`))}`);

  const control = run();
  writeFileSync(`${RAW_DIR}/control.${stamp}.txt`, control.out);
  if (control.code !== 0) {
    console.error(`CONTROL_RED 未注入变异的副本判红（exit=${control.code}），牙齿讨论无意义`);
    process.exit(2);
  }
  console.log(`control exit=${control.code} raw=${RAW_DIR}/control.${stamp}.txt`);

  let violations = 0;
  for (const m of picked) {
    verifyBaseline();
    apply(m);
    const r = run();
    const rawFile = `${RAW_DIR}/${m.id}.${stamp}.txt`;
    writeFileSync(rawFile, r.out);
    verifyBaseline(); // 整份重拷 ⇒ 仓库字节从不被触碰

    if (r.buildFailed) {
      console.error(`${m.id} BUILD_FAILED 变异没编译过，本次没有牙齿可言（不计杀红） raw=${rawFile}`);
      violations++;
      continue;
    }
    if (r.panic) {
      console.error(`${m.id} CRASH_OR_RACE 崩溃或 DATA RACE，不计牙齿 raw=${rawFile}`);
      violations++;
      continue;
    }
    if (r.code !== 0 && r.fails.length === 0) {
      console.error(`${m.id} NO_ASSERTION_RED 场景命令非零退出但没有判红行 ⇒ 环境红，不计牙齿 raw=${rawFile}`);
      violations++;
      continue;
    }
    if (r.code === 0 && m.expectKill.length) {
      console.error(`${m.id} NO_TEETH(空转) 注入后仍全绿 raw=${rawFile}`);
      violations++;
      continue;
    }

    const { hit, unmapped } = attribute(r.fails);
    const missed = m.expectKill.filter((row) => !hit.has(row));
    const brokeGreen = m.expectGreen.filter((row) => hit.has(row));
    const undeclared = [...hit].filter((row) => !m.expectKill.includes(row));
    let verdict = "ok";
    if (missed.length) verdict = `NO_TEETH(空转) missing=${missed.join(",")}`;
    else if (brokeGreen.length) verdict = `ATTRIBUTION_LEAK leaked=${brokeGreen.join(",")}`;
    else if (undeclared.length) verdict = `UNDECLARED_KILL rows=${undeclared.join(",")}`;
    else if (m.equivalent && hit.size) verdict = `EQUIVALENCE_REGISTRATION_DRIFT killed=${[...hit].join(",")}`;
    if (verdict !== "ok") violations++;

    console.log(
      [
        m.id,
        `exit=${r.code}`,
        `接手行=${[...hit].sort().join(",") || "-"}`,
        `声明杀=${m.expectKill.join(",") || "(等价变异，不充当牙齿)"}`,
        verdict,
        `raw=${rawFile}`,
        unmapped.length ? `UNMAPPED=${unmapped.join(",")}` : "",
      ].join(" | "),
    );
    console.log(`      ${m.name} — ${m.note}`);
  }

  verifyBaseline();
  const drift = [PRODUCERS, STREAM].filter((f) => sha(readFileSync(`${repo}/${f}`)) !== sha(readFileSync(`${work}/${f}`)));
  console.log(`finalCheck 副本与仓库同字节=${drift.length === 0} 工作树从未被注入=${drift.length === 0}`);
  console.log(`矩阵条数=${picked.length} 违规=${violations}`);
  if (drift.length) process.exit(2);
  if (violations) process.exit(1);
}

main();
