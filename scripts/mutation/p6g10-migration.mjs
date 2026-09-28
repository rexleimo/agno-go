#!/usr/bin/env node
// P6（G10 OPT-β：workflow 控制流全换图内核）的变异矩阵执行体。
// 延续 p5s28-sidecar.mjs 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `--- FAIL:` 名）。
//
// 用法：
//   node scripts/mutation/p6g10-migration.mjs            # 跑全部变异
//   node scripts/mutation/p6g10-migration.mjs m1 m5      # 只跑指定的几条
//   node scripts/mutation/p6g10-migration.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const executorGo = "pkg/hno/workflow/executor.go";
const compilerGo = "pkg/hno/workflow/graph_compiler.go";

// 命令 A = 契约绑定场景（TestP6G10_，-race，克隆隔离等并发判据靠它杀红）；
// 命令 B = 零改动主判据族（workflow 既有 13 文件全绿）+ 邻近面（agent / session contract）。
// 刻意不含 pkg/hno/store 与 ./...（兄弟片在途领地，派发说明明令禁止）。
const CMD_A = {
  label: "bound TestP6G10_",
  args: ["test", "./pkg/hno/workflow/...", "-run", "TestP6G10_", "-count=1", "-race", "-v"],
};
const CMD_B = {
  label: "workflow-full+neighbors",
  args: ["test", "./pkg/hno/workflow/...", "./pkg/hno/agent/...", "./internal/session/contract/...", "-count=1"],
};

const MUTANTS = [
  {
    id: "m1",
    desc: "编译结果弃用（D1 破）：executeSteps 成功路径退回种子上下文，图内核跑完即丢",
    edits: [
      {
        file: executorGo,
        anchor:
          "\tfinal := lastEC\n\tif car, ok := res.Output().(*carrier); ok && car.ec != nil {\n\t\tfinal = car.ec\n\t}",
        repl: "\tfinal := execCtx",
      },
    ],
  },
  {
    id: "m2",
    desc: "别名隔离摘除（D8/[P6] 破）：并行分支头不再克隆私有 EC，四支并发写同一上下文——只被 -race 杀红",
    edits: [
      {
        file: compilerGo,
        anchor: "return &carrier{ec: cloneBranchEC(in.(*carrier).ec)}, nil",
        repl: "return &carrier{ec: in.(*carrier).ec}, nil",
      },
    ],
  },
  {
    id: "m3",
    desc: "loop 迭代记账漂移（D6 破）：loop_<id>_iterations 多记一次",
    edits: [
      {
        file: compilerGo,
        anchor: 'car.ec.Set(fmt.Sprintf("loop_%s_iterations", l.ID), cur)',
        repl: 'car.ec.Set(fmt.Sprintf("loop_%s_iterations", l.ID), cur+1)',
      },
    ],
  },
  {
    id: "m4",
    desc: "按步事件台账清零（D9 破）：成功路径不再从最终 EC 反查 step_<id>_events",
    edits: [
      {
        file: executorGo,
        anchor: "events:     cp.ledger.collectEvents(final, false),",
        repl: "events:     map[string]run.Events{},",
      },
    ],
  },
  {
    id: "m5",
    desc: "错误归属反转为报失败步（D3/[P4] 破）：`step %s failed` 填失败节点而非最后成功主干步",
    edits: [
      {
        file: compilerGo,
        anchor: 'fmt.Sprintf("step %s failed", lastSpineID), inner)',
        repl: 'fmt.Sprintf("step %s failed", failing), inner)',
      },
    ],
  },
  {
    id: "m6",
    desc: "router 未命中静默兜底（D5/前置(iii) 破）：未命中错误节点改判恒等穿出，借 AddDefault 放行",
    edits: [
      {
        file: compilerGo,
        anchor: 'return nil, fmt.Errorf("router %s: route \'%s\' not found", r.ID, car.route)',
        repl: "return &carrier{ec: car.ec}, nil",
      },
    ],
  },
  {
    id: "m7",
    desc: "并行汇聚取首支（D7 破）：merge 的 Output 改取第一个分支下标（旧语义=最后下标）",
    edits: [
      {
        file: compilerGo,
        anchor:
          "\tif len(results) > 0 && results[len(results)-1] != nil {\n\t\tsrc.Output = results[len(results)-1].Output\n\t}",
        repl: "\tif len(results) > 0 && results[0] != nil {\n\t\tsrc.Output = results[0].Output\n\t}",
      },
    ],
  },
  {
    id: "m8",
    desc: "主干错误包摘除（D3 破）：不再包 types.NewError(UNKNOWN, step … failed)，引擎裸错误直接上交",
    edits: [
      {
        file: compilerGo,
        anchor: 'return types.NewError(types.ErrCodeUnknown, fmt.Sprintf("step %s failed", lastSpineID), inner)',
        repl: "return inner",
      },
    ],
  },
  {
    id: "m9",
    desc: "resume 入口错位（D2 破）：startIdx 被忽略，总是从第 0 步全量编译",
    edits: [
      {
        file: executorGo,
        anchor: "\ttail := steps[startIdx:]",
        repl: "\ttail := steps[0:]",
      },
    ],
  },
  {
    id: "m10",
    desc: "预算回退默认（D11 破）：WithStepLimit 的结构上界不再传入，高迭代 loop 撞默认 1000",
    edits: [
      {
        file: compilerGo,
        anchor: "\tg := graph.New(graph.WithStepLimit(limit + 1)) // +1 终止节点",
        repl: "\t_ = limit\n\tg := graph.New()",
      },
    ],
  },
  {
    id: "m11",
    desc: "取消边界粒度倒退（D10 破）：每个中间节点完成都停调度器，刚完成的主干步不再保证入账",
    edits: [
      {
        file: compilerGo,
        anchor: "\tif spineID != \"\" && watch != nil && stop != nil && watch.Err() != nil {\n\t\tstop()\n\t}",
        repl: "\tif watch != nil && stop != nil && watch.Err() != nil {\n\t\tstop()\n\t}",
      },
    ],
  },
  // 预期「全绿」的等价变异：登记观察上限，不算杀红。
  {
    id: "e1",
    desc: "（预期全绿）终止节点把入参信封浅拷贝后返回——信封本体身份不在任何公共面上（Result.Output 只取 .ec）",
    edits: [
      {
        file: compilerGo,
        anchor:
          "\tcc.g.AddNode(graph.NodeFunc(endNodeName, func(ctx context.Context, in any) (any, error) {\n\t\treturn in, nil\n\t}), cc.traceOpt(\"\"))",
        repl:
          "\tcc.g.AddNode(graph.NodeFunc(endNodeName, func(ctx context.Context, in any) (any, error) {\n\t\tc := in.(*carrier)\n\t\treturn &carrier{ec: c.ec}, nil\n\t}), cc.traceOpt(\"\"))",
      },
    ],
  },
  {
    id: "e2",
    desc: "（预期全绿）主干步记账的去重守卫摘除——收集器每步恰完成一次，守卫是死分支",
    edits: [
      {
        file: compilerGo,
        anchor: "\t\tif spineID != \"\" && !l.recorded[spineID] {",
        repl: "\t\tif spineID != \"\" {",
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
    out = execFileSync("go", cmd.args, { cwd: repo, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"], timeout: 420000 });
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
    const killed = !scriptError && ((a && a.code !== 0) || (b && b.code !== 0));
    rows.push({ id: m.id, desc: m.desc, aExit: a?.code, bExit: b?.code, aFails: a?.fails, bFails: b?.fails, aBuild: a?.buildFailed, bBuild: b?.buildFailed, restored: drift.length === 0 && !scriptError, killed, scriptError });
    const r = rows.at(-1);
    console.log(
      [
        r.id,
        `exitA=${r.aExit}`,
        `exitB=${r.bExit}`,
        `restored=${r.restored}`,
        r.killed ? "KILLED" : "SURVIVED",
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
