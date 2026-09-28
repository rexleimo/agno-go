#!/usr/bin/env node
// 切片 29（P8 第 1 片：G8 Store 最小核，D2=OPT-a1 落地）的变异矩阵执行体。
// 延续 p5s28-sidecar.mjs 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `--- FAIL:` 名），
// 另加一类 judge:"grep" 的结构性变异（D8/D9 的「代码里建表 / import session 面」牙齿
// 不在 go test 判红面上，而在契约 grep 判据上）；⑪「给 Agent 接线 Store」不做文件变异——
// pkg/hno/agent 是兄弟片在途领地，写恢复循环可能踩掉并发编辑，改为常驻字节锚 + 零接线 grep 检查。
//
// 用法：
//   node scripts/mutation/p8g8-store.mjs            # 跑全部变异
//   node scripts/mutation/p8g8-store.mjs m1 m5      # 只跑指定的几条
//   node scripts/mutation/p8g8-store.mjs --check    # 只预检锚唯一性
import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";

const repo = "/Users/rex/codes/agno-go";
const itemGo = "pkg/hno/store/item.go";
const namespaceGo = "pkg/hno/store/namespace.go";
const memoryGo = "pkg/hno/store/memory.go";
const postgresGo = "pkg/hno/store/postgres/postgres.go";

// 命令 A = 契约绑定场景（TestP8G8_，-race -v）；命令 B = 邻近包零误伤面
//（Store 今日全仓零消费方，B 侧对每条变异都应保持绿）。
const CMD_A = {
  label: "bound TestP8G8_",
  args: ["test", "./pkg/hno/store/...", "-run", "TestP8G8_", "-count=1", "-race", "-v"],
};
const CMD_B = {
  label: "neighbors",
  args: [
    "test",
    "./pkg/hno/knowledge/...",
    "./pkg/hno/vectordb/...",
    "./pkg/hno/memory/...",
    "./pkg/hno/embeddings/...",
    "./pkg/hno/session/db/...",
    "./pkg/agentos/...",
    "-count=1",
  ],
};
// D10 结构判据（只读，跑两行）：12 个锚文件 + 存量包零接线 grep。
const ANCHORS = [
  "pkg/hno/agent/agent.go",
  "pkg/hno/agent/config.go",
  "pkg/hno/knowledge/chunker.go",
  "pkg/hno/knowledge/document.go",
  "pkg/hno/knowledge/knowledge_test.go",
  "pkg/hno/vectordb/base.go",
  "pkg/hno/memory/memory.go",
  "pkg/hno/session/session.go",
  "pkg/agentos/knowledge_handlers.go",
  "pkg/hno/session/db/postgres/storage.go",
  "internal/session/store/store.go",
  "pkg/hno/vectordb/chromadb/chromadb.go",
];

const MUTANTS = [
  {
    id: "m1",
    desc: "Key 裸用作身份（D4 破）：upsert 以裸 Key 为行身份，跨 namespace 同名 Key 后写覆盖（裁决材料 §1.4 [A4] 事故形态）",
    edits: [{ file: memoryGo, anchor: "\tid := Composite(in.Namespace, in.Key)", repl: "\tid := in.Key" }],
  },
  {
    id: "m2",
    desc: "namespace 用 '.' 压平进规范编码（D4 破）：Composite 退化成 naive join，[\"a.b\",\"c\"] 与 [\"a\",\"b.c\"] 撞同一串",
    edits: [
      {
        file: namespaceGo,
        anchor:
          "\tfor _, seg := range ns {\n\t\tb.WriteString(strconv.Itoa(len(seg)))\n\t\tb.WriteString(UnitSeparator)\n\t\tb.WriteString(seg)\n\t\tb.WriteString(UnitSeparator)\n\t}",
        repl: "\tfor _, seg := range ns {\n\t\tb.WriteString(seg)\n\t\tb.WriteString(\".\")\n\t}",
      },
    ],
  },
  {
    id: "m3",
    desc: "sql.ErrNoRows 裸传（D3/D8 破）：Get 未命中不再翻译为包内哨兵 ErrNotFound",
    edits: [
      {
        file: postgresGo,
        anchor:
          "\tif err := row.Scan(&value, &createdAt, &updatedAt); err != nil {\n\t\tif errors.Is(err, sql.ErrNoRows) {\n\t\t\treturn nil, store.ErrNotFound\n\t\t}\n\t\treturn nil, err\n\t}",
        repl: "\tif err := row.Scan(&value, &createdAt, &updatedAt); err != nil {\n\t\treturn nil, err\n\t}",
      },
      { file: postgresGo, anchor: '\t"errors"\n', repl: "" },
    ],
  },
  {
    id: "m4",
    desc: "Delete 未命中静默 no-op（D3 破）：不报 ErrNotFound（对齐仓库先例的语义被拆掉）",
    edits: [
      {
        file: memoryGo,
        anchor: "\tif _, ok := m.rows[id]; !ok {\n\t\treturn ErrNotFound\n\t}",
        repl: "\tif _, ok := m.rows[id]; !ok {\n\t\treturn nil\n\t}",
      },
    ],
  },
  {
    id: "m5",
    desc: "每次 Put 重打 CreatedAt（D5 破）：created_at 写一次的 upsert 语义被拆掉",
    edits: [
      {
        file: memoryGo,
        anchor: "\tif prev, ok := m.rows[id]; ok {\n\t\tin.CreatedAt = prev.item.CreatedAt\n\t\tin.UpdatedAt = now\n\t}",
        repl: "\tif _, ok := m.rows[id]; ok {\n\t\tin.CreatedAt = now\n\t\tin.UpdatedAt = now\n\t}",
      },
    ],
  },
  {
    id: "m6",
    desc: "读取期逐条重嵌 Value（D7 破）：Search 对每行再做一次嵌入调用，调用数从 200+10 变 200+10+10×行数",
    edits: [
      {
        file: memoryGo,
        anchor: "\t\thits = append(hits, SearchHit{Item: &cp, Score: cosine(qvec, s.vec)})",
        repl: "\t\trv, _ := m.emb.EmbedSingle(ctx, string(s.item.Value))\n\t\thits = append(hits, SearchHit{Item: &cp, Score: cosine(rv, rv)})",
      },
    ],
  },
  {
    id: "m7",
    desc: "把向量塞进 Item 字段（D1/D7 破）：母约草图五字段被加到六字段",
    edits: [
      {
        file: itemGo,
        anchor: "\tValue     []byte\n\tCreatedAt time.Time",
        repl: "\tValue     []byte\n\tEmbedding []float32\n\tCreatedAt time.Time",
      },
    ],
  },
  {
    id: "m8",
    desc: "nil 注入时 Search 静默返回空集（D2 fail-closed 破）：ErrNoEmbedder 哨兵被拆掉",
    edits: [
      {
        file: memoryGo,
        anchor: "\tif m.emb == nil {\n\t\treturn nil, ErrNoEmbedder\n\t}",
        repl: "\tif m.emb == nil {\n\t\treturn nil, nil\n\t}",
      },
    ],
  },
  {
    id: "m9",
    desc: "表名不走标识符校验裸拼（D8 破）：注入型非法 schema/table 直接进 SQL",
    edits: [
      {
        file: postgresGo,
        anchor: "\ttableName, err := buildQualifiedName(cfg.Schema, cfg.Table)\n\tif err != nil {\n\t\treturn nil, err\n\t}",
        repl: "\ttableName := fmt.Sprintf(\"\\\"%s\\\".\\\"%s\\\"\", cfg.Schema, cfg.Table)",
      },
    ],
  },
  {
    id: "m10",
    desc: "非语义打分冒充检索（D6/D7 破）：相似度恒 0（省掉嵌入打分的形态），排序退化为 Key 序",
    edits: [
      {
        file: memoryGo,
        anchor: "\t\thits = append(hits, SearchHit{Item: &cp, Score: cosine(qvec, s.vec)})",
        repl: "\t\thits = append(hits, SearchHit{Item: &cp, Score: 0})",
      },
    ],
  },
  {
    id: "m11",
    desc: "List 用 map 迭代序返回（D6 确定性破 + ADJ-3 顺序确定破）",
    edits: [
      {
        file: memoryGo,
        anchor: "\tsort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })\n",
        repl: "",
      },
    ],
  },
  {
    id: "m12",
    desc: "Delete 零行受影响不翻译（D8 破）：持久侧未命中静默成功，RowsAffected 守卫被拆掉",
    edits: [
      {
        file: postgresGo,
        anchor: "\trows, err := result.RowsAffected()\n\tif err == nil && rows == 0 {\n\t\treturn store.ErrNotFound\n\t}\n\treturn nil",
        repl: "\t_, _ = result.RowsAffected()\n\treturn nil",
      },
    ],
  },
  // 结构性变异：牙齿在契约 grep 判据上（go test 对此面天然全绿，判红走 grep 计数）。
  {
    id: "m13",
    desc: "代码里进 DDL / 预置原生向量列（D8/D9 破）：CREATE TABLE 与 vector( 字样进产品字节，D9 grep 计数 >0",
    judge: "grep",
    grepArgs: ["grep", "-rniE", "pgvector|CREATE EXTENSION|vector\\(|<=>|ivfflat|hnsw", "pkg/hno/store", "scripts/migrations", "deploy"],
    edits: [
      {
        file: postgresGo,
        anchor: "var identifierPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)",
        repl: "var identifierPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)\n\n// mutated: CREATE TABLE IF NOT EXISTS t (v vector(3))",
      },
    ],
  },
  {
    id: "m14",
    desc: "import 存量会话类型（D8/D11 破）：后端依赖图渗入 pkg/hno/session，D11 grep 计数 >0",
    judge: "grep",
    grepArgs: ["grep", "-rnE", "vectordb\\.(VectorDB|Document|SearchResult)|pkg/hno/knowledge|types\\.Message|pkg/hno/session|internal/session|pgx|BatchWriter", "pkg/hno/store"],
    edits: [
      {
        file: postgresGo,
        anchor: "\t\"github.com/rexleimo/agno-go/pkg/hno/store\"\n\t\"github.com/rexleimo/agno-go/pkg/hno/vectordb\"",
        repl: "\t\"github.com/rexleimo/agno-go/pkg/hno/session\"\n\t\"github.com/rexleimo/agno-go/pkg/hno/store\"\n\t\"github.com/rexleimo/agno-go/pkg/hno/vectordb\"",
      },
    ],
  },
  // 预期「全绿」的等价变异：登记观察上限，不算杀红。
  {
    id: "e1",
    desc: "（预期全绿）包内声明一个 unexported embedder 接口（死声明）——D11 白名单只看导出面，D2 判据锁的是「注入的抽象」而非包内死类型；结构性禁令由 D11 导出面测试与 grep 常驻",
    edits: [
      {
        file: memoryGo,
        anchor: "// stored 是内存后端的内部行：Item 与写入期算出的向量（物理列形态，不进 Item）。\ntype stored struct {",
        repl: "// mutated: dead unexported declaration\ntype deadEmbedder interface {\n\tEmbed(ctx context.Context, text string) ([]float32, error)\n}\n\n// stored 是内存后端的内部行：Item 与写入期算出的向量（物理列形态，不进 Item）。\ntype stored struct {",
      },
    ],
  },
  {
    id: "e2",
    desc: "（预期全绿）List 排序键从 Key 升序换成 UpdatedAt 降序——契约只断「全量 + 两次同输入序一致」，排序算法属 forbiddenObservations 的观察上限",
    edits: [
      {
        file: memoryGo,
        anchor: "\tsort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })",
        repl: "\tsort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })",
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

function runRaw(argv, cwd = repo) {
  try {
    execFileSync(argv[0], argv.slice(1), { cwd, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"], timeout: 120000 });
    return 0;
  } catch (err) {
    return typeof err.status === "number" ? err.status : -1;
  }
}

function structuralCheck() {
  const anchorExit = runRaw(["git", "hash-object", "--no-filters", ...ANCHORS]);
  const wiringExit = runRaw(["grep", "-rn", "hno/store", "pkg/hno/agent", "pkg/hno/session", "internal/session", "pkg/agentos", "--include=*.go"]);
  return { anchorExit, wiringExit };
}

function main() {
  const checkOnly = process.argv.includes("--check");
  const wanted = process.argv.slice(2).filter((a) => !a.startsWith("-"));
  const picked = wanted.length ? MUTANTS.filter((m) => wanted.includes(m.id)) : MUTANTS;
  const files = [...new Set(MUTANTS.flatMap((m) => m.edits.map((e) => e.file)))];

  for (const m of picked) {
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
  console.log(`structural(before) ${JSON.stringify(structuralCheck())}`);
  const rows = [];
  for (const m of picked) {
    let a = null;
    let b = null;
    let grepExit = null;
    let scriptError = null;
    try {
      const originals = new Map([[m.edits[0].file, readFileSync(`${repo}/${m.edits[0].file}`)]]);
      writeFileSync(`${repo}/${m.edits[0].file}`, apply(readFileSync(`${repo}/${m.edits[0].file}`, "utf8"), m.edits));
      a = run(CMD_A);
      b = run(CMD_B);
      if (m.judge === "grep") grepExit = runRaw(m.grepArgs);
    } catch (err) {
      scriptError = String(err.message ?? err);
    } finally {
      restore(baseline);
    }
    const drift = files.filter((f) => sha(readFileSync(`${repo}/${f}`)) !== sha(baseline.get(f)));
    // grep 退出码语义：0 = 有命中 = 结构牙齿击杀；1 = 无命中。go test 判红只认 `--- FAIL:` 名。
    const grepKilled = grepExit !== null && grepExit === 0;
    const goKilled = (a?.fails.length ?? 0) > 0;
    rows.push({
      id: m.id,
      desc: m.desc,
      aExit: a?.code,
      bExit: b?.code,
      aFails: a?.fails,
      bFails: b?.fails,
      aBuild: a?.buildFailed,
      bBuild: b?.buildFailed,
      grepExit,
      killed: goKilled || grepKilled,
      restored: drift.length === 0 && !scriptError,
      scriptError,
    });
    const r = rows.at(-1);
    console.log(
      [
        r.id,
        `exitA=${r.aExit}`,
        `exitB=${r.bExit}`,
        r.grepExit !== null ? `grepExit=${r.grepExit}` : "",
        `killed=${r.killed}`,
        `restored=${r.restored}`,
        r.aBuild || r.bBuild ? "BUILD_FAILED(不计杀红)" : "",
        r.scriptError ? `SCRIPT_ERROR=${r.scriptError}` : "",
        `A_FAIL=${(r.aFails ?? []).join(",") || "-"}`,
        `B_FAIL=${(r.bFails ?? []).join(",") || "-"}`,
      ]
        .filter(Boolean)
        .join(" | "),
    );
  }
  const finalDrift = files.filter((f) => sha(readFileSync(`${repo}/${f}`)) !== sha(baseline.get(f)));
  console.log(`structural(after) ${JSON.stringify(structuralCheck())}`);
  console.log(`finalCheck identical=${finalDrift.length === 0} files=${files.join(",")}`);
  if (finalDrift.length) process.exitCode = 2;
}

main();
