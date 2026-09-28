#!/usr/bin/env node
// 切片 31（P4 G4 StreamTasks 生产者）的变异矩阵执行体：契约 completionCriteria 2 的牙齿在这里复跑。
//
// 为什么矩阵是交付物：本片新增的三条判据（按族门控、handler error 精确分类、截断调用静默）
// 都是「不做什么」型事实——绿灯本身不证明门控存在，只有把门控摘掉仍判红才算牙齿。
//
// 用法：
//   node scripts/mutation/p4s31-tasks-producer.mjs --check      # 只预检锚唯一性 + 接手行声明完整
//   node scripts/mutation/p4s31-tasks-producer.mjs              # 对照跑 + 全部 9 条变异
//   node scripts/mutation/p4s31-tasks-producer.mjs m-gate m-dup # 只跑指定几条
//   S31_MUT_WORK=/tmp/s31mut/work node scripts/mutation/p4s31-tasks-producer.mjs
//
// 与切片 30 执行体唯一不同的纪律：**注入不落在仓库工作树**。每条变异先把仓库源码
// （排除 .git/website/node_modules/bin/dist/.rex-harness）整份复制到 S31_MUT_WORK，再在该副本里
// 替换精确串锚，用 `go -C <副本>` 跑契约绑定的同一条场景命令，跑完整份重拷 ⇒ 工作树字节
// 全程不可被本脚本改动（R17 那套「改完再复原 + 哈希复核」在同仓兄弟代理并发的在途里是多余风险）。
//
// 其余两条纪律照旧写在代码里：
//   1) 杀红只认 `--- FAIL` 判据行；构建失败/崩溃单独记录，绝不记成牙齿。
//   2) 声明为杀手的变异没杀掉它声明的那一行 ⇒ 退出码 1；登记为等价变异的那条若反而杀红 ⇒
//      同样退出码 1，因为它推翻了契约的等价登记。
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync, mkdirSync, cpSync, rmSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = process.env.S31_MUT_REPO || "/Users/rex/codes/agno-go";
const work = process.env.S31_MUT_WORK || "/tmp/s31mut/work";
const RAW_DIR = process.env.S31_MUT_RAW || "/tmp/s31-matrix";

const PRODUCERS = "pkg/hno/agent/stream_producers.go";
const STREAM = "pkg/hno/agent/stream.go";

// 契约绑定的场景命令（逐字同一串，只是在副本里跑）。
const BOUND = ["test", "./pkg/hno/agent", "-run", "TestP4S31_", "-count=1", "-race", "-v"];

const ROWS = {
  D1: "TestP4S31_TasksModeIsWiredAndStartsTheStream",
  D2: "TestP4S31_UnwiredModesStillFailClosed",
  D3: "TestP4S31_TasksOnlyStreamCarriesTaskEventsOnly",
  D4: "TestP4S31_TaskEventPayloadsAndWireShape",
  "D5/err": "TestP4S31_TaskFailureClassifiedByTheHandlerError/handler_error_becomes_task_error",
  "D5/lookalike":
    "TestP4S31_TaskFailureClassifiedByTheHandlerError/success_whose_text_looks_like_the_failure_stays_completed",
  D6: "TestP4S31_ConcurrentBatchEmitsInDeclarationOrder",
  D7: "TestP4S31_MessagesPathsUnchangedByTaskWiring",
  "D8/swap": "TestP4S31_ModeSelectionIsAnOrderIndependentUnion/swapping_the_mode_arguments_changes_nothing",
  "D8/dup": "TestP4S31_ModeSelectionIsAnOrderIndependentUnion/repeating_a_mode_does_not_re-emit",
  "D8/none": "TestP4S31_ModeSelectionIsAnOrderIndependentUnion/no_modes_equals_messages",
  "D8/debug": "TestP4S31_ModeSelectionIsAnOrderIndependentUnion/debug_is_not_accepted_as_a_tasks_alias",
  D9: "TestP4S31_CallsTruncatedByTheLimitStaySilent",
};

const A = {
  gate: "func (e *streamEmitter) enabled(mode run.StreamMode) bool {\n\treturn e.modes[mode]\n}",
  degrade:
    '\t\tif !wiredStreamModes[mode] {\n\t\t\treturn nil, fmt.Errorf("agent: stream mode %d has no wired producer: %w", int(mode), run.ErrUnsupportedStreamMode)\n\t\t}',
  classify:
    "\tif res.err != nil {\n\t\treturn e.emit(ctx, run.NewTaskErrorEvent(e.runID, res.node, res.message))\n\t}",
  startedBody:
    "\treturn e.emit(ctx, run.NewNodeStartedEvent(e.runID, call.Function.Name, 1, call.Function.Arguments))",
  completedBody:
    "\treturn e.emit(ctx, run.NewNodeCompletedEvent(e.runID, res.node, 1, res.message))",
  executeHead:
    "\tfor _, call := range calls {\n\t\tif err := e.emitter.taskStarted(ctx, call); err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t}\n\n\toutcomes, results := e.agent.runToolBatch(ctx, calls)",
  executeFull:
    "\tfor _, call := range calls {\n\t\tif err := e.emitter.taskStarted(ctx, call); err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t}\n\n\toutcomes, results := e.agent.runToolBatch(ctx, calls)\n\n\tfor _, res := range results {\n\t\tif err := e.emitter.taskFinished(ctx, res); err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t}\n\treturn outcomes, nil",
  contentEmit:
    "\tevt := run.NewRunContentEvent(e.runID, e.agentID, string(types.RoleAssistant), content, e.sequence)\n\te.sequence++\n\treturn e.emit(ctx, evt)",
  normalizeSet:
    "\tset := make(map[run.StreamMode]bool, len(modes))\n\tfor _, mode := range modes {\n\t\tif !wiredStreamModes[mode] {\n\t\t\treturn nil, fmt.Errorf(\"agent: stream mode %d has no wired producer: %w\", int(mode), run.ErrUnsupportedStreamMode)\n\t\t}\n\t\tset[mode] = true\n\t}\n\treturn set, nil",
  normalizeSig: "func normalizeStreamModes(modes []run.StreamMode) (map[run.StreamMode]bool, error) {",
  emitterField: "\tmodes    map[run.StreamMode]bool\n",
  invoker:
    "\t\tinvoker := runner.TurnInvokerFunc(func(turnCtx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {\n\t\t\treturn a.streamOnce(turnCtx, req, emitter)\n\t\t})",
  streamSig: "func (a *Agent) runStreamMessages(ctx context.Context, input string, modes map[run.StreamMode]bool)",
  imports: 'import (\n\t"context"\n\t"fmt"\n',
};

const importStrings = { anchor: A.imports, repl: 'import (\n\t"context"\n\t"fmt"\n\t"strings"\n' };

const MUTANTS = [
  {
    id: "m-gate",
    name: "摘掉按族门控（enabled 恒真）",
    edits: [
      { file: PRODUCERS, anchor: A.gate, repl: "func (e *streamEmitter) enabled(mode run.StreamMode) bool {\n\treturn true\n}" },
    ],
    expectKill: ["D3", "D7"],
    alsoKills: ["D4", "D5/err", "D5/lookalike", "D6", "D8/dup", "D8/none"],
    expectGreen: ["D1", "D2", "D8/swap", "D8/debug", "D9"],
    note: "契约 completionCriteria 2 的 m-gate：门一摘，Tasks-only 流漏出 run_content（D3）且 Messages 流被塞进任务事件（D7）；另六行按绝对 kind 序列同时判红，如实登记为额外红",
  },
  {
    id: "m-degrade",
    name: "摘掉 fail-closed 校验（未接线模式静默跳过）",
    edits: [
      { file: PRODUCERS, anchor: A.degrade, repl: "\t\tif !wiredStreamModes[mode] {\n\t\t\tcontinue\n\t\t}" },
      { file: PRODUCERS, anchor: A.imports, repl: 'import (\n\t"context"\n' },
    ],
    expectKill: ["D2"],
    alsoKills: ["D8/debug"],
    expectGreen: ["D1", "D3", "D4", "D5/err", "D5/lookalike", "D6", "D7", "D8/swap", "D8/dup", "D8/none", "D9"],
    note: "m-degrade：降级成零事件流而不是报错，D2 的十条 fail-closed 判据与 D8「Debug 不当 Tasks 别名」一起落空（Errorf 撤掉后 fmt 必须同步撤，否则编译不过）",
  },
  {
    id: "m-sniff",
    name: "用 Contains 式文本嗅探分类",
    edits: [
      { file: PRODUCERS, anchor: A.classify, repl: '\tif strings.Contains(res.message, "error") {\n\t\treturn e.emit(ctx, run.NewTaskErrorEvent(e.runID, res.node, res.message))\n\t}' },
      { file: PRODUCERS, ...importStrings },
    ],
    expectKill: ["D5/lookalike"],
    alsoKills: ["D6"],
    expectGreen: ["D1", "D2", "D3", "D4", "D5/err", "D7", "D8/swap", "D8/dup", "D8/none", "D8/debug", "D9"],
    note: "m-sniff：lookalike 工具成功返回的那句话里含 error ⇒ 被误判成 task_error，正是 D5 第二段建的对照；D6 的批次序列同时点名",
  },
  {
    id: "m-sniff-prefix",
    name: "HasPrefix 式嗅探（契约登记的等价变异）",
    equivalent: true,
    edits: [
      {
        file: PRODUCERS,
        anchor: A.classify,
        repl: '\tif strings.HasPrefix(res.message, "tool execution error:") {\n\t\treturn e.emit(ctx, run.NewTaskErrorEvent(e.runID, res.node, res.message))\n\t}',
      },
      { file: PRODUCERS, ...importStrings },
    ],
    expectKill: [],
    expectGreen: Object.keys(ROWS),
    note: "M2 登记的等价变异：成功结果文本是 JSON 串（带引号），前缀嗅探与 err 判定在这套夹具上同解 ⇒ 不充当牙齿；若这里反而杀红，等价登记需重写",
  },
  {
    id: "m-order",
    name: "把 started 挪到批次汇合之后（与结束事件同批发射）",
    edits: [
      {
        file: PRODUCERS,
        anchor: A.executeFull,
        repl: "\toutcomes, results := e.agent.runToolBatch(ctx, calls)\n\n\tfor i, res := range results {\n\t\tif err := e.emitter.taskStarted(ctx, calls[i]); err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\tif err := e.emitter.taskFinished(ctx, res); err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t}\n\treturn outcomes, nil",
      },
    ],
    expectKill: ["D3", "D6"],
    alsoKills: ["D4", "D8/swap", "D8/dup"],
    expectGreen: ["D1", "D2", "D5/err", "D5/lookalike", "D7", "D8/none", "D8/debug", "D9"],
    note: "m-order：结果序顶掉声明序，D6 的八个位置逐个点名 node 列表；D3 的绝对 kind 序列同时落空",
  },
  {
    id: "m-skip",
    name: "对被截断的调用也发声（started 改挂在模型回合上）",
    edits: [
      { file: PRODUCERS, anchor: A.executeHead, repl: "\toutcomes, results := e.agent.runToolBatch(ctx, calls)" },
      {
        file: STREAM,
        anchor: A.invoker,
        repl:
          "\t\tinvoker := runner.TurnInvokerFunc(func(turnCtx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {\n\t\t\tresp, err := a.streamOnce(turnCtx, req, emitter)\n\t\t\tif err == nil && resp != nil && emitter.enabled(run.StreamTasks) {\n\t\t\t\tfor _, call := range resp.ToolCalls {\n\t\t\t\t\tif emitErr := emitter.taskStarted(turnCtx, call); emitErr != nil {\n\t\t\t\t\t\treturn nil, emitErr\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t\treturn resp, err\n\t\t})",
      },
    ],
    expectKill: ["D9"],
    expectGreen: ["D1", "D2", "D3", "D4", "D5/err", "D5/lookalike", "D6", "D7", "D8/swap", "D8/dup", "D8/none", "D8/debug"],
    note: "m-skip：runner 的 ToolCallLimit 判定发生在执行器之前，挂到回合上就绕过了截断 ⇒ D9 的「boom 不发声」判红。单批次场景（D1/D3/D6）在两种挂点下同序，故如实登记为不杀手",
  },
  {
    id: "m-payload-id",
    name: "node 字段填成工具调用 ID",
    edits: [{ file: PRODUCERS, anchor: A.startedBody, repl: "\treturn e.emit(ctx, run.NewNodeStartedEvent(e.runID, call.ID, 1, call.Function.Arguments))" }],
    expectKill: ["D4", "D6"],
    expectGreen: ["D1", "D2", "D3", "D5/err", "D5/lookalike", "D7", "D8/swap", "D8/dup", "D8/none", "D8/debug", "D9"],
    note: "m-payload 第一段：started 的 node 不再是函数名",
  },
  {
    id: "m-payload-agent",
    name: "node 字段填成 agentID",
    edits: [{ file: PRODUCERS, anchor: A.completedBody, repl: "\treturn e.emit(ctx, run.NewNodeCompletedEvent(e.runID, e.agentID, 1, res.message))" }],
    expectKill: ["D4", "D6"],
    expectGreen: ["D1", "D2", "D3", "D5/err", "D5/lookalike", "D7", "D8/swap", "D8/dup", "D8/none", "D8/debug", "D9"],
    note: "m-payload 第二段：completed 的 node 被写成运行主体（图侧语义里 node 是被调节点名，不是谁在跑）",
  },
  {
    id: "m-dup",
    name: "模式集合不过滤重复（选择退化成列表）",
    edits: [
      { file: PRODUCERS, anchor: A.normalizeSig, repl: "func normalizeStreamModes(modes []run.StreamMode) ([]run.StreamMode, error) {" },
      { file: PRODUCERS, anchor: A.normalizeSet, repl: "\tset := make([]run.StreamMode, 0, len(modes))\n\tfor _, mode := range modes {\n\t\tif !wiredStreamModes[mode] {\n\t\t\treturn nil, fmt.Errorf(\"agent: stream mode %d has no wired producer: %w\", int(mode), run.ErrUnsupportedStreamMode)\n\t\t}\n\t\tset = append(set, mode)\n\t}\n\treturn set, nil" },
      { file: PRODUCERS, anchor: A.emitterField, repl: "\tmodes    []run.StreamMode\n" },
      { file: PRODUCERS, anchor: A.gate, repl: "func (e *streamEmitter) enabled(mode run.StreamMode) bool {\n\tfor _, m := range e.modes {\n\t\tif m == mode {\n\t\t\treturn true\n\t\t}\n\t}\n\treturn false\n}" },
      {
        file: PRODUCERS,
        anchor: A.contentEmit,
        repl:
          "\tvar lastErr error\n\tfor _, m := range e.modes {\n\t\tif m != run.StreamMessages {\n\t\t\tcontinue\n\t\t}\n\t\tevt := run.NewRunContentEvent(e.runID, e.agentID, string(types.RoleAssistant), content, e.sequence)\n\t\te.sequence++\n\t\tlastErr = e.emit(ctx, evt)\n\t}\n\treturn lastErr",
      },
      { file: STREAM, anchor: A.streamSig, repl: "func (a *Agent) runStreamMessages(ctx context.Context, input string, modes []run.StreamMode)" },
    ],
    expectKill: ["D8/dup"],
    expectGreen: ["D1", "D2", "D3", "D4", "D5/err", "D5/lookalike", "D6", "D7", "D8/swap", "D8/none", "D8/debug", "D9"],
    note: "m-dup：重复模式各发一份 ⇒ D8 第二段「重复给同一模式不会重发」判红；次序行 D8/swap 与默认行 D8/none 不受影响",
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
  if (mutant.edits.some((e) => e.repl.includes("strings."))) {
    const path = `${work}/${PRODUCERS}`;
    const text = readFileSync(path, "utf8");
    if (!text.includes('"strings"')) writeFileSync(path, text.replace(A.imports, importStrings.repl));
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
    for (const r of [...m.expectKill, ...(m.alsoKills ?? []), ...m.expectGreen]) if (!ROWS[r]) throw new Error(`${m.id} 声明了未知接手行 ${r}`);
    const declared = new Set([...m.expectKill, ...(m.alsoKills ?? []), ...m.expectGreen]);
    const uncovered = Object.keys(ROWS).filter((r) => !declared.has(r));
    if (uncovered.length) throw new Error(`${m.id} 的接手行没覆盖全部判据：${uncovered.join(",")}`);
    const overlap = m.expectKill.filter((r) => (m.alsoKills ?? []).includes(r));
    if (overlap.length) throw new Error(`${m.id} 把 ${overlap.join(",")} 同时声明为主杀手与额外红`);
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
    const alsoMissed = (m.alsoKills ?? []).filter((row) => !hit.has(row));
    const brokeGreen = m.expectGreen.filter((row) => hit.has(row));
    const undeclared = [...hit].filter((row) => !m.expectKill.includes(row) && !(m.alsoKills ?? []).includes(row));
    let verdict = "ok";
    if (missed.length) verdict = `NO_TEETH(空转) missing=${missed.join(",")}`;
    else if (alsoMissed.length) verdict = `ALSO_KILL_DRIFT 登记的额外红没再现 missing=${alsoMissed.join(",")}`;
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
        `额外红=${(m.alsoKills ?? []).join(",") || "-"}`,
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
